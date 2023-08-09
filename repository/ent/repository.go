package ent

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/google/uuid"
	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/gokugen/repository/ent/gen"
	"github.com/ngicks/gokugen/repository/ent/gen/predicate"
	"github.com/ngicks/gokugen/repository/ent/gen/task"
	"github.com/ngicks/gokugen/scheduler"
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
	// other than those, use schema default. see ./schema/task.go

	created, err := builder.Save(ctx)
	if err != nil {
		return def.Task{}, err
	}

	return mapEntToDefTask(created), nil
}

func (c *Core) GetById(ctx context.Context, id string) (def.Task, error) {
	t, err := c.client.Task.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return def.Task{}, &scheduler.RepositoryError{Kind: scheduler.IdNotFound, Id: id, Raw: err}
		}
		return def.Task{}, err
	}
	return mapEntToDefTask(t), nil
}

func (c *Core) UpdateById(ctx context.Context, id string, param def.TaskParam) error {
	builder := c.client.Task.UpdateOneID(id).Where(task.StateEQ(task.DefaultState))
	if param.ScheduledAt.IsSome() {
		builder = builder.SetScheduledAt(param.ScheduledAt.Value())
	}
	if param.WorkId.IsSome() {
		builder = builder.SetWorkID(param.WorkId.Value())
	}
	if param.Param.IsSome() {
		for k, v := range param.Param.Value() {
			builder = builder.Modify(func(u *sql.UpdateBuilder) {
				sqljson.Append(u, task.FieldParam, []string{v.Val}, sqljson.Path(k))
			})
		}
	}
	if param.Priority.IsSome() {
		builder = builder.SetPriority(param.Priority.Value())
	}
	if param.Meta.IsSome() {
		for k, v := range param.Meta.Value() {
			builder = builder.Modify(func(u *sql.UpdateBuilder) {
				switch v.OpTy {
				case def.MapUpdateRemoveKey:
					sqljson.Append(u, task.FieldMeta, []any{nil}, sqljson.Path(k))
				case def.MapUpdateSetKey:
					sqljson.Append(u, task.FieldMeta, []string{v.Val}, sqljson.Path(k))
				}
			})
		}
	}

	err := builder.Exec(ctx)
	if gen.IsNotFound(err) {
		t, err := c.GetById(ctx, id)
		if err != nil {
			return err
		}
		return def.ErrKindUpdate(t)
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *Core) Update(
	ctx context.Context,
	matcher def.SearchMatcher,
	param def.TaskParam,
) (rowsAffected int, err error) {

}

func (c *Core) Cancel(ctx context.Context, id string) error {
	err := c.client.Task.
		UpdateOneID(id).
		Where(task.StateEQ(task.StateScheduled)).
		SetState(task.StateCancelled).
		SetCancelledAt(c.clock.Now()).
		Exec(ctx)

	if gen.IsNotFound(err) {
		t, err := c.GetById(ctx, id)
		if err != nil {
			return err
		}
		return def.ErrKindCancel(t)
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *Core) MarkAsDispatched(ctx context.Context, id string) error {
	err := c.client.Task.
		UpdateOneID(id).
		Where(task.StateEQ(task.StateScheduled)).
		SetState(task.StateDispatched).
		SetDispatchedAt(c.clock.Now()).
		Exec(ctx)

	if gen.IsNotFound(err) {
		t, err := c.GetById(ctx, id)
		if err != nil {
			return err
		}
		return def.ErrKindMarkAsDispatch(t)
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *Core) MarkAsDone(ctx context.Context, id string, err error) error {
	builder := c.client.Task.
		UpdateOneID(id).
		Where(task.StateEQ(task.StateDispatched))

	if err == nil {
		builder = builder.SetState(task.StateDone).
			SetDoneAt(c.clock.Now())
	} else {
		builder = builder.SetState(task.StateErr).
			SetDoneAt(c.clock.Now()).
			SetErr(err.Error())
	}

	updateErr := builder.Exec(ctx)

	if gen.IsNotFound(updateErr) {
		t, err := c.GetById(ctx, id)
		if err != nil {
			return err
		}
		return def.ErrKindMarkAsDone(t)
	}
	if updateErr != nil {
		return updateErr
	}
	return nil
}

func (c *Core) Find(ctx context.Context, matcher def.SearchMatcher, offset, limit int) ([]def.Task, error) {
	matcher.TaskParam = matcher.TaskParam.TruncTime()

	builder := c.client.Task.Query()

	if matcher.WorkId.IsSome() {
		builder = builder.Where(task.WorkIDEQ(matcher.WorkId.Value()))
	}
	if matcher.Priority.IsSome() {
		builder = builder.Where(task.PriorityEQ(matcher.Priority.Value()))
	}
	if matcher.State.IsSome() {
		builder = builder.Where(task.StateEQ(task.State(matcher.State.Value())))
	}
	if matcher.ScheduledAt.IsSome() {
		builder = builder.Where(task.ScheduledAtEQ(matcher.ScheduledAt.Value()))
	}
	if matcher.CreatedAt.IsSome() {
		builder = builder.Where(task.CreatedAtEQ(matcher.CreatedAt.Value()))
	}
	if matcher.CancelledAt.IsSome() {
		cancelledAt := matcher.CancelledAt.Value()
		if cancelledAt.IsSome() {
			builder = builder.Where(task.CancelledAtEQ(cancelledAt.Value()))
		} else {
			builder = builder.Where(task.CancelledAtIsNil())
		}
	}
	if matcher.DispatchedAt.IsSome() {
		dispatchedAt := matcher.DispatchedAt.Value()
		if dispatchedAt.IsSome() {
			builder = builder.Where(task.DispatchedAtEQ(dispatchedAt.Value()))
		} else {
			builder = builder.Where(task.DispatchedAtIsNil())
		}
	}
	if matcher.DoneAt.IsSome() {
		doneAt := matcher.DoneAt.Value()
		if doneAt.IsSome() {
			builder = builder.Where(task.DoneAtEQ(doneAt.Value()))
		} else {
			builder = builder.Where(task.DoneAtIsNil())
		}
	}
	if matcher.Err.IsSome() {
		builder = builder.Where(task.ErrEQ(matcher.Err.Value()))
	}

	if matcher.Param.IsSome() {
		param := matcher.Param.Value()
		if len(param) > 0 {
			builder = builder.Where(jsonMatcher(task.FieldParam, param))
		}
	}
	if matcher.Meta.IsSome() {
		meta := matcher.Meta.Value()
		if len(meta) > 0 {
			builder = builder.Where(jsonMatcher(task.FieldMeta, meta))
		}
	}

	tasks, err := builder.Offset(offset).Limit(limit).All(ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]def.Task, len(tasks))
	for idx, t := range tasks {
		ret[idx] = mapEntToDefTask(t)
	}
	return ret, nil
}

func (c *Core) GetNext(ctx context.Context) (def.Task, error) {
	t, err := c.client.Task.
		Query().
		Where(task.StateEQ(task.DefaultState)).
		Order(task.ByScheduledAt(sql.OrderAsc()), task.ByPriority(sql.OrderDesc())).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return def.Task{}, &scheduler.RepositoryError{Kind: scheduler.Empty, Raw: err}
		}
		return def.Task{}, err
	}
	return mapEntToDefTask(t), nil
}

func mapEntToDefTask(created *gen.Task) def.Task {
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
	}
}

func mapPointerToOption[T any](v *T) option.Option[T] {
	if v == nil {
		return option.None[T]()
	} else {
		return option.Some[T](*v)
	}
}

func derefOrZero[T any](v *T) T {
	if v == nil {
		var zero T
		return zero
	} else {
		return *v
	}
}

func jsonMatcher(field string, matchers []def.MapMatcher) predicate.Task {
	return func(s *sql.Selector) {
		for _, kv := range matchers {
			var pred *sql.Predicate
			switch kv.MatchTy {
			case def.MapMatcherHasKey:
				pred = sqljson.HasKey(
					field,
					sqljson.Path(kv.Key),
				)
			case def.MapMatcherExact:
				pred = sqljson.ValueEQ(
					field,
					kv.Value,
					sqljson.Path(kv.Key),
				)
			case def.MapMatcherForward:
				pred = sqljson.StringHasPrefix(
					field,
					kv.Value,
					sqljson.Path(kv.Key),
				)
			case def.MapMatcherBackward:
				pred = sqljson.StringHasSuffix(
					field,
					kv.Value,
					sqljson.Path(kv.Key),
				)
			case def.MapMatcherMiddle:
				pred = sqljson.StringContains(
					field,
					kv.Value,
					sqljson.Path(kv.Key),
				)
			}
			s.Where(pred)
		}
	}
}
