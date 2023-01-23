package rescheduler

import "github.com/ngicks/gokugen/scheduler"

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

func (r *Rescheduler) Down() {
	if r.cb == nil {
		return
	}
	r.sched.RemoveOnTaskDone(r.cb)
	r.cb = nil
}

func (r *Rescheduler) OnTaskDone(task scheduler.Task, err error) {

}
