package scheduler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ngicks/gokugen/common"
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

func NewScheduler(initialWorkerNum, queueMax uint, getNow common.GetNow) *Scheduler {
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
			w, err := NewWorker(taskCh, received, done)
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

func (s *Scheduler) SchedTask(task *Task) (*TaskController, error) {
	if s.IsEnded() {
		return nil, ErrAlreadyEnded
	}

	err := s.dispatchLoop.PushTask(task)
	if err != nil {
		return nil, err
	}
	return &TaskController{t: task}, nil
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
	return int(s.workerPool.Add(delta))
}

func (s *Scheduler) RemoveWorker(delta uint32) (workerNum int) {
	return s.workerPool.Remove(delta)
}

func (s *Scheduler) ActiveWorkerNum() int64 {
	return atomic.LoadInt64(s.activeWorkerNum)
}

// End remove all workers and let this scheduler to step into ended-state where no new Start is allowed.
// End also cancel tasks if they are working on in any work.
// Calling this method *before* cancelling of ctx passed to Start will cause blocking forever.
func (s *Scheduler) End() {
	s.setEnded()
	// wait for the Start loop to be done.
	s.wg.Wait()
	s.workerPool.Kill()
	s.workerPool.Wait()
}
