package scheduler

import "context"

type Dispatcher interface {
	Dispatch(ctx context.Context, task Task) error
}
