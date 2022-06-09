package taskstorage

import (
	"errors"
	"fmt"

	"github.com/ngicks/gokugen"
)

type taskIdKeyTy string
type workIdKeyTy string
type paramKeyTy string

var (
	taskIdKey *taskIdKeyTy = new(taskIdKeyTy)
	workIdKey *workIdKeyTy = new(workIdKeyTy)
	paramKey  *paramKeyTy  = new(paramKeyTy)
)

var (
	ErrOtherNodeWorkingOnTheTask = errors.New("other node is already working on the task")
)

func WithWorkIdAndParam(parent gokugen.SchedulerContext, workId string, param any) gokugen.SchedulerContext {
	base := baseCtx{
		SchedulerContext: parent,
		workId:           workId,
	}

	loader := paramLoadableCtx{
		SchedulerContext: &base,
		paramLoader:      func() (any, error) { return param, nil },
	}

	return &loader
}

type baseCtx struct {
	gokugen.SchedulerContext
	taskId string
	workId string
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
	switch key {
	case taskIdKey:
		return ctx.taskId, nil
	case workIdKey:
		return ctx.workId, nil
	}
	return ctx.SchedulerContext.Value(key)
}

func (ctx *baseCtx) Unwrap() gokugen.SchedulerContext {
	return ctx.SchedulerContext
}

type fnWrapperCtx struct {
	gokugen.SchedulerContext
	wrapper func(self gokugen.SchedulerContext, workFn WorkFn) WorkFn
}

func (ctx *fnWrapperCtx) Work() WorkFn {
	return ctx.wrapper(ctx, ctx.SchedulerContext.Work())
}

func (ctx *fnWrapperCtx) Unwrap() gokugen.SchedulerContext {
	return ctx.SchedulerContext
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
		return "", fmt.Errorf("%w: key=%v", gokugen.ErrValueNotFound, workIdKey)
	}
	return id.(string), nil
}

func GetWorkId(ctx gokugen.SchedulerContext) (string, error) {
	id, err := ctx.Value(workIdKey)
	if err != nil {
		return "", err
	}
	if id == nil {
		return "", fmt.Errorf("%w: key=%v", gokugen.ErrValueNotFound, workIdKey)
	}
	return id.(string), nil
}

func GetParam(ctx gokugen.SchedulerContext) (any, error) {
	return ctx.Value(paramKey)
}
