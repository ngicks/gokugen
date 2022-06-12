package gokugen

import "fmt"

type workIdKeyTy string

func (s workIdKeyTy) String() string {
	return "workIdKeyTy"
}

var workIdKey *workIdKeyTy = new(workIdKeyTy)

func WithWorkId(parent SchedulerContext, workId string) SchedulerContext {
	return &workIdCtx{
		SchedulerContext: parent,
		workId:           workId,
	}
}

type workIdCtx struct {
	SchedulerContext
	workId string
}

func (ctx *workIdCtx) Value(key any) (any, error) {
	if key == workIdKey {
		return ctx.workId, nil
	}
	return ctx.SchedulerContext.Value(key)
}

func GetWorkId(ctx SchedulerContext) (string, error) {
	id, err := ctx.Value(workIdKey)
	if err != nil {
		return "", err
	}
	if id == nil {
		return "", fmt.Errorf("%w: key=%s", ErrValueNotFound, workIdKey)
	}
	return id.(string), nil
}
