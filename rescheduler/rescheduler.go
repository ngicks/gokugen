package rescheduler

import (
	"errors"
	"time"

	"github.com/ngicks/gokugen/scheduler"
)

const (
	metaKeyId       = "github.com/ngicks/gokugen/rescheduler:id"
	metaKeyParam    = "github.com/ngicks/gokugen/rescheduler:param"
	metaKeyDone     = "github.com/ngicks/gokugen/rescheduler:done"
	metaKeyParentId = "github.com/ngicks/gokugen/rescheduler:parent_id"
)

func init() {
	scheduler.RegisterMeta(metaKeyId)
	scheduler.RegisterMeta(metaKeyParam)
	scheduler.RegisterMeta(metaKeyDone)
	scheduler.RegisterMeta(metaKeyParentId)
}

var _ Scheduler = (*scheduler.Scheduler)(nil)

// Scheduler is an interface compatible with scheduler.Scheduler.
// This allows wrapped schedulers.
type Scheduler interface {
	AddTask(param scheduler.TaskParam) (scheduler.Task, error)
	Update(id string, param scheduler.TaskParam) error
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

func (r *Rescheduler) AddTaskWithReschedule(
	param scheduler.TaskParam,
	ruleId string,
	from time.Time,
) (scheduler.Task, error) {
	param = param.Clone()

	rule, ok := r.rule[ruleId]
	if !ok {
		return scheduler.Task{}, &RuleNotFoundErr{ruleId}
	}

	reschedParam := rule.Initial(from)

	param.Meta[metaKeyId] = ruleId
	param.Meta[metaKeyParam] = string(reschedParam)
	param.Meta[metaKeyDone] = "false"
	param.Meta[metaKeyParentId] = ""

	return r.sched.AddTask(param)
}

func (r *Rescheduler) onTaskDone(task scheduler.Task, err error) {
	rescheduleId, param, done, _, ok := GetMeta(task.Meta)
	if !ok {
		return
	}
	if done == "true" {
		return
	}

	// Deferred error check. If meta is not set, it has nothing to do with it.
	if err != nil {
		var e *Done
		if errors.As(err, &e) || !r.hook.OnTaskError(task, err) {
			return
		}
	}

	newTask, err := r.reschedule(task, rescheduleId, param)
	r.hook.OnReschedule(newTask, err)
}

// GetMeta retrieves rescheduling related meta data from task.Meta.
func GetMeta(meta map[string]string) (rescheduleId, param, done, parentId string, ok bool) {
	rescheduleId, ok = meta[metaKeyId]
	if !ok {
		return "", "", "", "", false
	}
	param, ok = meta[metaKeyParam]
	if !ok {
		return "", "", "", "", false
	}
	done, ok = meta[metaKeyDone]
	if !ok {
		return "", "", "", "", false
	}
	parentId, ok = meta[metaKeyParentId]
	if !ok {
		return "", "", "", "", false
	}
	return rescheduleId, param, done, parentId, true
}

func (r *Rescheduler) reschedule(
	oldTask scheduler.Task, rescheduleId, param string,
) (scheduler.Task, error) {
	rule, ok := r.rule[rescheduleId]
	if !ok {
		return scheduler.Task{}, &RuleNotFoundErr{rescheduleId}
	}

	nextTime, nextParam, err := rule.Next([]byte(param))
	if err != nil {
		return scheduler.Task{}, &RuleNextErr{rescheduleId, []byte(param), err}
	}

	newParam := oldTask.ToParam().Clone()
	newParam.ScheduledAt = nextTime

	newParam.Meta[metaKeyParentId] = oldTask.Id
	newParam.Meta[metaKeyParam] = string(nextParam)

	newTask, err := r.sched.AddTask(newParam)
	if err != nil {
		return scheduler.Task{}, err
	}

	updatedToDone := oldTask.ToParam().Clone().Meta
	updatedToDone[metaKeyDone] = "true"
	_ = r.sched.Update(oldTask.Id, scheduler.TaskParam{Meta: updatedToDone})

	return newTask, nil
}

func (r *Rescheduler) Down() {
	if r.cb == nil {
		return
	}
	r.sched.RemoveOnTaskDone(r.cb)
	r.cb = nil
}
