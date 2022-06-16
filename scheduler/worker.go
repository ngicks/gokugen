package scheduler

// Worker represents single loop that executes tasks.
// It will work on a single task at a time.
// It may be in stopped-state where loop is stopped,
// working-state where looping in goroutine,
// or ended-state where no way is given to step into working-state again.
type Worker[T any] struct {
	workingState
	endState
	id           T
	stopCh       chan struct{}
	killCh       chan struct{}
	taskCh       <-chan *Task
	taskReceived func()
	taskDone     func()
}

func NewWorker[T any](id T, taskCh <-chan *Task, taskReceived, taskDone func()) (*Worker[T], error) {
	if taskCh == nil {
		return nil, ErrInvalidArg
	}

	if taskReceived == nil {
		taskReceived = func() {}
	}
	if taskDone == nil {
		taskDone = func() {}
	}

	worker := &Worker[T]{
		id:           id,
		stopCh:       make(chan struct{}, 1),
		killCh:       make(chan struct{}),
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
// If taskCh is closed, Start returns nil, becoming ended-state.
func (w *Worker[T]) Start() (err error) {
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
		default:
		}
	}()

	var normalReturn bool
	defer func() {
		if !normalReturn {
			w.setEnded()
		}
	}()

LOOP:
	for {
		select {
		case <-w.killCh:
			// in case of racy kill
			break LOOP
		case <-w.stopCh:
			break LOOP
		default:
			select {
			case <-w.stopCh:
				break LOOP
			case task, ok := <-w.taskCh:
				if !ok {
					w.setEnded()
					break LOOP
				}
				w.taskReceived()
				task.Do(w.killCh)
				w.taskDone()
			}
		}
	}
	// If task exits abnormally, called runtime.Goexit or panicing, it would not reach this line.
	normalReturn = true
	return
}

// Stop stops an active Start loop.
// If Start is not in work when Stop is called,
// it will stops next Start immediately after an invocation of Start.
func (w *Worker[T]) Stop() {
	select {
	case <-w.stopCh:
	default:
	}
	w.stopCh <- struct{}{}
	return
}

// Kill kills this worker.
// If a task is being worked,
// a channel passed to the task will be closed immediately after an invocation of this method.
// Kill makes this worker to step into ended state, making it impossible to Start-ed again.
func (w *Worker[T]) Kill() {
	if w.setEnded() {
		close(w.killCh)
	}
	w.Stop()
}

func (w *Worker[T]) Id() T {
	return w.id
}
