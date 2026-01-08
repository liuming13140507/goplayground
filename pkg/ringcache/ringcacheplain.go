package ringcache

import "sync"

type RingCache[T any] struct {
	items    []T
	capacity int
	head     int
	tail     int
	size     int
	mu       sync.Mutex
}

func NewRingCache[T any](capacity int) *RingCache[T] {
	return &RingCache[T]{
		items:    make([]T, capacity),
		capacity: capacity,
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

func (c *RingCache[T]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.size
}
