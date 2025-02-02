package logger

import (
	"fmt"
	"github.com/mattn/go-isatty"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/url"
	"os"
	"syscall"
)

const (
	stdoutTransport        = "stdout"
	fileTransport          = "file"
	stdoutAndFileTransport = "stdout+file"
)

type transport struct {
	core zapcore.Core
	stop func()
}

func getStdoutTransport(info *loggerInfo) *transport {
	res := &transport{
		stop: func() {},
	}
	sink := zapcore.AddSync(os.Stdout)

	var encoder zapcore.Encoder
	if info.cfg.DevMode {
		if isatty.IsTerminal(os.Stdout.Fd()) {
			info.encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
		encoder = zapcore.NewConsoleEncoder(info.encCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(info.encCfg)
	}

	res.core = zapcore.NewCore(encoder, sink, info.lvl)

	return res
}

func getFileTransport(info *loggerInfo) (*transport, error) {
	res := &transport{}

	if info.cfg.FilePath == "" {
		return nil, fmt.Errorf("no file path specified")
	}

	u := &url.URL{
		Path: info.cfg.FilePath,
	}

	sink, err := NewLogrotateSink(u, syscall.SIGUSR1)
	if err != nil {
		return nil, fmt.Errorf("failed to open logrotate sink: %w", err)
	}

	fileCore := zapcore.NewCore(zapcore.NewJSONEncoder(info.encCfg), sink, info.lvl)

	res.core = zapcore.NewTee(fileCore)
	res.stop = func() {
		if err := sink.Close(); err != nil {
			fallbackLogger.Error("failed to close sink", zap.Error(err))
		}
	}

	return res, nil
}
