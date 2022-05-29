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
	task                                               *scheduler.Task
	workCallCount, isContextCancelled, isTaskCancelled *int32
	selectCh                                           chan struct{}
	cancel                                             func()
}

func (s *mockTaskSet) WorkCallCount() int32 {
	return atomic.LoadInt32(s.workCallCount)
}
func (s *mockTaskSet) IsContextCancelled() bool {
	return atomic.LoadInt32(s.isContextCancelled) == 1
}
func (s *mockTaskSet) IsTaskCancelled() bool {
	return atomic.LoadInt32(s.isTaskCancelled) == 1
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
	var workCallCount, isContextCancelled, isTaskCancelled int32

	selectCh := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())

	testSet := &mockTaskSet{
		workCallCount:      &workCallCount,
		isContextCancelled: &isContextCancelled,
		isTaskCancelled:    &isTaskCancelled,
		selectCh:           selectCh,
		cancel:             cancel,
	}
	t := scheduler.NewTask(time.Now(), func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) {
		atomic.AddInt32(&workCallCount, 1)
		testSet.selectCh <- struct{}{}
		go func() {
			select {
			case <-ctx.Done():
				return
			case <-ctxCancelCh:
				atomic.StoreInt32(&isContextCancelled, 1)
				testSet.selectCh <- struct{}{}
			}
			return
		}()
		go func() {
			select {
			case <-ctx.Done():
				return
			case <-taskCancelCh:
				atomic.StoreInt32(&isTaskCancelled, 1)
				testSet.selectCh <- struct{}{}
			}
			return
		}()
		<-ctx.Done()
	})
	testSet.task = t
	return testSet
}

func TestTask(t *testing.T) {
	t.Run("cancel", func(t *testing.T) {
		taskSet := mockTaskFactory()
		task := taskSet.Task()

		if task.IsCancelled() {
			t.Fatalf("IsCancelled must be false")
		}
		if !task.Cancel() {
			t.Fatalf("closed must be true")
		}
		for i := 0; i < 10; i++ {
			// This does not block. Bacause task is already cancelled, internal work will no be called.
			task.Do(make(<-chan struct{}))
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
			task.Do(make(<-chan struct{}))
			wg.Done()
		}()

		<-taskSet.GetSelectCh()

		taskSet.Close()
		wg.Wait()

		if taskSet.WorkCallCount() != 1 {
			t.Fatalf("work call count is not correct")
		}

		// This does not block. Because if it's done, it does not call internal work
		task.Do(make(<-chan struct{}))

		if !task.IsDone() {
			t.Fatalf("IsDone must be true")
		}

		if !task.Cancel() {
			t.Fatalf("closed must be true")
		}
		if !task.IsCancelled() {
			t.Fatalf("IsCancelled must be true")
		}

		task.Do(make(<-chan struct{}))

		if taskSet.WorkCallCount() != 1 {
			t.Fatalf("work call count is not correct")
		}
	})

	t.Run("passing already closed chan to Do", func(t *testing.T) {
		testSet := mockTaskFactory()
		task := testSet.Task()

		if task.IsDone() {
			t.Fatalf("IsDone must be false")
		}

		ctxCancelCh := make(chan struct{})
		close(ctxCancelCh)
		// This does not block.
		task.Do(ctxCancelCh)
		if !task.IsDone() {
			t.Fatalf("IsDone must be true")
		}
	})

	t.Run("cancelling task and closing chan passed to Do", func(t *testing.T) {
		testSet := mockTaskFactory()
		task := testSet.Task()

		ctxCancelCh := make(chan struct{})
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			task.Do(ctxCancelCh)
			wg.Done()
		}()

		if testSet.IsContextCancelled() {
			t.Fatalf("ctx must NOT be cancelled at this point")
		}
		if testSet.IsTaskCancelled() {
			t.Fatalf("task must NOT be cancelled at this point")
		}

		selectCh := testSet.GetSelectCh()
		// waiting for Do to start
		<-selectCh

		go func() {
			task.Cancel()
		}()
		<-selectCh
		if !testSet.IsTaskCancelled() {
			t.Fatalf("task must be cancelled")
		}
		if testSet.IsContextCancelled() {
			t.Fatalf("ctx must NOT be cancelled")
		}

		go func() {
			close(ctxCancelCh)
		}()
		<-selectCh

		if !testSet.IsTaskCancelled() {
			t.Fatalf("task must be cancelled")
		}
		if !testSet.IsContextCancelled() {
			t.Fatalf("ctx must be cancelled")
		}

		testSet.Close()
		wg.Wait()
	})

	t.Run("GetScheduledTime", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			n := time.Now().Add(time.Duration(rand.Int()))
			task := scheduler.NewTask(n, func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) {})
			if n != task.GetScheduledTime() {
				t.Errorf("time mismatched! passed=%s, received=%s", n, task.GetScheduledTime())
			}
		}
	})
}
