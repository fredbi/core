package typeutils

import (
	"maps"
	"sync"
)

// MergeMaps merges maps into the target.
func MergeMaps[M ~map[K]V, K comparable, V any](target M, merged ...M) M {
	if target == nil {
		var c int
		for _, m := range merged {
			c += len(m)
		}
		target = make(map[K]V, c)
	}

	for _, m := range merged {
		maps.Copy(target, m)
	}

	return target
}

// SyncMap is a generic wrapper around [sync.Map].
type SyncMap[K comparable, V any] struct {
	m sync.Map
}

func (m *SyncMap[K, V]) Clear(key K) { m.m.Clear() }

func (m *SyncMap[K, V]) CompareAndDelete(key V, old V) (deleted bool) {
	return m.m.CompareAndDelete(key, old)
}

func (m *SyncMap[K, V]) CompareAndSwap(key V, old, new V) (swapped bool) {
	return m.m.CompareAndSwap(key, old, new)
}

func (m *SyncMap[K, V]) Delete(key K) { m.m.Delete(key) }

func (m *SyncMap[K, V]) Load(key K) (value V, ok bool) {
	v, ok := m.m.Load(key)
	if !ok {
		return value, ok
	}

	return v.(V), ok
}

func (m *SyncMap[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	v, loaded := m.m.LoadAndDelete(key)
	if !loaded {
		return value, loaded
	}

	return v.(V), loaded
}

func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	a, loaded := m.m.LoadOrStore(key, value)

	return a.(V), loaded
}

func (m *SyncMap[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(key, value any) bool { return f(key.(K), value.(V)) })
}

func (m *SyncMap[K, V]) Store(key K, value V) { m.m.Store(key, value) }

func (m *SyncMap[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	a, loaded := m.m.Swap(key, value)

	return a.(V), loaded
}
