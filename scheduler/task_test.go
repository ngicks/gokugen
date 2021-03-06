package scheduler_test

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ngicks/gokugen/scheduler"
)

type mockTaskSet struct {
	task                              *scheduler.Task
	workCallCount, isContextCancelled *int32
	selectCh                          chan struct{}
	cancel                            func()
}

func (s *mockTaskSet) WorkCallCount() int32 {
	return atomic.LoadInt32(s.workCallCount)
}
func (s *mockTaskSet) IsContextCancelled() bool {
	return atomic.LoadInt32(s.isContextCancelled) == 1
}
func (s *mockTaskSet) GetSelectCh() <-chan struct{} {
	return s.selectCh
}
func (s *mockTaskSet) Close() {
	s.cancel()
}
func (s *mockTaskSet) Task() *scheduler.Task {
	return s.task
}

func mockTaskFactory() *mockTaskSet {
	var workCallCount, isContextCancelled int32

	selectCh := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())

	testSet := &mockTaskSet{
		workCallCount:      &workCallCount,
		isContextCancelled: &isContextCancelled,
		selectCh:           selectCh,
		cancel:             cancel,
	}
	t := scheduler.NewTask(time.Now(), func(taskCtx context.Context, scheduled time.Time) {
		atomic.AddInt32(&workCallCount, 1)
		testSet.selectCh <- struct{}{}
		go func() {
			select {
			case <-ctx.Done():
				return
			case <-taskCtx.Done():
				atomic.StoreInt32(&isContextCancelled, 1)
				testSet.selectCh <- struct{}{}
			}
			return
		}()
		<-ctx.Done()
	})
	testSet.task = t
	return testSet
}

func exhaustSelectChan(ch <-chan struct{}) {
	timer := time.NewTimer(time.Millisecond)
loop:
	for {
		select {
		case <-timer.C:
			break loop
		default:
			{
				select {
				case <-timer.C:
					break loop
				case <-ch:
				default:
				}
			}
		}
	}
}

func TestTask(t *testing.T) {
	t.Run("cancel", func(t *testing.T) {
		taskSet := mockTaskFactory()
		defer exhaustSelectChan(taskSet.GetSelectCh())

		task := taskSet.Task()

		if task.IsCancelled() {
			t.Fatalf("IsCancelled must be false")
		}
		if !task.Cancel() {
			t.Fatalf("closed must be true")
		}
		for i := 0; i < 10; i++ {
			// This does not block. Bacause task is already cancelled, internal work will no be called.
			task.Do(context.TODO())
			if !task.IsCancelled() {
				t.Fatalf("IsCancelled must be true")
			}
			if task.Cancel() {
				t.Fatalf("closed must be false")
			}
		}

		if taskSet.WorkCallCount() != 0 {
			t.Fatalf("work must be called")
		}
	})

	t.Run("do and cancel", func(t *testing.T) {
		taskSet := mockTaskFactory()
		defer exhaustSelectChan(taskSet.GetSelectCh())
		task := taskSet.Task()

		if taskSet.WorkCallCount() != 0 {
			t.Fatalf("work must not be called at this point")
		}
		if task.IsDone() {
			t.Fatalf("IsDone must be false")
		}

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			task.Do(context.TODO())
			wg.Done()
		}()

		<-taskSet.GetSelectCh()

		taskSet.Close()
		wg.Wait()

		if taskSet.WorkCallCount() != 1 {
			t.Fatalf("work call count is not correct")
		}

		// This does not block. Because if it's done, it does not call internal work
		task.Do(context.TODO())

		if !task.IsDone() {
			t.Fatalf("IsDone must be true")
		}

		if !task.Cancel() {
			t.Fatalf("closed must be true")
		}
		if !task.IsCancelled() {
			t.Fatalf("IsCancelled must be true")
		}

		task.Do(context.TODO())

		if taskSet.WorkCallCount() != 1 {
			t.Fatalf("work call count is not correct")
		}
	})

	t.Run("passing already closed chan to Do", func(t *testing.T) {
		taskSet := mockTaskFactory()
		defer exhaustSelectChan(taskSet.GetSelectCh())
		task := taskSet.Task()

		if task.IsDone() {
			t.Fatalf("IsDone must be false")
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		// This does not block.
		task.Do(ctx)
		if !task.IsDone() {
			t.Fatalf("IsDone must be true")
		}
	})

	t.Run("cancelling task and closing chan passed to Do", func(t *testing.T) {
		taskSet := mockTaskFactory()
		defer exhaustSelectChan(taskSet.GetSelectCh())
		task := taskSet.Task()

		ctx := context.Background()
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			task.Do(ctx)
			wg.Done()
		}()

		if taskSet.IsContextCancelled() {
			t.Fatalf("ctx must NOT be cancelled at this point")
		}
		selectCh := taskSet.GetSelectCh()
		// waiting for Do to start
		<-selectCh

		go func() {
			task.Cancel()
		}()
		<-selectCh
		if !taskSet.IsContextCancelled() {
			t.Fatalf("ctx must be cancelled")
		}

		taskSet.Close()
		wg.Wait()
	})

	t.Run("GetScheduledTime", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			n := time.Now().Add(time.Duration(rand.Int()))
			task := scheduler.NewTask(n, func(taskCtx context.Context, scheduled time.Time) {})
			if n != task.GetScheduledTime() {
				t.Errorf("time mismatched! passed=%s, received=%s", n, task.GetScheduledTime())
			}
		}
	})
}
