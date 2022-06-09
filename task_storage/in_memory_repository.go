package taskstorage

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"
)

var (
	ErrNoEnt             = errors.New("no ent")
	ErrNotUpdatableState = errors.New("not updatable")
)

var _ RepositoryUpdater = &InMemoryRepo{}

type ent struct {
	mu   sync.Mutex
	info TaskInfo
}

func (e *ent) Update(new TaskState, updateIf func(old TaskState) bool) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if updateIf(e.info.State) {
		e.info.State = new
		return true
	}
	return false
}

type InMemoryRepo struct {
	randomStr *RandStringGenerator
	store     *sync.Map
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{
		randomStr: NewRandStringGenerator(int64(time.Now().Nanosecond()), 32, hex.NewEncoder),
		store:     new(sync.Map),
	}
}

func (r *InMemoryRepo) Insert(taskInfo TaskInfo) (taskId string, err error) {
	for {
		taskId, err = r.randomStr.Generate()
		if err != nil {
			return
		}

		taskInfo.Id = taskId

		_, loaded := r.store.LoadOrStore(taskId, &ent{info: taskInfo})

		if !loaded {
			break
		}
	}
	return
}

func (r *InMemoryRepo) GetAll() ([]TaskInfo, error) {
	arr := make([]TaskInfo, 0)
	r.store.Range(func(key, value any) bool {
		arr = append(arr, value.(*ent).info)
		return true
	})

	return arr, nil
}

func (r *InMemoryRepo) GetById(taskId string) (TaskInfo, error) {
	val, ok := r.store.Load(taskId)
	if !ok {
		return TaskInfo{}, fmt.Errorf("%w: no such id [%s]", ErrNoEnt, taskId)
	}
	return val.(*ent).info, nil
}

func (r *InMemoryRepo) MarkAsDone(id string) (ok bool, err error) {
	return updateState(r.store, id, Done)
}
func (r *InMemoryRepo) MarkAsCancelled(id string) (ok bool, err error) {
	return updateState(r.store, id, Cancelled)
}
func (r *InMemoryRepo) MarkAsFailed(id string) (ok bool, err error) {
	return updateState(r.store, id, Failed)
}

func (r *InMemoryRepo) UpdateState(id string, old, new TaskState) (swapped bool, err error) {
	val, ok := r.store.Load(id)
	if !ok {
		return false, fmt.Errorf("%w: no such id [%s]", ErrNoEnt, id)
	}
	entry := val.(*ent)
	return entry.Update(new, func(old_ TaskState) bool { return old_ == old }), nil
}

func updateState(store *sync.Map, id string, state TaskState) (bool, error) {
	val, ok := store.Load(id)
	if !ok {
		return false, fmt.Errorf("%w: no such id [%s]", ErrNoEnt, id)
	}
	entry := val.(*ent)
	return entry.Update(state, isUpdatable), nil
}

func (r *InMemoryRepo) Update(id string, diff UpdateDiff) error {
	val, ok := r.store.Load(id)
	if !ok {
		return fmt.Errorf("%w: no such id [%s]", ErrNoEnt, id)
	}

	entry := val.(*ent)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if !isUpdatable(entry.info.State) {
		return fmt.Errorf("%w: is now %s", ErrNotUpdatableState, entry.info.State)
	}

	if diff.WorkId != nil {
		entry.info.WorkId = *diff.WorkId
	}
	if diff.Param != nil {
		entry.info.Param = *diff.Param
	}
	if diff.ScheduledTime != nil {
		entry.info.ScheduledTime = *diff.ScheduledTime
	}
	if diff.State != nil {
		entry.info.State = *diff.State
	}
	return nil
}

func isUpdatable(state TaskState) bool {
	return !(state == Done || state == Cancelled || state == Failed)
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
				return bytes.NewBuffer(make([]byte, 32))
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
