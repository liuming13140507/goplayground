package mpmc

import (
	"runtime"
	"sync/atomic"
)

type slot struct {
	data any
	seq  uint64
}

type RingCache struct {
	_     [8]uint64 //False Sharing
	items []slot
	size  uint64
	mask  uint64
	_     [8]uint64
	head  uint64
	_     [8]uint64
	tail  uint64
	_     [8]uint64
}

func NewRingCache(size uint64) *RingCache {
	if size&(size-1) != 0 {
		panic("size must be power of 2")
	}
	c := &RingCache{
		items: make([]slot, size),
		size:  size,
		mask:  size - 1,
	}
	for i := uint64(0); i < size; i++ {
		c.items[i].seq = i
	}
	return c
}

func (c *RingCache) Put(val any) bool {
	var s *slot
	pos := atomic.LoadUint64(&c.tail)

	for {
		s = &c.items[pos&c.mask]
		seq := atomic.LoadUint64(&s.seq)
		diff := int64(seq) - int64(pos)

		if diff == 0 {
			// 槽位正好准备好被写入
			if atomic.CompareAndSwapUint64(&c.tail, pos, pos+1) {
				break
			}
		} else if diff < 0 {
			// 缓冲区已满
			return false
		} else {
			// 其他生产者正在操作，重新获取最新的 tail
			pos = atomic.LoadUint64(&c.tail)
		}
		runtime.Gosched() // 让出 CPU，避免死循环消耗
	}

	s.data = val
	// 将 seq 置为 pos+1，标志着数据已就绪，可以被消费者读取
	atomic.StoreUint64(&s.seq, pos+1)
	return true
}

func (c *RingCache) Get() (any, bool) {
	var s *slot
	pos := atomic.LoadUint64(&c.head)

	for {
		s = &c.items[pos&c.mask]
		seq := atomic.LoadUint64(&s.seq)
		diff := int64(seq) - int64(pos+1)

		if diff == 0 {
			// 数据已准备好被读取
			if atomic.CompareAndSwapUint64(&c.head, pos, pos+1) {
				break
			}
		} else if diff < 0 {
			// 队列为空
			return nil, false
		} else {
			// 其他消费者正在操作
			pos = atomic.LoadUint64(&c.head)
		}
		runtime.Gosched()
	}

	val := s.data
	s.data = nil // 避免内存泄露
	// 将 seq 置为 pos + size，标志着槽位已空，可以被重新写入
	atomic.StoreUint64(&s.seq, pos+c.size)
	return val, true
}
