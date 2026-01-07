package task

import "context"

type taskPool interface {
	Submit(task func(ctx context.Context) error) error
	Stop()
}
