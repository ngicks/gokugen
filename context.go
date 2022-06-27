package gokugen

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrValueNotFound = errors.New("value not found")
)

type WorkFn = func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) (any, error)
type WorkFnWParam = func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time, param any) (any, error)

// SchedulerContext is minimal set of data relevant to scheduling and middlewares.
type SchedulerContext interface {
	ScheduledTime() time.Time
	Work() WorkFn
	Value(key any) (any, error)
}

type (
	paramKeyTy  string
	taskIdKeyTy string
	workIdKeyTy string
)

func (s paramKeyTy) String() string {
	return "paramKeyTy"
}
func (s taskIdKeyTy) String() string {
	return "taskIdKeyTy"
}
func (s workIdKeyTy) String() string {
	return "workIdKeyTy"
}

var (
	paramKey  *paramKeyTy  = new(paramKeyTy)
	taskIdKey *taskIdKeyTy = new(taskIdKeyTy)
	workIdKey *workIdKeyTy = new(workIdKeyTy)
)

type PlainContext struct {
	scheduledTime time.Time
	workFn        WorkFn
	values        map[any]any
}

func NewPlainContext(scheduledTime time.Time, workFn WorkFn, values map[any]any) SchedulerContext {
	return &PlainContext{
		scheduledTime: scheduledTime,
		workFn:        workFn,
		values:        values,
	}
}

func (ctx *PlainContext) ScheduledTime() time.Time {
	return ctx.scheduledTime
}
func (ctx *PlainContext) Work() WorkFn {
	return ctx.workFn
}
func (ctx *PlainContext) Value(key any) (any, error) {
	if ctx.values == nil {
		return nil, nil
	}
	return ctx.values[key], nil
}

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

func WithParam(parent SchedulerContext, param any) SchedulerContext {
	return &paramLoadableCtx{
		SchedulerContext: parent,
		paramLoader:      func() (any, error) { return param, nil },
	}
}

func WithParamLoader(parent SchedulerContext, loader func() (any, error)) SchedulerContext {
	return &paramLoadableCtx{
		SchedulerContext: parent,
		paramLoader:      loader,
	}
}

type paramLoadableCtx struct {
	SchedulerContext
	paramLoader func() (any, error)
}

func (ctx *paramLoadableCtx) Value(key any) (any, error) {
	if key == paramKey {
		return ctx.paramLoader()
	}
	return ctx.SchedulerContext.Value(key)
}

func WithWorkFn(parent SchedulerContext, workFn WorkFn) SchedulerContext {
	return &fnWrapperCtx{
		SchedulerContext: parent,
		wrapper: func(self SchedulerContext, _ WorkFn) WorkFn {
			return workFn
		},
	}
}

type WorkFnWrapper = func(self SchedulerContext, workFn WorkFn) WorkFn

func WithWorkFnWrapper(parent SchedulerContext, wrapper WorkFnWrapper) SchedulerContext {
	return &fnWrapperCtx{
		SchedulerContext: parent,
		wrapper:          wrapper,
	}
}

type fnWrapperCtx struct {
	SchedulerContext
	wrapper func(self SchedulerContext, workFn WorkFn) WorkFn
}

func (ctx *fnWrapperCtx) Work() WorkFn {
	return ctx.wrapper(ctx, ctx.SchedulerContext.Work())
}

func WithTaskId(parent SchedulerContext, taskId string) SchedulerContext {
	return &taskIdCtx{
		SchedulerContext: parent,
		taskId:           taskId,
	}
}

type taskIdCtx struct {
	SchedulerContext
	taskId string
}

func (ctx *taskIdCtx) Value(key any) (any, error) {
	if key == taskIdKey {
		return ctx.taskId, nil
	}
	return ctx.SchedulerContext.Value(key)
}

func GetParam(ctx SchedulerContext) (any, error) {
	return ctx.Value(paramKey)
}

func GetTaskId(ctx SchedulerContext) (string, error) {
	id, err := ctx.Value(taskIdKey)
	if err != nil {
		return "", err
	}
	if id == nil {
		return "", fmt.Errorf("%w: key=%s", ErrValueNotFound, taskIdKey)
	}
	return id.(string), nil
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
