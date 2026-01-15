package signal

import "context"

// TODO:asaskevich/EventBus,maybe it is concurrent safe

// Listener defines the signature for signal observers.
type Listener[T any] func(ctx context.Context, data T)

// Signal defines the interface for an observable event.
type Signal[T any] interface {
	// Emit triggers the signal, notifying all listeners.
	Emit(ctx context.Context, data T)
	// AddListener attaches a listener to the signal. An optional key can be provided for later removal.
	AddListener(listener Listener[T], key ...string)
	// RemoveListener detaches a listener by its key.
	RemoveListener(key string)
	// Reset clears all listeners.
	Reset()
	// Len returns the number of listeners.
	Len() int
	// IsEmpty returns true if there are no listeners.
	IsEmpty() bool
}

// New creates a new synchronous signal.
func New[T any]() Signal[T] {
	return NewSync[T]()
}
