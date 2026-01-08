package pipeline

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
)

type Pipeline[T any] struct {
	ctx     context.Context
	source  chan T
	errChan chan error
	wg      *sync.WaitGroup
}

func Start[T any](ctx context.Context, source chan T) *Pipeline[T] {
	return &Pipeline[T]{
		ctx:     ctx,
		source:  source,
		errChan: make(chan error, 1),
		wg:      &sync.WaitGroup{},
	}
}

func Next[In any, Out any](p *Pipeline[In], maxWorkers int, buffer int, transform func(context.Context, In) (Out, error)) *Pipeline[Out] {
	nxWg := &sync.WaitGroup{}
	nxWg.Add(maxWorkers)
	outCh := make(chan Out, buffer)
	for i := 0; i < maxWorkers; i++ {
		go func() {
			defer nxWg.Done()
			defer func() {
				if r := recover(); r != nil {
					select {
					case p.errChan <- fmt.Errorf("%v, track:%v", r, debug.Stack()):
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
						case p.errChan <- err:
						default:
						}
					}
					outCh <- res
				}
			}
		}()
	}

	go func() {
		nxWg.Wait()
		close(outCh)
	}()

	return &Pipeline[Out]{
		ctx:     p.ctx,
		source:  outCh,
		errChan: p.errChan,
		wg:      nxWg,
	}
}

func (p *Pipeline[T]) Wait(handler func(T) error) error {
	for {
		select {
		case <-p.ctx.Done():
			return nil
		case err := <-p.errChan:
			return err
		case res, ok := <-p.source:
			if !ok {
				return nil
			}
			if err := handler(res); err != nil {
				return err
			}
		}
	}
}
