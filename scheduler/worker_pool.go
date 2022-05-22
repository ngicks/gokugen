package scheduler

import (
	"context"
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
	waked             []*Worker
	asleep            []*Worker
}

func NewWorkerPool(workerConstructor func() *Worker) *WorkerPool {
	w := WorkerPool{
		workerConstructor: workerConstructor,
		waked:             make([]*Worker, 0),
		asleep:            make([]*Worker, 0),
	}
	return &w
}

func (p *WorkerPool) Add(delta uint32) (new uint) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := uint32(0); i < delta; i++ {
		p.asleep = append(p.asleep, p.workerConstructor())
	}
	return uint(len(p.waked) + len(p.asleep))
}

func (p *WorkerPool) Remove(delta uint32, removeFromActive bool) (total, waked, asleep int) {
	p.mu.Lock()
	var removedWaked []*Worker
	if removeFromActive {
		var remaining uint
		removedWaked, remaining = remove(&p.waked, uint(delta))
		remove(&p.asleep, remaining)
	} else {
		_, remaining := remove(&p.asleep, uint(delta))
		removedWaked, _ = remove(&p.waked, remaining)
	}
	for _, w := range removedWaked {
		w.Stop()
	}
	p.mu.Unlock()
	return p.Len()
}

func (p *WorkerPool) Wake(delta uint32) (waked uint) {
	p.mu.Lock()
	defer p.mu.Unlock()

	removed, _ := remove(&p.asleep, uint(delta))

	waked = uint(len(removed))
	for _, w := range removed {
		p.wg.Add(1)
		go func(w *Worker) {
			w.Start(context.Background())
			p.wg.Done()
		}(w)
	}
	p.waked = append(p.waked, removed...)
	return
}

func (p *WorkerPool) Sleep(delta uint32) (slept uint) {
	p.mu.Lock()
	defer p.mu.Unlock()

	removed, _ := remove(&p.waked, uint(delta))

	slept = uint(len(removed))
	for _, w := range removed {
		w.Stop()
	}
	p.asleep = append(p.asleep, removed...)
	return
}

func (p *WorkerPool) Len() (total, waked, asleep int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	waked = len(p.waked)
	asleep = len(p.asleep)
	total = waked + asleep
	return
}

// Wait wait until all workers are aleep or removed.
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
