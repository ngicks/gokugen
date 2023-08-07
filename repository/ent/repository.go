package ent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/gokugen/repository/ent/gen"
	"github.com/ngicks/mockable"
	"github.com/ngicks/und/option"
)

var _ def.Repository = (*Core)(nil)

type RandStrGen func() string

type Core struct {
	client     *gen.Client
	randStrGen RandStrGen
	clock      mockable.Clock
}

func NewCore(client *gen.Client) *Core {
	return &Core{
		client:     client,
		randStrGen: uuid.NewString,
		clock:      mockable.NewClockReal(),
	}
}

func (c *Core) AddTask(ctx context.Context, param def.TaskParam) (def.Task, error) {
	t := param.ToTask(true)
	t.Id = c.randStrGen()
	t.CreatedAt = c.clock.Now()

	if !t.IsValid() {
		return def.Task{},
			fmt.Errorf("%w. reason = %v", def.ErrInvalidTask, t.ReportInvalidity())
	}

	builder := c.client.Task.Create().
		SetID(t.Id).
		SetWorkID(t.WorkId).
		SetPriority(t.Priority).
		SetScheduledAt(t.ScheduledAt).
		SetCreatedAt(t.CreatedAt)
	if t.Param != nil {
		builder = builder.SetParam(t.Param)
	}
	if t.Meta != nil {
		builder = builder.SetMeta(t.Meta)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return def.Task{}, err
	}

	return def.Task{
		Id:           created.ID,
		WorkId:       created.WorkID,
		Param:        created.Param,
		Priority:     created.Priority,
		State:        def.State(created.State),
		ScheduledAt:  created.ScheduledAt,
		CreatedAt:    created.CreatedAt,
		CancelledAt:  mapPointerToOption(created.CancelledAt),
		DispatchedAt: mapPointerToOption(created.DispatchedAt),
		DoneAt:       mapPointerToOption(created.DoneAt),
		Err:          derefOrZero(created.Err),
		Meta:         created.Meta,
	}, nil
}

func (c *Core) GetById(ctx context.Context, id string) (def.Task, error)
func (c *Core) Update(ctx context.Context, id string, param def.TaskParam) (updated bool, err error)
func (c *Core) Cancel(ctx context.Context, id string) (cancelled bool, err error)
func (c *Core) MarkAsDispatched(ctx context.Context, id string) error
func (c *Core) MarkAsDone(ctx context.Context, id string, err error) error
func (c *Core) Find(ctx context.Context, matcher def.SearchMatcher) ([]def.Task, error)
func (c *Core) GetNext(ctx context.Context) (def.Task, error)

func derefOrZero[T any](v *T) T {
	if v == nil {
		var zero T
		return zero
	} else {
		return *v
	}
}

func mapPointerToOption[T any](v *T) option.Option[T] {
	if v == nil {
		return option.None[T]()
	} else {
		return option.Some[T](*v)
	}
}
