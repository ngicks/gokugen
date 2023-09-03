package scheduler

import (
	"strings"

	"github.com/ngicks/gokugen/def"
)

type State string

const (
	TimerUpdateError State = "timer_update_error"
	AwaitingNext     State = "awaiting_next"
	NextTask         State = "next_task"
	DispatchErr      State = "dispatch_err"
	Dispatched       State = "dispatched"
	TaskDone         State = "task_done"
)

type nextTask struct {
	Task def.Task
	Err  error
}

type dispatchResult struct {
	Task   def.Task
	IsRepo bool
	Err    error
}

type taskDoneResult struct {
	Id        string
	TaskErr   error
	UpdateErr error
}

type StepState struct {
	s State
	v any
}

func StateTimerUpdateError(err error) StepState {
	return StepState{
		s: TimerUpdateError,
		v: err,
	}
}

func StateAwaitingNext(err error) StepState {
	return StepState{
		s: AwaitingNext,
		v: err,
	}
}

func StateNextTask(task def.Task, err error) StepState {
	return StepState{
		s: NextTask,
		v: nextTask{
			Task: task,
			Err:  err,
		},
	}
}

func StateDispatchErr(task def.Task, isRepo bool, err error) StepState {
	return StepState{
		s: DispatchErr,
		v: dispatchResult{
			Task:   task,
			IsRepo: isRepo,
			Err:    err,
		},
	}
}

func StateDispatched(id string) StepState {
	return StepState{
		s: Dispatched,
		v: id,
	}
}

func StateTaskDone(id string, taskErr error, updateErr error) StepState {
	return StepState{
		s: TaskDone,
		v: taskDoneResult{
			Id:        id,
			TaskErr:   taskErr,
			UpdateErr: updateErr,
		},
	}
}

func (r StepState) Match(h StepResultHandler) error {
	if ok, report := h.IsValid(); !ok {
		panic(report)
	}

	switch r.s {
	default:
		panic("unknown state")
	case TimerUpdateError:
		return h.TimerUpdateError(r.v.(error))
	case AwaitingNext:
		return h.AwaitingNext(r.v.(error))
	case NextTask:
		v := r.v.(nextTask)
		return h.NextTask(v.Task, v.Err)
	case DispatchErr:
		v := r.v.(dispatchResult)
		return h.DispatchErr(v.Task, v.IsRepo, v.Err)
	case Dispatched:
		return h.Dispatched(r.v.(string))
	case TaskDone:
		v := r.v.(taskDoneResult)
		return h.TaskDone(v.Id, v.TaskErr, v.UpdateErr)
	}
}

func (r StepState) Err() error {
	return r.Match(StepResultHandler{
		TimerUpdateError: func(err error) error {
			return err
		},
		AwaitingNext: func(err error) error {
			return err
		},
		NextTask: func(task def.Task, err error) error {
			return err
		},
		DispatchErr: func(task def.Task, isRepo bool, err error) error {
			return err
		},
		Dispatched: func(id string) error {
			return nil
		},
		TaskDone: func(id string, taskErr error, updateErr error) error {
			return updateErr
		},
	})
}

// StepResultHandler handles all StepResult States.
// If StepResultHandler has any nil field it is considered invalid,
// causing panic when invoked with (StepResult).Match.
type StepResultHandler struct {
	TimerUpdateError func(err error) error
	AwaitingNext     func(err error) error
	NextTask         func(task def.Task, err error) error
	DispatchErr      func(task def.Task, isRepo bool, err error) error
	Dispatched       func(id string) error
	TaskDone         func(id string, taskErr error, updateErr error) error
}

func (h StepResultHandler) IsValid() (ok bool, report string) {
	builder := strings.Builder{}
	if h.TimerUpdateError == nil {
		builder.WriteString(", TimerUpdateError")
	}
	if h.AwaitingNext == nil {
		builder.WriteString(", AwaitingNext")
	}
	if h.NextTask == nil {
		builder.WriteString(", NextFetchErr")
	}
	if h.DispatchErr == nil {
		builder.WriteString(", DispatchErr")
	}
	if h.Dispatched == nil {
		builder.WriteString(", Dispatched")
	}
	if h.TaskDone == nil {
		builder.WriteString(", TaskDone")
	}

	if builder.Len() > 0 {
		return false, "h has nil fields (" + builder.String()[2:] + ")"
	}
	return true, ""
}
