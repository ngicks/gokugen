package cron_test

import (
	"container/list"
	"sync"
	"testing"
	"time"

	"github.com/ngicks/gokugen"
	"github.com/ngicks/gokugen/cron"
	syncparam "github.com/ngicks/type-param-common/sync-param"
)

var _ cron.RowLike = fakeRowLike{}

type fakeRowLike struct {
	nextTime time.Time
	command  []string
}

func (r fakeRowLike) NextSchedule(now time.Time) (time.Time, error) {
	return r.nextTime, nil
}

func (r fakeRowLike) GetCommand() []string {
	return r.command
}

var _ gokugen.Task = &fakeTask{}

type fakeTask struct {
	ctx gokugen.SchedulerContext
}

func (t fakeTask) Cancel() (cancelled bool) {
	return true
}

func (t fakeTask) CancelWithReason(err error) (cancelled bool) {
	return true
}

func (t fakeTask) GetScheduledTime() time.Time {
	return t.ctx.ScheduledTime()
}
func (t fakeTask) IsCancelled() (cancelled bool) {
	return
}
func (t fakeTask) IsDone() (done bool) {
	return
}

var _ cron.Scheduler = &fakeScheduler{}

type fakeScheduler struct {
	mu      sync.Mutex
	idx     int
	ctxList *list.List
}

func (s *fakeScheduler) runAllTasks() (results []error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	prevLen := s.ctxList.Len()
	cur := s.ctxList.Front()
	for i := 0; i < s.idx; i++ {
		cur = cur.Next()
	}
	for i := 0; i < prevLen-s.idx; i++ {
		ctx := cur.Value.(gokugen.SchedulerContext)
		_, err := ctx.Work()(make(<-chan struct{}), make(<-chan struct{}), ctx.ScheduledTime())
		results = append(results, err)

		cur = cur.Next()
	}
	s.idx = prevLen
	return
}

func (s *fakeScheduler) Schedule(ctx gokugen.SchedulerContext) (gokugen.Task, error) {
	s.ctxList.PushBack(ctx)

	return fakeTask{ctx: ctx}, nil
}

func prepareController(
	shouldReschedule func(workErr error, callCount int) bool,
) (
	rowLike *fakeRowLike,
	scheduler *fakeScheduler,
	registry *syncparam.Map[string, gokugen.WorkFnWParam],
	cronLikeRescheduler *cron.CronLikeRescheduler,
) {
	rowLike = &fakeRowLike{command: []string{"foo", "bar", "baz"}}
	scheduler = &fakeScheduler{
		ctxList: list.New(),
	}
	registry = new(syncparam.Map[string, gokugen.WorkFnWParam])
	registry.LoadOrStore("foo", func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time, param any) (any, error) {
		return nil, nil
	})

	cronLikeRescheduler = cron.NewCronLikeRescheduler(
		rowLike,
		time.Now(),
		shouldReschedule,
		scheduler,
		registry,
	)
	return
}

func TestController(t *testing.T) {
	t.Run("rescheduling is controlled by shouldReschedule passed by user.", func(t *testing.T) {
		_, scheduler, _, cronLikeRescheduler := prepareController(
			func(workErr error, callCount int) bool { return callCount == 0 },
		)

		err := cronLikeRescheduler.Schedule()
		if err != nil {
			t.Fatal(err)
		}

		results := scheduler.runAllTasks()
		if len(results) != 1 {
			t.Fatalf("reschedule is not occuring.")
		}
		for _, res := range results {
			if res != nil {
				t.Fatal(res)
			}
		}

		results = scheduler.runAllTasks()
		if len(results) != 1 {
			t.Fatalf("reschedule is not occuring.")
		}
		for _, res := range results {
			if res != nil {
				t.Fatal(res)
			}
		}

		results = scheduler.runAllTasks()
		if len(results) != 0 {
			t.Fatalf("reschedule is wrongly ocurred.")
		}
	})

}
