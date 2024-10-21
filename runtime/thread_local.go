package runtime

import "sync"

type ThreadLocal[T any] interface {
	Load() T
	Store(T)
	Ex() bool
	Remove()
}

type goroutineLocal[T any] struct {
	m    *sync.Map
	init func() T
}

func NewThreadLocal[T any](init func() T) ThreadLocal[T] {
	return &goroutineLocal[T]{
		m:    &sync.Map{},
		init: init,
	}
}

func (g goroutineLocal[T]) Load() T {
	key := Goid()
	value, ok := g.m.Load(key)
	if !ok && g.init != nil {
		value = g.init()
		g.m.Store(key, value)
	}
	return value.(T)
}

func (g goroutineLocal[T]) Store(value T) {
	g.m.Store(Goid(), value)
}

func (g goroutineLocal[T]) Ex() (ok bool) {
	_, ok = g.m.Load(Goid())
	return
}

func (g goroutineLocal[T]) Remove() {
	g.m.Delete(Goid())
}
