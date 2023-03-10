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
	return i.Task.Less(j.Task)
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
		// let it panic here, with out of range message.
		_ = (*(*[]*wrappedTask)(slice))[slice.Len()-1]
	}
	popped.Index = -1
	return popped
}

type HeapRepository struct {
	mu sync.RWMutex

	// if true, all internal times will be truncated to multiple of milli sec. mainly for testing.
	isMilliSecPrecise bool

	heap           *heap.FilterableHeap[*wrappedTask]
	taskMap        taskMap
	getNow         common.NowGetter
	timer          common.Timer
	isTimerStarted bool
}

func newHeapRepository(taskMap TaskMap) *HeapRepository {
	h := &HeapRepository{
		heap:   heap.NewFilterableHeap[*wrappedTask](),
		getNow: common.NowGetterReal{},
		timer:  common.NewTimerReal(),
	}

	if !taskMap.IsInitialized() {
		h.taskMap = newTaskMap()
		return h
	}

	h.taskMap = fromExternal(taskMap)
	for _, task := range h.taskMap.Scheduled {
		if task.DispatchedAt == nil {
			// keeping invariants where MarkAsDispatched removes element from heap.
			h.heap.Push(task)
		}
	}
	return h
}

func NewHeapRepository() *HeapRepository {
	return newHeapRepository(TaskMap{})
}

func NewHeapRepositoryFromMap(taskMap TaskMap) *HeapRepository {
	return newHeapRepository(taskMap)
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
	r.taskMap.Add(wrapped)

	if wrapped.Index == 0 {
		r.resetTimer()
	}

	return task, nil
}

func (r *HeapRepository) GetById(id string) (scheduler.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wrapped, ok := r.taskMap.Get(id)
	if !ok {
		return scheduler.Task{}, &scheduler.RepositoryError{Id: id, Kind: scheduler.IdNotFound}
	}
	task := wrapped.Task
	return task, nil
}

func (r *HeapRepository) Delete(id string) (deleted bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	wrapped, ok := r.taskMap.Get(id)
	if !ok {
		return false, nil
	}
	if wrapped.Index >= 0 {
		r.heap.Remove(wrapped.Index)
	}
	r.taskMap.Delete(id)
	return true, nil
}

func (r *HeapRepository) Find(t scheduler.TaskMatcher) ([]scheduler.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.taskMap.Find(t), nil
}

func (r *HeapRepository) FindMetaContain(matcher []scheduler.KeyValuePairMatcher) ([]scheduler.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.taskMap.FindMetaContain(matcher), nil
}

func (r *HeapRepository) Update(id string, param scheduler.TaskParam) (updated bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	wrapped, ok := r.taskMap.Get(id)
	if !ok {
		return false, &scheduler.RepositoryError{Id: id, Kind: scheduler.IdNotFound}
	}

	if !param.HasOnlyMeta() {
		err := ErrKindUpdate(wrapped.Task)
		if err != nil {
			return false, err
		}
	}

	old := wrapped.Task

	wrapped.Task = wrapped.Task.Update(param, r.isMilliSecPrecise)

	if wrapped.Index >= 0 {
		oldIndex := wrapped.Index
		r.heap.Fix(wrapped.Index)
		newIndex := wrapped.Index

		if oldIndex != newIndex && (oldIndex == 0 || newIndex == 0) {
			r.resetTimer()
		}
	}

	return !wrapped.Task.Equal(old), nil
}

func (r *HeapRepository) Cancel(id string) (cancelled bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	wrapped, ok := r.taskMap.Get(id)
	if !ok {
		return false, &scheduler.RepositoryError{Id: id, Kind: scheduler.IdNotFound}
	}

	if err := ErrKindCancel(wrapped.Task); err != nil {
		return false, err
	}

	if wrapped.Task.CancelledAt != nil {
		return false, nil
	}

	r.taskMap.SetCancelled(id, r.getNow.GetNow())

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

	wrapped, ok := r.taskMap.Get(id)
	if !ok {
		return &scheduler.RepositoryError{Id: id, Kind: scheduler.IdNotFound}
	}

	if err := ErrKindMarkAsDispatch(wrapped.Task); err != nil {
		return err
	}

	wrapped.Task.DispatchedAt = util.Escape(r.getNow.GetNow())
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

	return nil
}

func (r *HeapRepository) MarkAsDone(id string, err error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	wrapped, ok := r.taskMap.Get(id)
	if !ok {
		return &scheduler.RepositoryError{Id: id, Kind: scheduler.IdNotFound}
	}

	if err := ErrKindMarkAsDone(wrapped.Task); err != nil {
		return err
	}

	r.taskMap.SetDone(id, r.getNow.GetNow(), err)

	if r.isMilliSecPrecise {
		wrapped.Task = wrapped.Task.DropMicros()
	}
	return nil
}

func (r *HeapRepository) GetNext() (scheduler.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	next := r.heap.Peek()
	if next == nil {
		r.timer.Stop()
		return scheduler.Task{}, &scheduler.RepositoryError{Kind: scheduler.Empty}
	}

	return next.Task, nil
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

func (r *HeapRepository) Remove(done, cancelled, deleted bool) TaskMap {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.timer.Stop()
	defer r.resetTimer()

	var out TaskMap
	if done {
		out.Done = cloneUnwrapping(r.taskMap.RemoveDone())
	}
	if cancelled {
		out.Cancelled = cloneUnwrapping(r.taskMap.RemoveCancelled())
	}
	if deleted {
		out.Deleted = cloneUnwrapping(r.taskMap.RemoveDeleted())
	}

	return out
}

func (r *HeapRepository) Dump() TaskMap {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.taskMap.Dump()
}
