// Package pool provides a typed pool for objects that can reset their state.
package pool

import "sync"

// Pool stores reusable values of one type. Create it with New; do not copy it.
type Pool[T interface{ Reset() }] struct {
	objects sync.Pool
}

// New creates a pool that uses create when no reusable value is available.
func New[T interface{ Reset() }](create func() T) *Pool[T] {
	if create == nil {
		panic("pool: nil create function")
	}
	return &Pool[T]{objects: sync.Pool{New: func() any { return create() }}}
}

// Get returns a reusable value or creates a new one.
func (pool *Pool[T]) Get() T {
	return pool.objects.Get().(T)
}

// Put resets value and makes it available for reuse.
func (pool *Pool[T]) Put(value T) {
	value.Reset()
	pool.objects.Put(value)
}
