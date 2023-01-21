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
		// let it panic here, with out of range message.
		_ = (*(*[]*wrappedTask)(slice))[slice.Len()-1]
	}
	popped.Index = -1
	return popped
}

type taskMap struct {
	Scheduled map[string]*wrappedTask
	Done      map[string]*wrappedTask
	Cancelled map[string]*wrappedTask
}

func newTaskMap() taskMap {
	return taskMap{
		Scheduled: make(map[string]*wrappedTask),
		Done:      make(map[string]*wrappedTask),
		Cancelled: make(map[string]*wrappedTask),
	}
}

func fromExternal(tm TaskMap) taskMap {
	ret := newTaskMap()

	for id, task := range tm.Scheduled {
		ret.Scheduled[id] = &wrappedTask{Task: task, Index: -1}
	}
	for id, task := range tm.Cancelled {
		ret.Cancelled[id] = &wrappedTask{Task: task, Index: -1}
	}
	for id, task := range tm.Done {
		ret.Done[id] = &wrappedTask{Task: task, Index: -1}
	}
	return ret
}

func (tm taskMap) Dump() TaskMap {
	ret := TaskMap{
		Scheduled: make(map[string]scheduler.Task, len(tm.Scheduled)),
		Cancelled: make(map[string]scheduler.Task, len(tm.Scheduled)),
		Done:      make(map[string]scheduler.Task, len(tm.Scheduled)),
	}
	for id, task := range tm.Scheduled {
		ret.Scheduled[id] = task.Task
	}
	for id, task := range tm.Cancelled {
		ret.Cancelled[id] = task.Task
	}
	for id, task := range tm.Done {
		ret.Done[id] = task.Task
	}
	return ret
}

func (tm taskMap) IsZero() bool {
	return tm.Scheduled == nil || tm.Cancelled == nil || tm.Done == nil
}

func (tm *taskMap) Get(id string) (task *wrappedTask, ok bool) {
	if t, ok := tm.Scheduled[id]; ok {
		return t, true
	}
	if t, ok := tm.Done[id]; ok {
		return t, true
	}
	if t, ok := tm.Cancelled[id]; ok {
		return t, true
	}
	return nil, false
}
func (tm *taskMap) Add(task *wrappedTask) {
	tm.Scheduled[task.Id] = task
}
func (tm *taskMap) SetCancelled(id string, now time.Time) {
	t, ok := tm.Scheduled[id]
	if ok {
		t.CancelledAt = util.Escape(now)
		tm.Cancelled[id] = t
		delete(tm.Scheduled, id)
	}
}
func (tm *taskMap) SetDone(id string, now time.Time, err error) {
	t, ok := tm.Scheduled[id]
	if ok {
		t.DoneAt = util.Escape(now)
		if err != nil {
			t.Err = err.Error()
		}
		tm.Done[id] = t
		delete(tm.Scheduled, id)
	}
}
func (tm *taskMap) Delete(id string) {
	delete(tm.Scheduled, id)
	delete(tm.Cancelled, id)
	delete(tm.Done, id)
}
func (tm *taskMap) RemoveDone() {
	tm.Done = make(map[string]*wrappedTask)
}
func (tm *taskMap) RemoveCancelled() {
	tm.Cancelled = make(map[string]*wrappedTask)
}

type HeapRepository struct {
	mu sync.RWMutex

	isMilliSecPrecise bool // if true, all internal times will be truncated to multiple of milli sec. mainly for testing.

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

	if taskMap.IsZero() {
		h.taskMap = newTaskMap()
		return h
	}

	h.taskMap = fromExternal(taskMap)
	for _, task := range h.taskMap.Scheduled {
		h.heap.Push(task)
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

func (r *HeapRepository) Update(id string, param scheduler.TaskParam) (updated bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	wrapped, ok := r.taskMap.Get(id)
	if !ok {
		return false, &scheduler.RepositoryError{Id: id, Kind: scheduler.IdNotFound}
	}

	if err := ErrKindUpdate(wrapped.Task); err != nil {
		return false, err
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

func (r *HeapRepository) remove(done bool) map[string]scheduler.Task {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.timer.Stop()
	defer r.resetTimer()

	var target map[string]*wrappedTask
	if done {
		target = r.taskMap.Done
	} else {
		target = r.taskMap.Cancelled
	}

	ret := make(map[string]scheduler.Task, len(r.taskMap.Cancelled))
	for id, task := range target {
		ret[id] = task.Task
	}

	if done {
		r.taskMap.RemoveDone()
	} else {
		r.taskMap.RemoveCancelled()
	}

	return ret
}

func (r *HeapRepository) RemoveCancelled() map[string]scheduler.Task {
	return r.remove(false)
}

func (r *HeapRepository) RemoveDone() map[string]scheduler.Task {
	return r.remove(true)
}

type TaskMap struct {
	Scheduled map[string]scheduler.Task
	Cancelled map[string]scheduler.Task
	Done      map[string]scheduler.Task
}

func (tm TaskMap) IsZero() bool {
	return tm.Scheduled == nil || tm.Cancelled == nil || tm.Done == nil
}

func (r *HeapRepository) Dump() TaskMap {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.taskMap.Dump()
}
