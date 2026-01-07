package task

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
)

type Task func(ctx context.Context) error

type Job struct {
	ctx context.Context
	task Task
}

type Worker struct {
	ctx     context.Context
	cancel  context.CancelFunc
	queue   chan Job
	wg      sync.WaitGroup
	OnError func(error)
}

// NewWorker create and start a worker pool
// maxWorkers: number of concurrent goroutines
// maxQueue: queue buffer size
// onError: callback function when task failed or panic
func NewWorker(maxWorkers int, maxQueue int, onError func(error)) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	w := &Worker{
		ctx:     ctx,
		cancel:  cancel,
		queue:   make(chan Job, maxQueue),
		OnError: onError,
	}
	w.start(maxWorkers)
	return w
}

func (w *Worker) start(maxWorkers int) {
	for i := 0; i < maxWorkers; i++ {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			for {
				select {
				case <-w.ctx.Done():
					w.drain()
					return
				case t, ok := <-w.queue:
					if !ok {
						return
					}
					w.runTask(t)
				}
			}
		}()
	}
}

// drain try to process remaining tasks in the queue when exiting
func (w *Worker) drain() {
	for {
		select {
		case t, ok := <-w.queue:
			if !ok {
				return
			}
			w.runTask(t)
		default:
			return
		}
	}
}

func (w *Worker) runTask(job Job) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic: %v\nstack: %s", r, string(debug.Stack()))
			w.handleError(err)
		}
	}()

	if err := job.task(job.ctx); err != nil {
		w.handleError(err)
	}
}

func (w *Worker) handleError(err error) {
	if w.OnError != nil {
		w.OnError(err)
	} else {
		log.Printf("[Worker] error: %v", err)
	}
}

func (w *Worker) Submit(ctx context.Context, task Task) error {
	select {
	case <-w.ctx.Done():
		return fmt.Errorf("worker pool is stopped: %w", w.ctx.Err())
	case <-ctx.Done():
		return fmt.Errorf("job context is done: %w", ctx.Err())
	case w.queue <- Job{ctx: ctx, task: task}:
		return nil
	}
}

func (w *Worker) Stop() {
	w.cancel()
	// close(w.queue) // panic if the queue is not empty
	w.wg.Wait()
}


