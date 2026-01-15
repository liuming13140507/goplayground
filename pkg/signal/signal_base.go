package signal

import (
	"context"
	"sync"
	"sync/atomic"
)

// keyedListener represents a listener with an optional identification key.
type keyedListener[T any] struct {
	key      string
	listener Listener[T]
}

// baseSignal implements the core management logic using a Copy-on-Write (COW) pattern.
// This allows Emit to be completely lock-free while ensuring thread-safety.
type baseSignal[T any] struct {
	mu           sync.Mutex                         // Protects write operations (Add/Remove/Reset)
	listenersPtr atomic.Pointer[[]keyedListener[T]] // Atomic pointer to the current snapshot of listeners
}

// newBaseSignal initializes a new baseSignal with an empty listener list.
func newBaseSignal[T any]() baseSignal[T] {
	s := baseSignal[T]{}
	empty := make([]keyedListener[T], 0)
	s.listenersPtr.Store(&empty)
	return s
}

// AddListener attaches a listener. If a key is provided and already exists, it replaces the listener.
func (s *baseSignal[T]) AddListener(listener Listener[T], keys ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var key string
	if len(keys) > 0 {
		key = keys[0]
	}

	// 1. Load the current list
	current := *s.listenersPtr.Load()

	// 2. Create a new list (Copy)
	next := make([]keyedListener[T], len(current))
	copy(next, current)

	// 3. Update or Append
	if key != "" {
		for i := range next {
			if next[i].key == key {
				next[i].listener = listener
				s.listenersPtr.Store(&next) // Atomic Update
				return
			}
		}
	}

	next = append(next, keyedListener[T]{
		key:      key,
		listener: listener,
	})

	// 4. Swap the pointer (Write)
	s.listenersPtr.Store(&next)
}

// RemoveListener detaches a listener by its key.
func (s *baseSignal[T]) RemoveListener(key string) {
	if key == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current := *s.listenersPtr.Load()
	next := make([]keyedListener[T], 0, len(current))

	found := false
	for _, l := range current {
		if l.key == key {
			found = true
			continue
		}
		next = append(next, l)
	}

	if found {
		s.listenersPtr.Store(&next)
	}
}

// Reset clears all listeners.
func (s *baseSignal[T]) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	empty := make([]keyedListener[T], 0)
	s.listenersPtr.Store(&empty)
}

// Len returns the current number of listeners.
func (s *baseSignal[T]) Len() int {
	return len(*s.listenersPtr.Load())
}

// IsEmpty returns true if there are no listeners.
func (s *baseSignal[T]) IsEmpty() bool {
	return s.Len() == 0
}

// getListeners returns a point-in-time snapshot of the listeners for Emit.
func (s *baseSignal[T]) getListeners() []keyedListener[T] {
	return *s.listenersPtr.Load()
}

// SyncSignal implements a synchronous Signal.
type SyncSignal[T any] struct {
	baseSignal[T]
}

// NewSync creates a new synchronous signal.
func NewSync[T any]() *SyncSignal[T] {
	return &SyncSignal[T]{
		baseSignal: newBaseSignal[T](),
	}
}

// Emit notifies all listeners synchronously.
// The lock-free read ensures that adding/removing listeners during Emit doesn't cause deadlocks or blocking.
func (s *SyncSignal[T]) Emit(ctx context.Context, data T) {
	listeners := s.getListeners()
	for _, l := range listeners {
		l.listener(ctx, data)
	}
}
