package sync

import "sync"

// Map is a generic wrapper around [sync.Map].
type Map[K comparable, V any] struct {
	m sync.Map
}

func (m *Map[K, V]) Delete(key K) { m.m.Delete(key) }

func (m *Map[K, V]) Load(key K) (value V, ok bool) {
	v, ok := m.m.Load(key)
	if !ok {
		return value, ok
	}

	return v.(V), ok
}

func (m *Map[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	v, loaded := m.m.LoadAndDelete(key)
	if !loaded {
		return value, loaded
	}

	return v.(V), loaded
}

func (m *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	a, loaded := m.m.LoadOrStore(key, value)

	return a.(V), loaded
}

func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(key, value any) bool { return f(key.(K), value.(V)) })
}

func (m *Map[K, V]) Store(key K, value V) { m.m.Store(key, value) }

func (m *Map[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	a, loaded := m.m.Swap(key, value)

	return a.(V), loaded
}

func (m *Map[K, V]) CompareAndSwap(key V, old, new V) (swapped bool) {
	return m.m.CompareAndSwap(key, old, new)
}

func (m *Map[K, V]) CompareAndDelete(key V, old V) (deleted bool) {
	return m.m.CompareAndDelete(key, old)
}
