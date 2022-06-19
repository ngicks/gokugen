package repository

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/ngicks/gokugen/common"
	taskstorage "github.com/ngicks/gokugen/task_storage"
	syncparam "github.com/ngicks/type-param-common/sync-param"
)

var _ taskstorage.RepositoryUpdater = &InMemoryRepo{}

type ent struct {
	mu   sync.Mutex
	info taskstorage.TaskInfo
}

func (e *ent) Update(new taskstorage.TaskState, updateIf func(old taskstorage.TaskState) bool, getNow common.GetNow) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if updateIf(e.info.State) {
		e.info.State = new
		e.info.LastModified = getNow.GetNow()
		return true
	}
	return false
}

func (e *ent) UpdateByDiff(diff taskstorage.UpdateDiff, getNow common.GetNow) bool {
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
	getNow    common.GetNow
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{
		randomStr: NewRandStringGenerator(int64(time.Now().Nanosecond()), 16, hex.NewEncoder),
		store:     new(syncparam.Map[string, *ent]),
		getNow:    common.GetNowImpl{},
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
		arr = append(arr, value.info)
		return true
	})

	return arr, nil
}

func (r *InMemoryRepo) GetUpdatedAfter(since time.Time) ([]taskstorage.TaskInfo, error) {
	results := make([]taskstorage.TaskInfo, 0)
	r.store.Range(func(key string, entry *ent) bool {
		entry.mu.Lock()
		defer entry.mu.Unlock()
		if entry.info.LastModified.After(since) {
			results = append(results, entry.info)
		}
		return true
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

func updateState(store *syncparam.Map[string, *ent], id string, state taskstorage.TaskState, getNow common.GetNow) (bool, error) {
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

	}

	return nil
}

func isUpdatable(state taskstorage.TaskState) bool {
	return !(state == taskstorage.Done || state == taskstorage.Cancelled || state == taskstorage.Failed)
}

type RandStringGenerator struct {
	randMu         sync.Mutex
	rand           *rand.Rand
	byteLen        uint
	bufPool        sync.Pool
	encoderFactory func(r io.Writer) io.Writer
}

func NewRandStringGenerator(seed int64, byteLen uint, encoderFactory func(r io.Writer) io.Writer) *RandStringGenerator {
	if encoderFactory == nil {
		encoderFactory = hex.NewEncoder
	}
	return &RandStringGenerator{
		rand:    rand.New(rand.NewSource(seed)),
		byteLen: byteLen,
		bufPool: sync.Pool{
			New: func() any {
				buf := bytes.NewBuffer(make([]byte, 32))
				buf.Reset()
				return buf
			},
		},
		encoderFactory: encoderFactory,
	}
}

func (f *RandStringGenerator) Generate() (randomStr string, err error) {
	buf := f.bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		f.bufPool.Put(buf)
	}()

	encoder := f.encoderFactory(buf)

	f.randMu.Lock()
	_, err = io.CopyN(encoder, f.rand, int64(f.byteLen))
	f.randMu.Unlock()

	if cl, ok := encoder.(io.Closer); ok {
		cl.Close()
	}

	if err != nil {
		return
	}
	return buf.String(), nil
}
