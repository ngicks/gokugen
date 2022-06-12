package taskstorage

import (
	"errors"
	"fmt"

	"github.com/ngicks/gokugen"
)

type taskIdKeyTy string

func (s taskIdKeyTy) String() string {
	return "taskIdKeyTy"
}

type paramKeyTy string

func (s paramKeyTy) String() string {
	return "paramKeyTy"
}

var (
	taskIdKey *taskIdKeyTy = new(taskIdKeyTy)
	paramKey  *paramKeyTy  = new(paramKeyTy)
)

var (
	ErrOtherNodeWorkingOnTheTask = errors.New("other node is already working on the task")
)

func WithParam(parent gokugen.SchedulerContext, param any) gokugen.SchedulerContext {
	loader := paramLoadableCtx{
		SchedulerContext: &baseCtx{
			SchedulerContext: parent,
		},
		paramLoader: func() (any, error) { return param, nil },
	}

	return &loader
}

type baseCtx struct {
	gokugen.SchedulerContext
	taskId string
	work   WorkFn
}

func (ctx *baseCtx) Work() WorkFn {
	if ctx.work != nil {
		return ctx.work
	} else {
		return ctx.SchedulerContext.Work()
	}
}

func (ctx *baseCtx) Value(key any) (any, error) {
	if key == taskIdKey {
		return ctx.taskId, nil
	}
	return ctx.SchedulerContext.Value(key)
}

type fnWrapperCtx struct {
	gokugen.SchedulerContext
	wrapper func(self gokugen.SchedulerContext, workFn WorkFn) WorkFn
}

func (ctx *fnWrapperCtx) Work() WorkFn {
	return ctx.wrapper(ctx, ctx.SchedulerContext.Work())
}

type paramLoadableCtx struct {
	gokugen.SchedulerContext
	paramLoader func() (any, error)
}

func (ctx *paramLoadableCtx) Value(key any) (any, error) {
	if key == paramKey {
		return ctx.paramLoader()
	}
	return ctx.SchedulerContext.Value(key)
}

func (ctx *paramLoadableCtx) Unwrap() gokugen.SchedulerContext {
	return ctx.SchedulerContext
}

func isTaskStorageCtx(ctx gokugen.SchedulerContext) bool {
	if _, ok := ctx.(*baseCtx); ok {
		return true
	} else if _, ok := ctx.(*fnWrapperCtx); ok {
		return true
	} else if _, ok := ctx.(*paramLoadableCtx); ok {
		return true
	}
	return false
}

func GetTaskId(ctx gokugen.SchedulerContext) (string, error) {
	id, err := ctx.Value(taskIdKey)
	if err != nil {
		return "", err
	}
	if id == nil {
		return "", fmt.Errorf("%w: key=%s", gokugen.ErrValueNotFound, taskIdKey)
	}
	return id.(string), nil
}

func GetParam(ctx gokugen.SchedulerContext) (any, error) {
	return ctx.Value(paramKey)
}
