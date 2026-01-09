package native

import (
	"goplayground/pkg/ringcache"
	"sync"
)

type RingCache[T any] struct {
	items    []T
	capacity uint64
	head     uint64
	tail     uint64
	size     uint64
	mu       sync.Mutex
	mask     uint64
}

func NewRingCache[T any](capacity uint64) *RingCache[T] {
	capacity = ringcache.NextPowerOf2(capacity)
	return &RingCache[T]{
		items:    make([]T, capacity),
		capacity: capacity,
		mask:     capacity - 1,
	}
}

func (c *RingCache[T]) Put(val T) (overWritten T, drop bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.size >= c.capacity {
		drop = true
		overWritten = c.items[c.head]
		c.items[c.head] = val
		c.head = (c.head + 1) % c.capacity
		c.tail = (c.tail + 1) % c.capacity
		return
	}
	c.items[c.tail] = val
	c.tail = (c.tail + 1) % c.capacity
	c.size++
	return
}

func (c *RingCache[T]) Get() (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.size == 0 {
		var zero T
		return zero, false
	}

	res := c.items[c.head]
	var zero T
	c.items[c.head] = zero
	c.head = (c.head + 1) % c.capacity
	c.size--
	return res, true
}

func (c *RingCache[T]) Len() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.size
}
