package rescheduler

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/ngicks/gokugen/scheduler"
)

const metaKey = "github.com/ngicks/gokugen/rescheduler"

var _ Scheduler = &scheduler.Scheduler{}

// Scheduler is an interface compatible with scheduler.Scheduler.
// This allows wrapped schedulers.
type Scheduler interface {
	AddTask(param scheduler.TaskParam) (scheduler.Task, error)
	AddOnTaskDone(fn *func(task scheduler.Task, err error))
	RemoveOnTaskDone(fn *func(task scheduler.Task, err error))
}

type ReschedulerHook interface {
	OnReschedule(t scheduler.Task, scheduleErr error)
	OnTaskError(t scheduler.Task, err error) (shouldContinue bool)
}

type NoopHook struct{}

func (h NoopHook) OnReschedule(t scheduler.Task, scheduleErr error) {}
func (h NoopHook) OnTaskError(t scheduler.Task, err error) (shouldContinue bool) {
	return true
}

type Rescheduler struct {
	sched Scheduler
	cb    *func(task scheduler.Task, err error) // cb points to r.onTaskDone
	rule  RescheduleRule
	hook  ReschedulerHook
}

func New(sched Scheduler, rule RescheduleRule, opts ...Option) *Rescheduler {
	r := &Rescheduler{
		sched: sched,
		rule:  rule,
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.hook == nil {
		r.hook = NoopHook{}
	}

	// This is needed to be a distinct pointer.
	fn := r.onTaskDone
	r.cb = &fn
	sched.AddOnTaskDone(r.cb)

	return r
}

func (r *Rescheduler) AddTask(
	ruleId string,
	from time.Time,
	param scheduler.TaskParam,
) (scheduler.Task, error) {
	param = param.Clone()

	rule, ok := r.rule[ruleId]
	if !ok {
		return scheduler.Task{}, &RuleNotFoundErr{ruleId}
	}

	return r.step(ruleId, rule.Initial(from), param)
}

func (r *Rescheduler) step(
	ruleId string,
	oldParam []byte,
	taskParam scheduler.TaskParam,
) (scheduler.Task, error) {
	t, err := r.stepInner(ruleId, oldParam, taskParam)
	r.hook.OnReschedule(t, err)
	return t, err
}

func (r *Rescheduler) stepInner(
	ruleId string,
	oldParam []byte,
	taskParam scheduler.TaskParam,
) (scheduler.Task, error) {
	rule, ok := r.rule[ruleId]
	if !ok {
		return scheduler.Task{}, &RuleNotFoundErr{ruleId}
	}

	nextTime, nextParam, err := rule.Next(oldParam)
	if err != nil {
		return scheduler.Task{}, &RuleNextErr{ruleId, oldParam, err}
	}

	bin, err := json.Marshal(RescheduleMeta{ruleId, nextParam})
	if err != nil {
		// this must not happen.
		panic(err)
	}

	taskParam.ScheduledAt = nextTime
	taskParam.Meta[metaKey] = string(bin)

	task, err := r.sched.AddTask(taskParam)
	r.hook.OnReschedule(task, err)
	return task, nil
}

func (r *Rescheduler) Down() {
	if r.cb == nil {
		return
	}
	r.sched.RemoveOnTaskDone(r.cb)
	r.cb = nil
}

func (r *Rescheduler) onTaskDone(task scheduler.Task, err error) {
	metaData, ok := task.Meta[metaKey]
	if !ok {
		return
	}

	// Deferred error check. If meta is not set, it has nothing to do with it.
	if err != nil {
		var e *Done
		if errors.As(err, &e) || !r.hook.OnTaskError(task, err) {
			return
		}
	}

	var meta RescheduleMeta
	err = json.Unmarshal([]byte(metaData), &meta)
	if err != nil {
		r.hook.OnReschedule(scheduler.Task{}, &MetaUnmarshalErr{err, task})
		return
	}

	_, _ = r.step(meta.Id, meta.Param, task.ToParam().Clone())
}
