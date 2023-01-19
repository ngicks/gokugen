package scheduler

import (
	"context"
	"fmt"
)

type WorkFn = func(ctx context.Context, param []byte) error

type WorkRegistry interface {
	Load(workId string) (fn WorkFn, ok bool)
}

type ErrWorkIdNotFound struct {
	Param TaskParam
}

func (e *ErrWorkIdNotFound) Error() string {
	return fmt.Sprintf("work id not found. id = %s", e.Param.WorkId)
}
