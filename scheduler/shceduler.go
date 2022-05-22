package scheduler

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// Scheduler is an in-memory scheduler backed by min heap.
type Scheduler struct {
	workingState
	endState
	wg sync.WaitGroup

	feeder          *TaskFeeder
	cancellerLoop   *CancellerLoop
	dispatchLoop    *DispatchLoop
	workerPool      *WorkerPool
	activeWorkerNum *int64
	taskCh          chan *Task
}

func NewScheduler(initialWorkerNum, queueMax uint, getNow GetNow) *Scheduler {
	taskCh := make(chan *Task)
	feeder := NewTaskFeeder(queueMax, getNow, NewTimerImpl())
	var activeWorkerNum int64
	received := func() {
		atomic.AddInt64(&activeWorkerNum, 1)
	}
	done := func() {
		atomic.AddInt64(&activeWorkerNum, -1)
	}
	s := &Scheduler{
		feeder:        feeder,
		cancellerLoop: NewCancellerLoop(feeder, getNow, time.Minute),
		dispatchLoop:  NewDispatchLoop(feeder, getNow),
		workerPool: NewWorkerPool(func() *Worker {
			w, err := NewWorker(taskCh, received, done, getNow)
			if err != nil {
				panic(err)
			}
			return w
		}),
		activeWorkerNum: &activeWorkerNum,
		taskCh:          taskCh,
	}
	s.AddWorker(uint32(initialWorkerNum))
	return s
}

func (s *Scheduler) SchedTask(targetTime time.Time, work func(scheduled, current time.Time)) (*TaskController, error) {
	if s.IsEnded() {
		return nil, ErrAlreadyEnded
	}

	t := NewTask(targetTime, work)
	err := s.dispatchLoop.PushTask(t)
	if err != nil {
		return nil, err
	}
	return &TaskController{t: t}, nil
}

type LoopError struct {
	cancellerLoopErr error
	dispatchLoopErr  error
}

func (e LoopError) Error() string {
	return fmt.Sprintf(
		"cancellerLoopErr: %s, dispatchLoopErr: %s",
		e.cancellerLoopErr,
		e.dispatchLoopErr,
	)
}

// Start starts loops to facilitate scheduler.
// Start creates one goroutine for periodical-cancelled-task-removal.
// Start blocks until ctx is cancelled, and other loops to return.
func (s *Scheduler) Start(ctx context.Context) error {
	if s.IsEnded() {
		return ErrAlreadyEnded
	}
	if !s.setWorking() {
		return ErrAlreadyStarted
	}
	defer s.setWorking(false)

	err := new(LoopError)
	s.wg.Add(1)
	s.feeder.Start()
	go func() {
		err.cancellerLoopErr = s.cancellerLoop.Start(ctx)
		s.wg.Done()
	}()
	err.dispatchLoopErr = s.dispatchLoop.Start(ctx, s.taskCh)
	s.wg.Wait()
	s.feeder.Stop()

	if err.cancellerLoopErr == nil && err.dispatchLoopErr == nil {
		return nil
	}
	return err
}

func (s *Scheduler) AddWorker(delta uint32) (workerNum int) {
	s.workerPool.Add(delta)
	s.workerPool.Wake(math.MaxUint32)
	workerNum, _, _ = s.workerPool.Len()
	return
}

func (s *Scheduler) RemoveWorker(delta uint32) (workerNum int) {
	workerNum, _, _ = s.workerPool.Remove(delta, true)
	return
}

func (s *Scheduler) ActiveWorkerNum() int64 {
	return atomic.LoadInt64(s.activeWorkerNum)
}

// End remove all workers and let this scheduler to ended-state where no new Start is allowed.
// Calling this method *before* cancelling of ctx passed to Start will cause blocking forever.
func (s *Scheduler) End() {
	s.setEnded()
	// wait for the Start loop to be done.
	s.wg.Wait()
	s.RemoveWorker(math.MaxInt32)
	s.workerPool.Wait()
}
