package scheduler_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ngicks/gokugen/scheduler"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
func TestScheduler(t *testing.T) {
	t.Run("scheduler basic usage", func(t *testing.T) {
		// TODO: change test or internal structure to remove flakiness.
		workerNum := int64(5)
		s := scheduler.NewScheduler(uint(workerNum), 0)
		now := time.Now()

		taskTicker := make(chan struct{}, 5)
		var count uint32
		taskFunc := func(ctxCanCh, canCh <-chan struct{}, t time.Time) {
			atomic.AddUint32(&count, 1)
			<-taskTicker
		}

		for i := 0; i < 10; i++ {
			s.SchedTask(scheduler.NewTask(now.Add(time.Millisecond), taskFunc))
		}

		ctx, cancel := context.WithCancel(context.Background())
		ctxChangeCh := make(chan struct{})
		go func() {
			<-ctxChangeCh
			s.Start(ctx)
		}()
		ctxChangeCh <- struct{}{}

		if c := atomic.LoadUint32(&count); c != 0 {
			t.Fatalf("worker number limitation is not wokring: should be %d, but %d", 0, c)
		}
		if s.ActiveWorkerNum() != 0 {
			t.Fatalf("there must no be any active worker.")
		}
		time.Sleep(2 * time.Millisecond)
		if c := atomic.LoadUint32(&count); c != 5 {
			t.Fatalf("worker number limitation is not wokring: should be %d, but %d", 5, c)
		}
		if s.ActiveWorkerNum() != workerNum {
			t.Fatalf("Active worker must be %d, but %d", workerNum, s.ActiveWorkerNum())
		}
		for i := 0; i < 3; i++ {
			taskTicker <- struct{}{}
		}
		time.Sleep(time.Millisecond)
		if c := atomic.LoadUint32(&count); c != 8 {
			t.Fatalf("worker number limitation is not wokring: should be %d, but %d", 8, c)
		}

		for i := 0; i < 5; i++ {
			taskTicker <- struct{}{}
		}
		time.Sleep(time.Millisecond)
		if activeWorker := s.ActiveWorkerNum(); activeWorker != 2 {
			t.Fatalf("Active worker must be %d, but %d", 2, activeWorker)
		}

		for i := 0; i < 2; i++ {
			taskTicker <- struct{}{}
		}
		time.Sleep(time.Millisecond)
		if c := atomic.LoadUint32(&count); c != 10 {
			t.Fatalf("worker number limitation is not wokring: should be %d, but %d", 10, c)
		}
		cancel()

		s.End()
	})

	t.Run("ended scheduler can not be started again.", func(t *testing.T) {
		s := scheduler.NewScheduler(5, 0)
		s.End()

		if err := s.Start(context.TODO()); err != scheduler.ErrAlreadyEnded {
			t.Fatalf("mismatched error; must be %v, but is %v", scheduler.ErrAlreadyEnded, err)
		}
	})

	t.Run("restart", func(t *testing.T) {
		s := scheduler.NewScheduler(5, 0)

		now := time.Now()
		var count uint32
		taskFunc := func(ctxCanCh, canCh <-chan struct{}, t time.Time) {
			atomic.AddUint32(&count, 1)
		}

		s.SchedTask(scheduler.NewTask(now.Add(time.Millisecond), taskFunc))
		s.SchedTask(scheduler.NewTask(now.Add(10*time.Millisecond), taskFunc))

		wg := sync.WaitGroup{}
		ctx, cancel := context.WithCancel(context.Background())

		ctxChangeCh := make(chan struct{})
		wg.Add(1)
		go func() {
			<-ctxChangeCh
			s.Start(ctx)
			wg.Done()
		}()
		ctxChangeCh <- struct{}{}

		time.Sleep(5 * time.Millisecond)
		if c := atomic.LoadUint32(&count); c != 1 {
			t.Fatalf("worker number limitation is not wokring: should be %d, but %d", 1, c)
		}
		cancel()

		wg.Wait()

		ctx, cancel = context.WithCancel(context.Background())
		wg.Add(1)
		var err error
		go func() {
			<-ctxChangeCh
			err = s.Start(ctx)
			wg.Done()
		}()
		ctxChangeCh <- struct{}{}

		time.Sleep(15 * time.Millisecond)
		if c := atomic.LoadUint32(&count); c != 2 {
			t.Fatalf("worker number limitation is not wokring: should be %d, but %d", 2, c)
		}

		cancel()
		wg.Wait()

		if err != nil {
			t.Fatalf("could not restart; %v", err)
		}

		s.End()
	})

}
