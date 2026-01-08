package ratelimit

import (
	"sync"
	"time"
)

type RateLimit struct {
	tokens   int64
	capacity int64
	rate     int64 // add tokens per nanosecond
	lastTime time.Time
	mtx      sync.Mutex
}

func NewRateLimit(rate int64, capacity int64) *RateLimit {
	return &RateLimit{
		tokens:   capacity,
		capacity: capacity,
		rate:     rate,
		lastTime: time.Now(),
	}
}

func (r *RateLimit) GetToken() bool {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	now := time.Now()
	delta := time.Since(now).Nanoseconds() * r.rate
	if delta > r.capacity {
		r.tokens = r.capacity
	} else {
		r.tokens += delta
	}
	r.lastTime = now

	if r.tokens < 1 {
		return false
	}

	r.tokens--
	return true
}
