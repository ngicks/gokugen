package repository

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/gommon/pkg/common"
	"github.com/ngicks/type-param-common/heap"
	"github.com/ngicks/type-param-common/slice"
	"github.com/ngicks/type-param-common/util"
)

type wrappedTask struct {
	scheduler.Task
	Index int
}

func (t *wrappedTask) Less(i, j *wrappedTask) bool {
	if i.CancelledAt != nil {
		return true
	}

	if !i.ScheduledAt.Equal(j.ScheduledAt) {
		return i.ScheduledAt.Before(j.ScheduledAt)
	}
	return i.Priority > j.Priority
}

func (t *wrappedTask) Swap(slice *slice.Stack[*wrappedTask], i, j int) {
	(*slice)[i], (*slice)[j] = (*slice)[j], (*slice)[i]
	(*slice)[i].Index = i
	(*slice)[j].Index = j
}

func (t *wrappedTask) Push(slice *slice.Stack[*wrappedTask], v *wrappedTask) {
	v.Index = slice.Len()
	slice.Push(v)
}

func (t *wrappedTask) Pop(slice *slice.Stack[*wrappedTask]) *wrappedTask {
	popped, ok := slice.Pop()
	if !ok {
		_ = (*(*[]*wrappedTask)(slice))[slice.Len()-1]
	}
	popped.Index = -1
	return popped
}

type HeapRepository struct {
	mu sync.RWMutex

	isMilliSecPrecise bool // if true, all internal times will be truncated to multiple of milli sec. mainly for testing.

	heap            *heap.FilterableHeap[*wrappedTask]
	mapLike         map[string]*wrappedTask
	beingDispatched map[string]*wrappedTask
	getNow          common.GetNower
	timer           common.Timer
	isTimerStarted  bool
}

func NewHeapRepository() *HeapRepository {
	return &HeapRepository{
		heap:            heap.NewFilterableHeap[*wrappedTask](),
		mapLike:         make(map[string]*wrappedTask),
		beingDispatched: make(map[string]*wrappedTask),
		getNow:          common.GetNowImpl{},
		timer:           common.NewTimerReal(),
	}
}

func (r *HeapRepository) AddTask(param scheduler.TaskParam) (scheduler.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task := param.ToTask(r.isMilliSecPrecise)
	task.CreatedAt = r.getNow.GetNow()
	task.Id = uuid.NewString()

	if r.isMilliSecPrecise {
		task = task.DropMicros()
	}

	wrapped := &wrappedTask{
		Task: task,
	}

	r.heap.Push(wrapped)
	r.mapLike[wrapped.Id] = wrapped

	if wrapped.Index == 0 {
		r.resetTimer()
	}

	return task, nil
}

func (r *HeapRepository) GetById(id string) (scheduler.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wrapped, ok := r.mapLike[id]
	if !ok {
		return scheduler.Task{}, &scheduler.RepositoryError{Id: id, Kind: scheduler.IdNotFound}
	}
	task := wrapped.Task
	return task, nil
}

func (r *HeapRepository) Update(id string, param scheduler.TaskParam) (updated bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	wrapped, ok := r.mapLike[id]
	if !ok {
		return false, &scheduler.RepositoryError{Id: id, Kind: scheduler.IdNotFound}
	}

	if errKind := errKind(wrapped.Task, errKindOption{}); errKind != "" {
		return false, &scheduler.RepositoryError{Id: id, Kind: errKind}
	}

	old := wrapped.Task

	wrapped.Task = wrapped.Task.Update(param, r.isMilliSecPrecise)

	oldIndex := wrapped.Index
	r.heap.Fix(wrapped.Index)
	newIndex := wrapped.Index

	if oldIndex != newIndex && (oldIndex == 0 || newIndex == 0) {
		r.resetTimer()
	}

	return !wrapped.Task.Equal(old), nil
}

func (r *HeapRepository) Cancel(id string) (cancelled bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	wrapped, ok := r.mapLike[id]
	if !ok {
		return false, &scheduler.RepositoryError{Id: id, Kind: scheduler.IdNotFound}
	}

	if errKind := errKind(wrapped.Task, errKindOption{
		skipCancelledAt: true,
	}); errKind != "" {
		return false, &scheduler.RepositoryError{Id: id, Kind: errKind}
	}

	if wrapped.Task.CancelledAt != nil {
		return false, nil
	}

	wrapped.Task.CancelledAt = util.Escape(r.getNow.GetNow())

	if r.isMilliSecPrecise {
		wrapped.Task = wrapped.Task.DropMicros()
	}

	if wrapped.Index >= 0 {
		index := wrapped.Index
		r.heap.Remove(index)
		if index == 0 {
			r.resetTimer()
		}
	}
	return true, nil
}

func (r *HeapRepository) MarkAsDispatched(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	wrapped, ok := r.mapLike[id]
	if !ok {
		return &scheduler.RepositoryError{Id: id, Kind: scheduler.IdNotFound}
	}

	if errKind := errKind(wrapped.Task, errKindOption{}); errKind != "" {
		return &scheduler.RepositoryError{Id: id, Kind: errKind}
	}
	wrapped.Task.DispatchedAt = util.Escape(r.getNow.GetNow())
	if r.isMilliSecPrecise {
		wrapped.Task = wrapped.Task.DropMicros()
	}

	r.beingDispatched[wrapped.Id] = wrapped

	if wrapped.Index >= 0 {
		index := wrapped.Index
		r.heap.Remove(index)
		if index == 0 {
			r.resetTimer()
		}
	}

	return nil
}

func (r *HeapRepository) MarkAsDone(id string, err error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	wrapped, ok := r.mapLike[id]
	if !ok {
		return &scheduler.RepositoryError{Id: id, Kind: scheduler.IdNotFound}
	}

	if errKind := errKind(wrapped.Task, errKindOption{
		returnOnEmptyDispatchedAt: true,
		skipDispatchedAt:          true,
	}); errKind != "" {
		return &scheduler.RepositoryError{Id: id, Kind: errKind}
	}

	delete(r.beingDispatched, id)

	wrapped.Task.DoneAt = util.Escape(r.getNow.GetNow())
	if err != nil {
		wrapped.Task.Err = err.Error()
	}

	if r.isMilliSecPrecise {
		wrapped.Task = wrapped.Task.DropMicros()
	}
	return nil
}

func (r *HeapRepository) GetNext() (scheduler.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var peeked *wrappedTask
	for {
		peeked = r.heap.Peek()
		if peeked == nil {
			r.timer.Stop()
			return scheduler.Task{}, &scheduler.RepositoryError{Kind: scheduler.Empty}
		}
		if peeked.CancelledAt != nil {
			r.timer.Stop()
			r.heap.Pop()
		} else {
			break
		}
	}
	r.resetTimer()
	return peeked.Task, nil
}

func (r *HeapRepository) StartTimer() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.isTimerStarted = true

	r.resetTimer()
}

func (r *HeapRepository) resetTimer() {
	if r.isTimerStarted {
		if peeked := r.heap.Peek(); peeked != nil {
			r.timer.Reset(peeked.ScheduledAt.Sub(r.getNow.GetNow()))
		}
	}
}

func (r *HeapRepository) StopTimer() {
	r.mu.Lock()
	r.isTimerStarted = false
	r.mu.Unlock()

	if !r.timer.Stop() {
		select {
		case <-r.timer.C():
		default:
		}
	}
}

func (r *HeapRepository) TimerChannel() <-chan time.Time {
	return r.timer.C()
}

func (r *HeapRepository) RemoveCancelled() []scheduler.Task {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.timer.Stop()
	defer r.resetTimer()

	var count int
	removed := make([]scheduler.Task, 0)
	for k, v := range r.mapLike {
		if count >= 10_000 {
			break
		}

		if v.CancelledAt != nil {
			delete(r.mapLike, k)
			if v.Index >= 0 {
				r.heap.Remove(v.Index)
			}
		}
		removed = append(removed, v.Task)
		count++
	}

	return removed
}

type errKindOption struct {
	skipCancelledAt           bool
	skipDispatchedAt          bool
	skipDoneAt                bool
	returnOnEmptyDispatchedAt bool
}

func errKind(t scheduler.Task, option errKindOption) scheduler.RepositoryErrorKind {
	if !option.skipDoneAt && t.DoneAt != nil {
		return scheduler.AlreadyDone
	}
	if !option.skipCancelledAt && t.CancelledAt != nil {
		return scheduler.AlreadyCancelled
	}
	if !option.skipDispatchedAt && t.DispatchedAt != nil {
		return scheduler.AlreadyDispatched
	}

	if option.returnOnEmptyDispatchedAt && t.DispatchedAt == nil {
		return scheduler.NotDispatched
	}

	return ""
}
