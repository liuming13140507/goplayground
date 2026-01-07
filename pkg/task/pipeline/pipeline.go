package pipeline

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
)

// Pipeline defines a generic pipeline
// T is the output data type of the current stage
type Pipeline[T any] struct {
	ctx    context.Context
	source <-chan T
	errCh  chan error
	wg     *sync.WaitGroup
}

// Start begins a pipeline from a data source
func Start[T any](ctx context.Context, source <-chan T) *Pipeline[T] {
	return &Pipeline[T]{
		ctx:    ctx,
		source: source,
		errCh:  make(chan error, 1),
		wg:     &sync.WaitGroup{},
	}
}

// Next connects the current stage to the next processing stage
// workers: number of concurrent goroutines to process the current stage
// buffer: size of the output channel buffer
// transform: processing logic, input In, output Out
func Next[In any, Out any](p *Pipeline[In], workers int, buffer int, transform func(context.Context, In) (Out, error)) *Pipeline[Out] {
	outCh := make(chan Out, buffer)
	nextWg := &sync.WaitGroup{}

	nextWg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer nextWg.Done()
			defer func() {
				if r := recover(); r != nil {
					select {
					case p.errCh <- fmt.Errorf("pipeline panic: %v\nstack: %s", r, debug.Stack()):
					default:
					}
				}
			}()

			for {
				select {
				case <-p.ctx.Done():
					return
				case val, ok := <-p.source:
					if !ok {
						return
					}

					res, err := transform(p.ctx, val)
					if err != nil {
						select {
						case p.errCh <- err:
						default:
						}
						return
					}

					select {
					case <-p.ctx.Done():
						return
					case outCh <- res:
					}
				}
			}
		}()
	}

	// Cascaded wait: start a goroutine to close the channel after this stage is complete
	go func() {
		nextWg.Wait()
		close(outCh)
	}()

	return &Pipeline[Out]{
		ctx:    p.ctx,
		source: outCh,
		errCh:  p.errCh,
		wg:     nextWg,
	}
}

// Wait terminates the pipeline and waits for all stages to complete or fail
// handler: function to process the final output result
func (p *Pipeline[T]) Wait(handler func(T) error) error {
	for {
		select {
		case <-p.ctx.Done():
			return p.ctx.Err()
		case err := <-p.errCh:
			return err
		case val, ok := <-p.source:
			if !ok {
				return nil
			}
			if err := handler(val); err != nil {
				return err
			}
		}
	}
}

/* 
Usage Example:

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

source := make(chan int)
go func() {
    for i := 0; i < 100; i++ { source <- i }
    close(source)
}()

err := pipeline.Start(ctx, source).
    Next(p, 5, 10, func(ctx context.Context, i int) (string, error) {
        return strconv.Itoa(i), nil // Stage 1: format
    }).
    Wait(func(s string) error {
        fmt.Println("Final result:", s) // Endpoint: consume result
        return nil
    })
*/
