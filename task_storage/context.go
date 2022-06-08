package taskstorage

import (
	"errors"
	"fmt"
	"sync"
	"time"

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
	return newTaskStorageCtx(parent, workId, param)
}

type taskStorageCtx struct {
	gokugen.SchedulerContext
	mu     sync.Mutex
	taskId string
	workId string
	param  any
	work   WorkFn
}

func newTaskStorageCtx(parent gokugen.SchedulerContext, workId string, param any) *taskStorageCtx {
	return &taskStorageCtx{
		SchedulerContext: parent,
		workId:           workId,
		param:            param,
	}
}

func (ctx *taskStorageCtx) Work() WorkFn {
	ctx.mu.Lock()
	if ctx.work != nil {
		defer ctx.mu.Unlock()
		return ctx.work
	} else {
		ctx.mu.Unlock()
		return ctx.SchedulerContext.Work()
	}
}

func (ctx *taskStorageCtx) Value(key any) (any, error) {
	ctx.mu.Lock()
	switch key {
	case taskIdKey:
		defer ctx.mu.Unlock()
		return ctx.taskId, nil
	case workIdKey:
		defer ctx.mu.Unlock()
		return ctx.workId, nil
	case paramKey:
		defer ctx.mu.Unlock()
		return ctx.param, nil
	}
	ctx.mu.Unlock()
	return ctx.SchedulerContext.Value(key)
}

type taskStorageParamLoadableCtx struct {
	gokugen.SchedulerContext
	mu            sync.Mutex
	taskId        string
	workId        string
	paramLoader   func() (any, error)
	workWithParam WorkFnWParam
	postProcesse  func(error)
}

func newtaskStorageParamLoadableCtx(
	parent *taskStorageCtx,
	paramLoader func() (any, error),
	workWithParam WorkFnWParam,
	postProcesse func(error),
) *taskStorageParamLoadableCtx {
	return &taskStorageParamLoadableCtx{
		SchedulerContext: parent.SchedulerContext,
		taskId:           parent.taskId,
		workId:           parent.workId,
		paramLoader:      paramLoader,
		workWithParam:    workWithParam,
		postProcesse:     postProcesse,
	}
}

func (ctx *taskStorageParamLoadableCtx) Work() WorkFn {
	return func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) error {
		param, err := ctx.paramLoader()
		if err != nil {
			return err
		}
		err = ctx.workWithParam(ctxCancelCh, taskCancelCh, scheduled, param)
		ctx.postProcesse(err)
		if err != nil {
			return err
		}
		return nil
	}
}

func (ctx *taskStorageParamLoadableCtx) Value(key any) (any, error) {
	switch key {
	case taskIdKey:
		return ctx.taskId, nil
	case workIdKey:
		return ctx.workId, nil
	case paramKey:
		return ctx.paramLoader()
	}
	return ctx.SchedulerContext.Value(key)
}

type taskStorageMarkingWorkingCtx struct {
	gokugen.SchedulerContext
	taskId string
	repo   RepositoryUpdater
}

func newtaskStorageMarkingWorkingCtx(
	parent gokugen.SchedulerContext,
	taskId string, repo RepositoryUpdater,
) *taskStorageMarkingWorkingCtx {
	return &taskStorageMarkingWorkingCtx{
		SchedulerContext: parent,
		taskId:           taskId,
		repo:             repo,
	}
}

func (ctx *taskStorageMarkingWorkingCtx) Work() WorkFn {
	return func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) error {
		swapped, err := ctx.repo.UpdateState(ctx.taskId, Initialized, Working)
		if err != nil {
			return err
		}
		if !swapped {
			return fmt.Errorf("%w: task id = %s", ErrOtherNodeWorkingOnTheTask, ctx.taskId)
		}
		return ctx.Work()(ctxCancelCh, taskCancelCh, scheduled)
	}
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

func setWork(ctx gokugen.SchedulerContext, work WorkFn) (ok bool) {
	v, ok := ctx.(*taskStorageCtx)
	if !ok {
		return false
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.work = work
	return true
}

func setTaskId(ctx gokugen.SchedulerContext, taskId string) (ok bool) {
	v, ok := ctx.(*taskStorageCtx)
	if !ok {
		return false
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.taskId = taskId
	return true
}
