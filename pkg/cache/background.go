package cache

import (
	"sync"
	"time"
)

type backgroundEntry[V any] struct {
	value       V
	lastUpdated time.Time
}

type backgroundSetGetter[K comparable, V any] struct {
	data      map[K]*backgroundEntry[V]
	mu        sync.RWMutex
	entryPool sync.Pool
	ttl       time.Duration
}

func newBackgroundSetGetter[K comparable, V any](ttl time.Duration) *backgroundSetGetter[K, V] {
	return &backgroundSetGetter[K, V]{
		data: make(map[K]*backgroundEntry[V]),
		entryPool: sync.Pool{
			New: func() interface{} {
				return &backgroundEntry[V]{}
			},
		},
		ttl: ttl,
	}
}

func (s *backgroundSetGetter[K, V]) Set(k K, v V, updateTime time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if prev, ok := s.data[k]; ok {
		prev.lastUpdated = updateTime
		prev.value = v
		return
	}

	s.data[k] = s.entryPool.Get().(*backgroundEntry[V])
	s.data[k].lastUpdated = updateTime
	s.data[k].value = v
}

func (s *backgroundSetGetter[K, V]) Get(k K) (V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.data[k]
	return v.value, ok
}

func (s *backgroundSetGetter[K, V]) LastUpdated(key K) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return time.Time{}, false
	}

	return v.lastUpdated, true
}

func (s *backgroundSetGetter[K, V]) CleanUp() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cleaned := 0

	for key, entry := range s.data {
		if time.Since(entry.lastUpdated) > 10*s.ttl {
			cleaned++
			delete(s.data, key)
			s.entryPool.Put(entry)
		}
	}

	return cleaned
}

func (s *backgroundSetGetter[K, V]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.data)
}
