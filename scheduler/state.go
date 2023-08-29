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
	Id  string
	Err error
}

type taskDoneResult struct {
	Id        string
	TaskErr   error
	UpdateErr error
}

type StepResult struct {
	s State
	v any
}

func StateTimerUpdateError(err error) StepResult {
	return StepResult{
		s: TimerUpdateError,
		v: err,
	}
}

func StateAwaitingNext(err error) StepResult {
	return StepResult{
		s: AwaitingNext,
		v: err,
	}
}

func StateNextTask(task def.Task, err error) StepResult {
	return StepResult{
		s: NextTask,
		v: nextTask{
			Task: task,
			Err:  err,
		},
	}
}

func StateDispatchErr(id string, err error) StepResult {
	return StepResult{
		s: DispatchErr,
		v: dispatchResult{
			Id:  id,
			Err: err,
		},
	}
}

func StateDispatched(id string) StepResult {
	return StepResult{
		s: Dispatched,
		v: id,
	}
}

func StateTaskDone(id string, taskErr error, updateErr error) StepResult {
	return StepResult{
		s: TaskDone,
		v: taskDoneResult{
			Id:        id,
			TaskErr:   taskErr,
			UpdateErr: updateErr,
		},
	}
}

func (r StepResult) Match(h StepResultHandler) error {
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
		return h.DispatchErr(v.Id, v.Err)
	case Dispatched:
		return h.Dispatched(r.v.(string))
	case TaskDone:
		v := r.v.(taskDoneResult)
		return h.TaskDone(v.Id, v.TaskErr, v.UpdateErr)
	}
}

// StepResultHandler handles all StepResult States.
// If StepResultHandler has any nil field it is considered invalid,
// causing panic when invoked with (StepResult).Match.
type StepResultHandler struct {
	TimerUpdateError func(err error) error
	AwaitingNext     func(err error) error
	NextTask         func(task def.Task, err error) error
	DispatchErr      func(id string, err error) error
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