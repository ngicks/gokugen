package scheduler

import (
	"context"
	"sync"

	"github.com/ngicks/gokugen/def"
)

type Repository interface {
	def.Observer
	GetById(ctx context.Context, id string) (def.Task, error)
	GetNext(ctx context.Context) (def.Task, error)
	MarkAsDispatched(ctx context.Context, id string) error
	MarkAsDone(ctx context.Context, id string, err error) error
}

type VolatileTask interface {
	def.Observer
	Peek(ctx context.Context) (def.Task, error)
	Pop(ctx context.Context) (def.Task, error)
}

type VolatileTaskRepo interface {
	Repository
	UpdateTasks(v VolatileTask)
}

var _ VolatileTaskRepo = (*volatileTaskRepo)(nil)

type volatileTaskRepo struct {
	mu sync.Mutex
	VolatileTask
	record map[string]def.Task
}

func NewVolatileTaskRepo(v VolatileTask) VolatileTaskRepo {
	return newVolatileTaskRepo(v)
}

func newVolatileTaskRepo(v VolatileTask) *volatileTaskRepo {
	return &volatileTaskRepo{
		VolatileTask: v,
		record:       make(map[string]def.Task),
	}
}

func (r *volatileTaskRepo) GetById(ctx context.Context, id string) (def.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, ok := r.record[id]
	if !ok {
		return def.Task{}, &def.RepositoryError{Id: id, Kind: def.IdNotFound}
	}
	return task, nil
}
func (r *volatileTaskRepo) GetNext(ctx context.Context) (def.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, err := r.VolatileTask.Peek(ctx)
	if err != nil {
		return def.Task{}, err
	}
	r.record[p.Id] = p
	return p.Clone(), err
}
func (r *volatileTaskRepo) MarkAsDispatched(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, err := r.Peek(ctx)
	if err != nil {
		return err
	}

	if p.Id == id {
		_, err := r.Pop(ctx)
		if err != nil {
			return err
		}
		return nil
	}

	if _, ok := r.record[id]; ok {
		delete(r.record, id)
		return &def.RepositoryError{
			Id:   id,
			Kind: def.AlreadyCancelled,
		}
	}

	return nil
}

func (r *volatileTaskRepo) MarkAsDone(ctx context.Context, id string, err error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.record, id)
	return nil
}

func (r *volatileTaskRepo) UpdateTasks(v VolatileTask) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.VolatileTask = v
}
