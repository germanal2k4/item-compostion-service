package cache

import (
	"container/list"
	"sync"
	"time"
)

type lruEntry[K comparable, V any] struct {
	key         K
	value       V
	lastUpdated time.Time
}

type LruSetGetter[K comparable, V any] struct {
	data      map[K]*list.Element
	list      *list.List
	mu        sync.RWMutex
	entryPool sync.Pool
	ttl       time.Duration
	capacity  int
}

func NewLruSetGetter[K comparable, V any](capacity int, ttl time.Duration) *LruSetGetter[K, V] {
	return &LruSetGetter[K, V]{
		data: make(map[K]*list.Element, capacity),
		list: list.New(),
		entryPool: sync.Pool{
			New: func() interface{} {
				return &lruEntry[K, V]{}
			},
		},
		ttl:      ttl,
		capacity: capacity,
	}
}

func (l *LruSetGetter[K, V]) Set(k K, v V, updateTime time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.capacity == 0 {
		return
	}

	if elem, ok := l.data[k]; ok {
		l.list.MoveToBack(elem)
		entry := elem.Value.(*lruEntry[K, V])
		entry.value = v
		entry.lastUpdated = updateTime
		return
	}

	if l.list.Len() >= l.capacity {
		oldest := l.list.Front()
		if oldest != nil {
			l.removeElement(oldest)
		}
	}

	entry := l.entryPool.Get().(*lruEntry[K, V])
	entry.key = k
	entry.value = v
	entry.lastUpdated = updateTime

	elem := l.list.PushBack(entry)
	l.data[k] = elem
}

func (l *LruSetGetter[K, V]) Get(k K) (V, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var res V

	elem, ok := l.data[k]
	if ok && time.Since(elem.Value.(*lruEntry[K, V]).lastUpdated) < l.ttl {
		l.removeElement(elem)
		return res, false
	}

	if ok {
		l.list.MoveToBack(elem)
		return elem.Value.(*lruEntry[K, V]).value, true
	}

	return res, false
}

func (l *LruSetGetter[K, V]) LastUpdated(key K) (time.Time, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	elem, ok := l.data[key]
	if ok {
		return elem.Value.(*lruEntry[K, V]).lastUpdated, true
	}

	return time.Time{}, false
}

func (l *LruSetGetter[K, V]) CleanUp() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	cleaned := 0

	for _, elem := range l.data {
		entry := elem.Value.(*lruEntry[K, V])
		if time.Since(entry.lastUpdated) > l.ttl {
			cleaned++
			l.removeElement(elem)
		}
	}

	return cleaned
}

func (l *LruSetGetter[K, V]) removeElement(elem *list.Element) {
	entry := elem.Value.(*lruEntry[K, V])
	delete(l.data, entry.key)
	l.entryPool.Put(entry)
	l.list.Remove(elem)
}
