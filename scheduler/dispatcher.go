package scheduler

import "context"

type Dispatcher interface {
	Dispatch(ctx context.Context, fetcher func() (Task, error)) (<-chan error, error)
}
