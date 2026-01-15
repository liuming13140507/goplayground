package middleware

import (
	"context"
	"fmt"
	"runtime/debug"
)

// Handler represents the core logic function signature with generics.
// I: Input type, O: Output type.
type Handler[I, O any] func(ctx context.Context, req I) (O, error)

// Middleware is a decorator function that wraps a Handler using generics.
type Middleware[I, O any] func(next Handler[I, O]) Handler[I, O]

// Plugin represents a named middleware with additional metadata.
type Plugin[I, O any] struct {
	Name        string
	Description string
	Action      Middleware[I, O]
}

// Manager handles the registration and execution of a middleware chain with generics.
type Manager[I, O any] struct {
	plugins []Plugin[I, O]
}

// NewManager creates a new middleware manager for specific input and output types.
func NewManager[I, O any]() *Manager[I, O] {
	return &Manager[I, O]{
		plugins: make([]Plugin[I, O], 0),
	}
}

// Register adds one or more plugins to the chain.
func (m *Manager[I, O]) Register(plugins ...Plugin[I, O]) {
	m.plugins = append(m.plugins, plugins...)
}

// Build compiles the registered plugins into a single Handler.
func (m *Manager[I, O]) Build(finalHandler Handler[I, O]) Handler[I, O] {
	chain := finalHandler

	// chain = P2(H)
	// chain = P1(P2(H))
	for i := len(m.plugins) - 1; i >= 0; i-- {
		chain = m.plugins[i].Action(chain)
	}

	return chain
}

// Run executes the chain directly.
func (m *Manager[I, O]) Run(ctx context.Context, req I, finalHandler Handler[I, O]) (O, error) {
	return m.Build(finalHandler)(ctx, req)
}

// Recovery creates a middleware that recovers from panics.
func Recovery[I, O any]() Middleware[I, O] {
	return func(next Handler[I, O]) Handler[I, O] {
		return func(ctx context.Context, req I) (res O, err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic recovered: %v\n%s", r, string(debug.Stack()))
				}
			}()
			return next(ctx, req)
		}
	}
}
