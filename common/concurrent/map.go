package concurrent

import (
	"sync"
)

// 泛型加持的 Map

type Map[K comparable, V any] struct {
	*sync.Map
}

func (c *Map[K, V]) SyncMap() *sync.Map {
	return c.Map
}

func (c *Map[K, V]) Get(key K) V {
	ret, _ := c.Load(key)
	return ret
}

func (c *Map[K, V]) Load(key K) (V, bool) {
	ret, ok := c.Map.Load(key)
	if ok {
		return ret.(V), ok
	}
	var val V
	return val, ok
}

func (c *Map[K, V]) Store(key K, val V) {
	c.Map.Store(key, val)
}

func (c *Map[K, V]) LoadOrStore(key K, val V) (V, bool) {
	actual, loaded := c.Map.LoadOrStore(key, val)
	if loaded {
		return actual.(V), loaded
	}
	return val, loaded
}

func (c *Map[K, V]) LoadAndDelete(key K) (V, bool) {
	ret, ok := c.Map.LoadAndDelete(key)
	if ok {
		return ret.(V), ok
	}
	var val V
	return val, ok
}

func (c *Map[K, V]) Delete(key K) {
	c.Map.Delete(key)
}

func (c *Map[K, V]) Swap(key K, val V) (V, bool) {
	previous, loaded := c.Map.Swap(key, val)
	if loaded {
		return previous.(V), loaded
	}
	var actualVal V
	return actualVal, loaded
}

func (c *Map[K, V]) CompareAndSwap(key K, old, new V) bool {
	return c.Map.CompareAndSwap(key, old, new)
}

func (c *Map[K, V]) CompareAndDelete(key K, old V) bool {
	return c.Map.CompareAndDelete(key, old)
}

func (c *Map[K, V]) Range(f func(key K, value V) bool) {
	c.Map.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{&sync.Map{}}
}

func NewMapFromMap[K comparable, V any](m map[K]V) *Map[K, V] {
	ret := &Map[K, V]{&sync.Map{}}
	for k, v := range m {
		ret.Store(k, v)
	}
	return ret
}
