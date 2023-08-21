package def

import (
	"context"
)

type WorkFn = func(ctx context.Context, param map[string]string) error

type WorkRegistry interface {
	Load(workId string) (fn *WorkFn, ok bool)
}
