package spsc

import (
	"goplayground/pkg/ringcache"
	"sync/atomic"
)

type slot struct {
	val any
	seq uint64
}

type RingCache struct {
	items []slot
	head  uint64
	tail  uint64
	size  uint64
	mask  uint64
}

func NewRingCache(size uint64) *RingCache {
	size = ringcache.NextPowerOf2(size)
	return &RingCache{
		items: make([]slot, size),
		size:  size,
		mask:  size - 1,
	}
}

func (c *RingCache) Put(val any) bool {
	head := atomic.LoadUint64(&c.head)
	tail := atomic.LoadUint64(&c.tail)

	if tail-head >= c.size {
		atomic.AddUint64(&c.head, 1)
	}
	s := &c.items[tail&c.mask]
	atomic.AddUint64(&s.seq, 1)
	s.val = val

	// c.items[tail&c.mask] = val
	atomic.AddUint64(&s.seq, 1)
	atomic.AddUint64(&c.tail, 1)
	return true

}

func (c *RingCache) Get() (any, bool) {
	for {
		head := atomic.LoadUint64(&c.head)
		tail := atomic.LoadUint64(&c.tail)
		if tail == head {
			return nil, false
		}
		s := &c.items[head&c.mask]
		seq1 := atomic.LoadUint64(&s.seq)
		if seq1>>1 == 0 {
			continue
		}

		val := s.val

		seq2 := atomic.LoadUint64(&s.seq)
		if seq2 != seq1 {
			continue
		}

		if atomic.CompareAndSwapUint64(&c.head, head, head+1) {
			return val, true
		}
	}
}
