package scheduler

import "context"

// Worker represents single loop that executes tasks.
// It will work on a single task at a time.
// It may be in stopped-state where loop is stopped,
// working-state where looping in goroutine,
// or ended-state where no way is given to step into working-state again.
type Worker struct {
	workingState
	endState
	stopCh       chan struct{}
	getNow       GetNow
	taskCh       <-chan *Task
	taskReceived func()
	taskDone     func()
}

func NewWorker(taskCh <-chan *Task, taskReceived, taskDone func(), getNow GetNow) (*Worker, error) {
	if taskCh == nil || getNow == nil {
		return nil, ErrInvalidArg
	}

	if taskReceived == nil {
		taskReceived = func() {}
	}
	if taskDone == nil {
		taskDone = func() {}
	}

	worker := &Worker{
		stopCh:       make(chan struct{}, 1),
		getNow:       getNow,
		taskCh:       taskCh,
		taskReceived: taskReceived,
		taskDone:     taskDone,
	}
	return worker, nil
}

// Start starts worker loop. So it would block long.
//
// If worker is already ended, it returns `ErrAlreadyEnded`.
// If worker is already started, it returns `ErrAlreadyStarted`.
// If taskCh is closed, Start returns nil immediately, becomes ended-state.
func (w *Worker) Start(ctx context.Context) (err error) {

	if w.IsEnded() {
		return ErrAlreadyEnded
	}
	if !w.setWorking() {
		return ErrAlreadyStarted
	}
	defer w.setWorking(false)

	defer func() {
		select {
		case <-w.stopCh:
			break
		default:
			break
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case job, ok := <-w.taskCh:
			if !ok {
				w.setEnded()
				return
			}
			w.taskReceived()
			job.Do(w.getNow.GetNow())
			w.taskDone()
		}
	}
}

// Stop stops an active Start loop.
// If Start is not in work when Stop is called, it will stops next Start immediately after invocation.
func (w *Worker) Stop() {
	select {
	case <-w.stopCh:
	default:
	}
	w.stopCh <- struct{}{}
	return
}
