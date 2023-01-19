package scheduler

import "context"

type Dispatcher interface {
	Dispatch(ctx context.Context, fetcher func(ctx context.Context) (Task, error)) (<-chan error, error)
}
