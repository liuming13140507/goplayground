package _deprecated

import "sync"

type Cache[T any] struct {
	items    []T
	capacity int
	head     int
	tail     int
	size     int
	mu       sync.Mutex
}

func NewCache[T any](capacity int) *Cache[T] {
	return &Cache[T]{
		capacity: capacity,
		items:    make([]T, capacity),
	}
}

func (c *Cache[T]) Put(val T) (overwritten T, dropper bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.size == c.capacity {
		overwritten = c.items[c.head]
		dropper = true
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

func (c *Cache[T]) Get() (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.size == 0 {
		var zero T
		return zero, false
	}

	val := c.items[c.head]
	var zero T
	c.items[c.head] = zero
	c.head = (c.head + 1) % c.capacity
	c.size--
	return val, true
}

func (c *Cache[T]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.size
}
