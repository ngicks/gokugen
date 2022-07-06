package repository

import (
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	taskstorage "github.com/ngicks/gokugen/task_storage"
	"github.com/ngicks/gommon"
	syncparam "github.com/ngicks/type-param-common/sync-param"
)

var _ taskstorage.RepositoryUpdater = &InMemoryRepo{}

type ent struct {
	mu   sync.Mutex
	info taskstorage.TaskInfo
}

func (e *ent) Update(new taskstorage.TaskState, updateIf func(old taskstorage.TaskState) bool, getNow gommon.GetNower) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if updateIf(e.info.State) {
		e.info.State = new
		e.info.LastModified = getNow.GetNow()
		return true
	}
	return false
}

func (e *ent) UpdateByDiff(diff taskstorage.UpdateDiff, getNow gommon.GetNower) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !isUpdatable(e.info.State) {
		return false
	}

	e.info.LastModified = getNow.GetNow()

	if diff.UpdateKey.WorkId {
		e.info.WorkId = diff.Diff.WorkId
	}
	if diff.UpdateKey.Param {
		e.info.Param = diff.Diff.Param
	}
	if diff.UpdateKey.ScheduledTime {
		e.info.ScheduledTime = diff.Diff.ScheduledTime
	}
	if diff.UpdateKey.State {
		e.info.State = diff.Diff.State
	}
	return true
}

type InMemoryRepo struct {
	randomStr *RandStringGenerator
	store     *syncparam.Map[string, *ent]
	getNow    gommon.GetNower
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{
		randomStr: NewRandStringGenerator(int64(time.Now().Nanosecond()), 16, hex.NewEncoder),
		store:     new(syncparam.Map[string, *ent]),
		getNow:    gommon.GetNowImpl{},
	}
}

func (r *InMemoryRepo) Insert(taskInfo taskstorage.TaskInfo) (taskId string, err error) {
	for {
		taskId, err = r.randomStr.Generate()
		if err != nil {
			return
		}

		taskInfo.Id = taskId
		taskInfo.LastModified = r.getNow.GetNow()

		_, loaded := r.store.LoadOrStore(taskId, &ent{info: taskInfo})

		if !loaded {
			break
		}
	}
	return
}

func (r *InMemoryRepo) GetAll() ([]taskstorage.TaskInfo, error) {
	arr := make([]taskstorage.TaskInfo, 0)
	r.store.Range(func(key string, value *ent) bool {
		value.mu.Lock()
		defer value.mu.Unlock()
		arr = append(arr, value.info)
		return true
	})

	sort.Slice(arr, func(i, j int) bool {
		return arr[i].LastModified.Before(arr[j].LastModified)
	})
	return arr, nil
}

func (r *InMemoryRepo) GetUpdatedSince(since time.Time) ([]taskstorage.TaskInfo, error) {
	results := make([]taskstorage.TaskInfo, 0)
	r.store.Range(func(key string, entry *ent) bool {
		entry.mu.Lock()
		defer entry.mu.Unlock()
		if entry.info.LastModified.After(since) || entry.info.LastModified.Equal(since) {
			results = append(results, entry.info)
		}
		return true
	})

	sort.Slice(results, func(i, j int) bool {
		return results[i].LastModified.Before(results[j].LastModified)
	})
	return results, nil
}

func (r *InMemoryRepo) GetById(taskId string) (taskstorage.TaskInfo, error) {
	val, ok := r.store.Load(taskId)
	if !ok {
		return taskstorage.TaskInfo{}, fmt.Errorf("%w: no such id [%s]", taskstorage.ErrNoEnt, taskId)
	}
	return val.info, nil
}

func (r *InMemoryRepo) MarkAsDone(id string) (ok bool, err error) {
	return updateState(r.store, id, taskstorage.Done, r.getNow)
}
func (r *InMemoryRepo) MarkAsCancelled(id string) (ok bool, err error) {
	return updateState(r.store, id, taskstorage.Cancelled, r.getNow)
}
func (r *InMemoryRepo) MarkAsFailed(id string) (ok bool, err error) {
	return updateState(r.store, id, taskstorage.Failed, r.getNow)
}

func (r *InMemoryRepo) UpdateState(id string, old, new taskstorage.TaskState) (swapped bool, err error) {
	entry, ok := r.store.Load(id)
	if !ok {
		return false, fmt.Errorf("%w: no such id [%s]", taskstorage.ErrNoEnt, id)
	}
	return entry.Update(new, func(old_ taskstorage.TaskState) bool { return old_ == old }, r.getNow), nil
}

func updateState(store *syncparam.Map[string, *ent], id string, state taskstorage.TaskState, getNow gommon.GetNower) (bool, error) {
	entry, ok := store.Load(id)
	if !ok {
		return false, fmt.Errorf("%w: no such id [%s]", taskstorage.ErrNoEnt, id)
	}
	return entry.Update(state, isUpdatable, getNow), nil
}

func (r *InMemoryRepo) Update(id string, diff taskstorage.UpdateDiff) error {
	entry, ok := r.store.Load(id)
	if !ok {
		return fmt.Errorf("%w: no such id [%s]", taskstorage.ErrNoEnt, id)
	}

	if !entry.UpdateByDiff(diff, r.getNow) {
		return taskstorage.ErrNotUpdatableState
	}

	return nil
}

func isUpdatable(state taskstorage.TaskState) bool {
	return !(state == taskstorage.Done || state == taskstorage.Cancelled || state == taskstorage.Failed)
}
