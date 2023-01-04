package scheduler

import (
	"context"
	"sync"

	"github.com/ngicks/type-param-common/slice"
)

type updateEventQueue struct {
	queue       slice.Deque[*updateEvent]
	mu          sync.Mutex
	isRunning   bool
	beingClosed bool
	hasUpdate   chan struct{}
	eventCh     chan updateEvent
	reserved    map[string]<-chan struct{}
}

func newUpdateEventQueue() *updateEventQueue {
	q := new(updateEventQueue)
	return q
}

func (q *updateEventQueue) init() {
	q.beingClosed = false
	q.hasUpdate = make(chan struct{}, 1)
	q.eventCh = make(chan updateEvent)
}

func (q *updateEventQueue) IsStopped() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return !q.isRunning
}

// IsClosed reports q is closed (stopped) or being closed.
func (q *updateEventQueue) IsClosed() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.isClosed()
}

func (q *updateEventQueue) isClosed() bool {
	return !q.isRunning || q.beingClosed
}

func (q *updateEventQueue) Push(e updateEvent) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.queue.PushBack(&e)

	if !q.isClosed() {
		select {
		case q.hasUpdate <- struct{}{}:
		default:
		}
	}
}

// Reserve reserves update. fn will be called in a newly created goroutine.
func (q *updateEventQueue) Reserve(id string, fn func() updateEvent) {
	doneCh := make(chan struct{})

	q.mu.Lock()
	if q.isClosed() {
		q.mu.Unlock()
		panic("Reserve is called when it is closed")
	}
	q.reserved[id] = doneCh
	q.mu.Unlock()

	go func() {
		q.Push(fn())
		close(doneCh)

		q.mu.Lock()
		delete(q.reserved, id)
		q.mu.Unlock()
	}()
}

func (q *updateEventQueue) WaitReserved() <-chan struct{} {
	q.mu.Lock()

	eventCh := make(chan struct{})
	wg := sync.WaitGroup{}
	for _, doneCh := range q.reserved {
		wg.Add(1)
		go func(doneCh <-chan struct{}) {
			<-doneCh
			eventCh <- struct{}{}
			wg.Done()
		}(doneCh)
	}

	q.mu.Unlock()

	go func() {
		wg.Wait()
		close(eventCh)
	}()

	return eventCh
}

func (q *updateEventQueue) Subscribe() <-chan updateEvent {
	return q.eventCh
}

func (q *updateEventQueue) close() {
	q.mu.Lock()
	if q.isClosed() {
		return
	}
	q.beingClosed = true
	// pushing a nil pointer as a partition.
	// Run loop stops its iteration when
	// reaching that nil pointer.
	q.queue.PushBack(nil)
	close(q.hasUpdate)
	q.mu.Unlock()
}

// Run has no way to stop other than cancelling ctx.
func (q *updateEventQueue) Run(ctx context.Context, started chan<- struct{}) {
	q.mu.Lock()
	if q.isRunning {
		panic("calling Run twice")
	}
	if q.beingClosed {
		panic("calling Run while being closed")
	}

	q.isRunning = true
	q.init()

	q.mu.Unlock()

	go func() {
		<-ctx.Done()
		q.close()
	}()

	defer func() {
		q.mu.Lock()
		q.isRunning = false
		q.beingClosed = false
		q.mu.Unlock()
	}()

	started <- struct{}{}

	for {
		// hasUpdate is closed when close is called (= ctx passed to this function is cancelled)
		_, ok := <-q.hasUpdate
		for {
			event, popped := q.pop()
			if !popped || event == nil {
				break
			}
			q.eventCh <- *event
		}
		if !ok {
			eventCh := q.WaitReserved()
			for range eventCh {
				for {
					event, popped := q.pop()
					if !popped || event == nil {
						break
					}
					q.eventCh <- *event
				}
			}
			close(q.eventCh)
			return
		}
	}
}

func (q *updateEventQueue) pop() (event *updateEvent, popped bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.queue.PopFront()
}
