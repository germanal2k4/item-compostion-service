package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
	"sync"
	"time"
)

type fallbackLogger struct {
	*zap.SugaredLogger
}

func (logger *fallbackLogger) Printf(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

type ElasticSink struct {
	client      *elastic.Client
	bulk        *elastic.BulkService
	cfg         *config
	fallbackLgr *fallbackLogger

	entries  chan []byte
	done     chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
	stopOnce sync.Once
}

func NewElasticSink(fallback *zap.SugaredLogger, opts ...Option) (*ElasticSink, error) {
	cfg := &config{}

	for _, opt := range opts {
		opt(cfg)
	}

	fallbackLgr := &fallbackLogger{fallback}

	client, err := elastic.NewClient(
		elastic.SetURL(cfg.Url),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(true),
		elastic.SetErrorLog(fallbackLgr),
	)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _, err = client.Ping(cfg.Url).Do(ctx)
	if err != nil {
		return nil, err
	}

	ews := &ElasticSink{
		client:      client,
		bulk:        client.Bulk().Index(cfg.Index),
		cfg:         cfg,
		fallbackLgr: fallbackLgr,
		entries:     make(chan []byte, cfg.WriteBufferSize),
		done:        make(chan struct{}),
	}

	ews.wg.Add(1)
	go ews.process()

	return ews, nil
}

func (ews *ElasticSink) Write(p []byte) (int, error) {
	select {
	case ews.entries <- bytes.Clone(p):
		if len(ews.entries) == cap(ews.entries) {
			ews.fallbackLgr.Warn("Log buffer for elasticsearch overflow")
		}
		return len(p), nil
	case <-ews.done:
		return 0, fmt.Errorf("writer is closed")
	}
}

func (ews *ElasticSink) Sync() error {
	select {
	case <-ews.done:
		return fmt.Errorf("writer is closed")
	default:
	}

	ews.flush(ews.bulk)
	return nil
}

func (ews *ElasticSink) Close() error {
	ews.stopOnce.Do(func() {
		close(ews.done)
		ews.wg.Wait()
		close(ews.entries)
	})
	return nil
}

func (ews *ElasticSink) process() {
	defer ews.wg.Done()

	ticker := time.NewTicker(ews.cfg.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case entry, ok := <-ews.entries:
			if !ok {
				ews.flush(ews.bulk)
				return
			}

			doc, err := ews.parseEntry(entry)
			if err != nil {
				continue
			}

			ews.bulk.Add(elastic.NewBulkIndexRequest().Doc(doc))

			if ews.bulk.NumberOfActions() >= bulkSize {
				ews.flush(ews.bulk)
			}

		case <-ticker.C:
			if ews.bulk.NumberOfActions() > 0 {
				ews.flush(ews.bulk)
			}

		case <-ews.done:
			for len(ews.entries) > 0 {
				entry := <-ews.entries
				doc, err := ews.parseEntry(entry)
				if err != nil {
					continue
				}
				ews.bulk.Add(elastic.NewBulkIndexRequest().Doc(doc))
			}
			ews.flush(ews.bulk)
			return
		}
	}
}

func (ews *ElasticSink) parseEntry(entry []byte) (map[string]interface{}, error) {
	var doc map[string]interface{}
	if err := json.Unmarshal(entry, &doc); err != nil {
		return nil, err
	}

	if _, exists := doc["@timestamp"]; !exists {
		doc["@timestamp"] = time.Now().UTC().Format(time.RFC3339)
	}

	return doc, nil
}

func (ews *ElasticSink) flush(bulk *elastic.BulkService) {
	ews.mu.Lock()
	defer ews.mu.Unlock()

	if bulk.NumberOfActions() == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := bulk.Do(ctx)
	if err != nil {
		ews.fallbackLgr.Errorw("ElasticSink flush error", "err", err)
		return
	}

	if res.Errors {
		for _, item := range res.Failed() {
			ews.fallbackLgr.Errorf("Failed to index document: %v\n", item.Error)
		}
	}
}
