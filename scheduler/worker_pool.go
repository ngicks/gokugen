package scheduler

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
)

type WorkerConstructor = func(id int, onTaskReceived func(), onTaskDone func()) *Worker[int]

// TODO: Change invariants where onTaskReceived_ and onTaskDone_ must not be nil, to
//  onTaskReceived__ and onTaskDone__ must no be nil
func BuildWorkerConstructor(taskCh <-chan *Task, onTaskReceived_ func(), onTaskDone_ func()) WorkerConstructor {
	return func(id int, onTaskReceived__ func(), onTaskDone__ func()) *Worker[int] {
		var onTaskReceived, onTaskDone func()
		if onTaskReceived__ != nil {
			onTaskReceived = func() {
				onTaskReceived_()
				onTaskReceived__()
			}
		} else {
			onTaskReceived = onTaskReceived_
		}
		if onTaskDone__ != nil {
			onTaskDone = func() {
				onTaskDone_()
				onTaskDone__()
			}
		} else {
			onTaskDone = onTaskDone_
		}
		w, err := NewWorker(id, taskCh, onTaskReceived, onTaskDone)
		if err != nil {
			panic(err)
		}
		return w
	}
}

// WorkerPool is container for workers.
type WorkerPool struct {
	mu     sync.RWMutex
	status workingState
	wg     sync.WaitGroup

	workerConstructor WorkerConstructor
	workerIdx         int
	workers           map[int]*Worker[int]
	sleepingWorkers   map[int]*Worker[int]
}

func NewWorkerPool(
	workerConstructor WorkerConstructor,
) *WorkerPool {
	w := WorkerPool{
		workerConstructor: workerConstructor,
		workers:           make(map[int]*Worker[int], 0),
		sleepingWorkers:   make(map[int]*Worker[int], 0),
	}
	return &w
}

func (p *WorkerPool) Add(delta uint32) (newAliveLen int) {
	p.mu.Lock()
	for i := uint32(0); i < delta; i++ {
		workerId := p.workerIdx
		p.workerIdx++
		worker := p.workerConstructor(workerId, nil, nil)
		p.wg.Add(1)
		go p.callWorkerStart(worker, true, func(err error) {})

		p.workers[worker.Id()] = worker
	}
	p.mu.Unlock()
	alive, _ := p.Len()
	return alive
}

var (
	errGoexit = errors.New("runtime.Goexit was called")
)

type panicErr struct {
	err   interface{}
	stack []byte
}

// Error implements error interface.
func (p *panicErr) Error() string {
	return fmt.Sprintf("%v\n\n%s", p.err, p.stack)
}

func (p *WorkerPool) callWorkerStart(worker *Worker[int], shouldRecover bool, abnormalReturnCb func(error)) (workerErr error) {
	var normalReturn, recovered bool
	var abnormalReturnErr error
	// see https://cs.opensource.google/go/x/sync/+/0de741cf:singleflight/singleflight.go;l=138-200;drc=0de741cfad7ff3874b219dfbc1b9195b58c7c490
	defer func() {
		// Done will be done right before the exit.
		defer p.wg.Done()
		p.mu.Lock()
		delete(p.workers, worker.Id())
		delete(p.sleepingWorkers, worker.Id())
		p.mu.Unlock()

		if !normalReturn && !recovered {
			abnormalReturnErr = errGoexit
		}
		if !normalReturn {
			abnormalReturnCb(abnormalReturnErr)
		}
		if recovered && !shouldRecover {
			panic(abnormalReturnErr)
		}
	}()

	func() {
		defer func() {
			if err := recover(); err != nil {
				abnormalReturnErr = &panicErr{
					err:   err,
					stack: debug.Stack(),
				}
			}
		}()
		workerErr = worker.Start()
		normalReturn = true
	}()
	if !normalReturn {
		recovered = true
	}
	return
}

func (p *WorkerPool) Remove(delta uint32) (alive int, sleeping int) {
	p.mu.Lock()
	var count uint32
	for _, worker := range p.workers {
		if count < delta {
			worker.Stop()
			delete(p.workers, worker.Id())
			p.sleepingWorkers[worker.Id()] = worker
			count++
		} else {
			break
		}
	}
	p.mu.Unlock()
	return p.Len()
}

func (p *WorkerPool) Len() (alive int, sleeping int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.workers), len(p.sleepingWorkers)
}

// Kill kills all worker.
// Kill also clears internal slept worker slice.
// It is advised to call this method after excessive Add-s and Remove-s.
func (p *WorkerPool) Kill() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, w := range p.workers {
		w.Kill()
	}
}

// Wait waits for all workers to stop.
// Calling this before sleeping or removing all worker may block forever.
func (p *WorkerPool) Wait() {
	p.wg.Wait()
}
