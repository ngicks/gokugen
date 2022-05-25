package scheduler

import (
	"sync"
)

// WorkerPool is container for workers.
// Workers can be created as sleeping state(no active goroutine is assigned).
// And then, they can be waked (looping in assigned groutine) as many as needed.
type WorkerPool struct {
	mu     sync.RWMutex
	status workingState
	wg     sync.WaitGroup

	workerConstructor func() *Worker
	workers           []*Worker
	sleptWorkers      []*Worker
}

func NewWorkerPool(workerConstructor func() *Worker) *WorkerPool {
	w := WorkerPool{
		workerConstructor: workerConstructor,
		workers:           make([]*Worker, 0),
		sleptWorkers:      make([]*Worker, 0),
	}
	return &w
}

func (p *WorkerPool) Add(delta uint32) (newLen int) {
	p.mu.Lock()
	for i := uint32(0); i < delta; i++ {
		worker := p.workerConstructor()
		p.wg.Add(1)
		go func() {
			worker.Start()
			p.wg.Done()
		}()
		p.workers = append(p.workers, worker)
	}
	p.mu.Unlock()
	return p.Len()
}

func (p *WorkerPool) Remove(delta uint32) (len int) {
	p.mu.Lock()
	removed, _ := remove(&p.workers, uint(delta))
	for _, w := range removed {
		w.Stop()
		p.sleptWorkers = append(p.sleptWorkers, w)
	}
	p.mu.Unlock()
	return p.Len()
}

func (p *WorkerPool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.workers)
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
	p.workers = p.workers[:]
	for _, w := range p.sleptWorkers {
		w.Kill()
	}
	p.sleptWorkers = p.sleptWorkers[:]
}

// Wait waits for all workers to stop.
// Calling this before sleeping or removing all worker may block forever.
func (p *WorkerPool) Wait() {
	p.wg.Wait()
}

func remove(sl *[]*Worker, delta uint) (removed []*Worker, remaining uint) {
	var roundedDelta uint
	if delta > uint(len(*sl)) {
		roundedDelta = uint(len(*sl))
	} else {
		roundedDelta = delta
	}
	*sl, removed = (*sl)[:uint(len(*sl))-roundedDelta], (*sl)[uint(len(*sl))-roundedDelta:]
	remaining = delta - uint(len(removed))
	return
}
