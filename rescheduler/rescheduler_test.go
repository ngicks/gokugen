package rescheduler

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/type-param-common/set"
	"github.com/ngicks/type-param-common/util"
	"github.com/stretchr/testify/assert"
)

type noopScheduler struct {
	set *set.Set[*func(task scheduler.Task, err error)]
}

func newNoopSched() *noopScheduler {
	return &noopScheduler{
		set: set.New[*func(task scheduler.Task, err error)](),
	}
}

func (s *noopScheduler) AddTask(param scheduler.TaskParam) (scheduler.Task, error) {
	return param.ToTask(true), nil
}

func (s *noopScheduler) Update(id string, param scheduler.TaskParam) error {
	return nil
}

func (s *noopScheduler) AddOnTaskDone(fn *func(task scheduler.Task, err error)) {
	s.set.Add(fn)
}

func (s *noopScheduler) RemoveOnTaskDone(fn *func(task scheduler.Task, err error)) {
	s.set.Delete(fn)
}

type NoopSchedule struct{}

func (s NoopSchedule) Next(param []byte) (next time.Time, nextParam []byte, err error) {
	return time.Time{}, []byte{}, nil
}
func (s NoopSchedule) Initial(from time.Time) (param []byte) {
	return []byte{}
}

type recorderHook struct {
	mu sync.Mutex

	onRescheduleT []scheduler.Task
	onRescheduleE []error
	onTaskErrorT  []scheduler.Task
	onTaskErrorE  []error
}

func (h *recorderHook) OnReschedule(t scheduler.Task, scheduleErr error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.onRescheduleT = append(h.onRescheduleT, t)
	h.onRescheduleE = append(h.onRescheduleE, scheduleErr)
}
func (h *recorderHook) OnTaskError(t scheduler.Task, err error) (shouldContinue bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.onTaskErrorT = append(h.onTaskErrorT, t)
	h.onTaskErrorE = append(h.onTaskErrorE, err)
	return true
}

func TestRescheduler_New_and_Down(t *testing.T) {
	assert := assert.New(t)

	noopSched := newNoopSched()

	rules := make(RescheduleRule)
	rules["foo"] = &LimitedSchedule{
		Schedule: NoopSchedule{},
		N:        3,
	}

	rr := make([]*Rescheduler, 0)
	for i := 0; i < 10; i++ {
		recorderHook := recorderHook{}
		r := New(noopSched, rules, SetHook(&recorderHook))
		rr = append(rr, r)
		assert.Equal(i+1, noopSched.set.Len())
	}

	meta := map[string]string{
		metaKeyId: string(util.Must(json.Marshal(RescheduleMeta{Id: "foo", Param: []byte{}}))),
	}
	for _, cb := range noopSched.set.Values().Collect() {
		(*cb)(scheduler.Task{Meta: meta}, errors.New("foo"))
	}

	for _, r := range rr {
		errMsg := "The function pointer registered to " +
			"the scheduler.Scheduler must be distinct but is not"

		assert.Len(
			r.hook.(*recorderHook).onTaskErrorT, 1,
			errMsg,
		)
		assert.Len(
			r.hook.(*recorderHook).onTaskErrorE, 1,
			errMsg,
		)
		assert.Len(
			r.hook.(*recorderHook).onRescheduleT, 1,
			errMsg,
		)
		assert.Len(
			r.hook.(*recorderHook).onRescheduleE, 1,
			errMsg,
		)
	}

	total := len(rr)
	for idx, r := range rr {
		r.Down()
		assert.Equal(total-idx-1, noopSched.set.Len())
	}
}
