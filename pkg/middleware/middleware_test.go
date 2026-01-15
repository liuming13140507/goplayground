package middleware

import (
	"context"
	"fmt"
	"log"
	"testing"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

func TestMiddleware(t *testing.T) {
	// Define types: Input is string, Output is string
	manager := NewManager[string, string]()

	// 1. Add Recovery Middleware (Best practice: first one to catch all panics)
	manager.Register(Plugin[string, string]{
		Name:   "Recovery",
		Action: Recovery[string, string](),
	})

	// 2. Add Logging Middleware with Context Value support
	manager.Register(Plugin[string, string]{
		Name: "Logger",
		Action: func(next Handler[string, string]) Handler[string, string] {
			return func(ctx context.Context, req string) (string, error) {
				userID := ctx.Value(userIDKey)
				log.Printf("[Logger] UserID: %v, Before: req=%v", userID, req)
				res, err := next(ctx, req)
				log.Printf("[Logger] After: res=%v, err=%v", res, err)
				return res, err
			}
		},
	})

	// 3. Add a Middleware that modifies context
	manager.Register(Plugin[string, string]{
		Name: "AuthInject",
		Action: func(next Handler[string, string]) Handler[string, string] {
			return func(ctx context.Context, req string) (string, error) {
				// Inject a user ID into context for downstream middlewares/handlers
				newCtx := context.WithValue(ctx, userIDKey, "user_123")
				return next(newCtx, req)
			}
		},
	})

	// 4. Define the final business logic
	finalHandler := func(ctx context.Context, req string) (string, error) {
		if req == "panic" {
			panic("something went wrong!")
		}
		log.Printf("[Business] Processing: %v", req)
		return fmt.Sprintf("Processed: %v", req), nil
	}

	ctx := context.Background()

	t.Run("Successful request with Context", func(t *testing.T) {
		res, err := manager.Run(ctx, "hello world", finalHandler)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		log.Printf("Final Result: %v", res)
	})

	t.Run("Panic Recovery Test", func(t *testing.T) {
		res, err := manager.Run(ctx, "panic", finalHandler)
		if err == nil {
			t.Errorf("expected error from panic recovery, got nil")
		} else {
			log.Printf("Recovered from panic as expected: %v", err)
		}
		if res != "" {
			t.Errorf("expected empty result on panic, got %v", res)
		}
	})
}
