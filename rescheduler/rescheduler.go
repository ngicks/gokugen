package rescheduler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ngicks/gokugen/scheduler"
)

const metaKey = "gokugen.rescheduler"

type Rescheduler struct {
	sched *scheduler.Scheduler
	cb    *func(task scheduler.Task, err error)
	rule  RescheduleRule
}

func New(s *scheduler.Scheduler, rule RescheduleRule) *Rescheduler {
	r := &Rescheduler{
		sched: s,
		rule:  rule,
	}

	r.sched.RegisterMetaKey(metaKey)
	// This is needed to be a distinct pointer...Maybe.
	cb := func(task scheduler.Task, err error) {
		r.OnTaskDone(task, err)
	}
	r.cb = &cb
	s.AddOnTaskDone(r.cb)

	return r
}

func (r *Rescheduler) AddTask(ruleId string, from time.Time, param scheduler.TaskParam) (scheduler.Task, error) {
	rule, ok := r.rule[ruleId]
	if !ok {
		return scheduler.Task{}, fmt.Errorf("unknown reschedule id = %s", ruleId)
	}
	nextTime, nextParam, err := rule.Next(rule.Initial(from))
	if err != nil {
		return scheduler.Task{}, err
	}
	bin, err := json.Marshal(RescheduleMeta{ruleId, nextParam})
	if err != nil {
		return scheduler.Task{}, err
	}

	param.ScheduledAt = nextTime
	param.Meta[metaKey] = bin

	return r.sched.AddTask(param)
}

func (r *Rescheduler) Down() {
	if r.cb == nil {
		return
	}
	r.sched.RemoveOnTaskDone(r.cb)
	r.cb = nil
}

func (r *Rescheduler) OnTaskDone(task scheduler.Task, err error) {
	if err != nil {
		// TODO: add on-error predicate
		return
	}
	var meta RescheduleMeta
	err = json.Unmarshal(task.Meta[metaKey], &meta)
	if err != nil {
		return
	}
	rule, ok := r.rule[meta.Id]
	if !ok {
		return
	}

	nextTime, nextParam, err := rule.Next(meta.Param)
	if err != nil {
		return
	}

	param := task.ToParam()
	param.ScheduledAt = nextTime
	param.Meta[metaKey] = nextParam
	r.sched.AddTask(param)
}
