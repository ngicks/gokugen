package cron_test

import (
	"sync"
	"testing"
	"time"

	"github.com/ngicks/gokugen"
	"github.com/ngicks/gokugen/cron"
	"github.com/ngicks/type-param-common/iterator"
	syncparam "github.com/ngicks/type-param-common/sync-param"
	"github.com/stretchr/testify/require"
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
	ctxList []gokugen.SchedulerContext
}

func (s *fakeScheduler) runAllTasks() (results []error) {
	var cloned []gokugen.SchedulerContext
	func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		cloned = make([]gokugen.SchedulerContext, len(s.ctxList))
		copy(cloned, s.ctxList)
	}()

	iter := iterator.Iterator[gokugen.SchedulerContext]{
		DeIterator: iterator.FromSlice(cloned),
	}.SkipN(s.idx)

	s.idx = iter.Len()

	for next, ok := iter.Next(); ok; next, ok = iter.Next() {
		_, err := next.Work()(make(<-chan struct{}), make(<-chan struct{}), next.ScheduledTime())
		results = append(results, err)
	}
	return
}

func (s *fakeScheduler) Schedule(ctx gokugen.SchedulerContext) (gokugen.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctxList = append(s.ctxList, ctx)
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
		ctxList: make([]gokugen.SchedulerContext, 0),
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
	t.Parallel()
	t.Run("rescheduling is controlled by shouldReschedule passed by user.", func(t *testing.T) {
		_, scheduler, _, cronLikeRescheduler := prepareController(
			func(workErr error, callCount int) bool { return callCount == 0 },
		)

		err := cronLikeRescheduler.Schedule()
		if err != nil {
			t.Fatal(err)
		}

		results := scheduler.runAllTasks()
		require.Len(t, results, 1)
		for _, res := range results {
			if res != nil {
				t.Fatal(res)
			}
		}

		results = scheduler.runAllTasks()
		require.Len(t, results, 1)
		for _, res := range results {
			if res != nil {
				t.Fatal(res)
			}
		}

		results = scheduler.runAllTasks()
		require.Len(t, results, 0)
	})

}
