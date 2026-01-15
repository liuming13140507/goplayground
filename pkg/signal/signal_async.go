package signal

import (
	"context"
)

// AsyncSignal implements an asynchronous Signal.
type AsyncSignal[T any] struct {
	baseSignal[T]
}

// NewAsync creates a new asynchronous signal.
func NewAsync[T any]() *AsyncSignal[T] {
	return &AsyncSignal[T]{
		baseSignal: newBaseSignal[T](),
	}
}

// Emit notifies all listeners asynchronously.
func (s *AsyncSignal[T]) Emit(ctx context.Context, data T) {
	listeners := s.getListeners()
	for _, l := range listeners {
		go l.listener(ctx, data)
	}
}
