package ent

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/google/uuid"
	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/gokugen/repository/ent/gen"
	"github.com/ngicks/gokugen/repository/ent/gen/predicate"
	"github.com/ngicks/gokugen/repository/ent/gen/task"
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

func (c *Core) Close() error {
	return c.client.Close()
}

func (c *Core) AddTask(ctx context.Context, param def.TaskUpdateParam) (def.Task, error) {
	t := param.ToTask(c.randStrGen(), c.clock.Now())

	if !t.IsValid() {
		return def.Task{},
			fmt.Errorf("%w. reason = %v", def.ErrInvalidTask, t.ReportInvalidity())
	}

	builder := c.client.Task.Create().
		SetID(t.Id).
		SetWorkID(t.WorkId).
		SetPriority(t.Priority).
		SetState(task.State(t.State)).
		SetScheduledAt(t.ScheduledAt).
		SetCreatedAt(t.CreatedAt).
		SetNillableDeadline(t.Deadline.Plain())
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
			return def.Task{},
				&def.RepositoryError{Kind: def.IdNotFound, Id: id, Raw: err}
		}
		return def.Task{}, err
	}
	return mapEntToDefTask(t), nil
}

var fakeTask = def.Task{
	Id:       def.NeverExistentId,
	WorkId:   "foo",
	Param:    map[string]string{"foo": "bar"},
	Priority: 0,
	State:    def.TaskScheduled,
	ScheduledAt: time.Date(
		2023,
		time.April,
		23,
		19,
		28,
		59,
		123000000,
		time.UTC,
	),
	CreatedAt: time.Date(
		2023,
		time.April,
		22,
		19,
		28,
		59,
		123000000,
		time.UTC,
	),
	Meta: map[string]string{"baz": "qux"},
}

func (c *Core) UpdateById(ctx context.Context, id string, param def.TaskUpdateParam) error {
	if !fakeTask.Update(param).IsValid() {
		return fmt.Errorf("%w: update to invalid state is not allowed", def.ErrInvalidTask)
	}
	param = param.Normalize()

	builder := c.client.Task.UpdateOneID(id).Where(task.StateEQ(task.DefaultState))
	if param.WorkId.IsSome() {
		builder = builder.SetWorkID(param.WorkId.Value())
	}
	if param.Param.IsSome() {
		paramParam := param.Param.Value()
		if paramParam == nil {
			paramParam = map[string]string{}
		}
		builder = builder.SetParam(paramParam)
	}
	if param.Priority.IsSome() {
		builder = builder.SetPriority(param.Priority.Value())
	}
	if param.ScheduledAt.IsSome() {
		builder = builder.SetScheduledAt(param.ScheduledAt.Value())
	}
	if param.Deadline.IsSome() {
		deadline := param.Deadline.Value()
		if deadline.IsSome() {
			builder = builder.SetDeadline(deadline.Value())
		} else {
			builder = builder.ClearDeadline()
		}
	}
	if param.Meta.IsSome() {
		meta := param.Meta.Value()
		if meta == nil {
			meta = map[string]string{}
		}
		builder = builder.SetMeta(meta)
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

func (c *Core) Cancel(ctx context.Context, id string) error {
	err := c.client.Task.
		UpdateOneID(id).
		Where(task.StateEQ(task.StateScheduled)).
		SetState(task.StateCancelled).
		SetCancelledAt(def.NormalizeTime(c.clock.Now())).
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
		SetDispatchedAt(def.NormalizeTime(c.clock.Now())).
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
			SetDoneAt(def.NormalizeTime(c.clock.Now()))
	} else {
		builder = builder.SetState(task.StateErr).
			SetDoneAt(def.NormalizeTime(c.clock.Now())).
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

func (c *Core) Find(
	ctx context.Context,
	query def.TaskQueryParam,
	offset, limit int,
) ([]def.Task, error) {
	query = query.Normalize()

	builder := where(c.client.Task.Query(), query)

	tasks, err := builder.
		Order(task.ByCreatedAt()).
		Offset(offset).
		Limit(limit).
		All(ctx)
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
		Order(
			task.ByScheduledAt(sql.OrderAsc()),
			task.ByPriority(sql.OrderDesc()),
			task.ByCreatedAt(sql.OrderAsc()),
		).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return def.Task{}, &def.RepositoryError{Kind: def.Exhausted, Raw: err}
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
		Deadline:     mapPointerToOption(created.Deadline),
		CancelledAt:  mapPointerToOption(created.CancelledAt),
		DispatchedAt: mapPointerToOption(created.DispatchedAt),
		DoneAt:       mapPointerToOption(created.DoneAt),
		Err:          created.Err,
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

func where[T interface {
	Where(ps ...predicate.Task) T
}](builder T, query def.TaskQueryParam) T {
	builder = whereFieldEq(builder, query.Id, task.IDEQ)
	builder = whereFieldEq(builder, query.WorkId, task.WorkIDEQ)
	builder = whereFieldEq(builder, query.Priority, task.PriorityEQ)
	builder = whereFieldEq(
		builder,
		query.State,
		func(v def.State) predicate.Task { return task.StateEQ(task.State(v)) },
	)
	builder = whereFieldEq(builder, query.Err, task.ErrEQ)

	builder = whereTime(builder, query.ScheduledAt, task.FieldScheduledAt)
	builder = whereTime(builder, query.CreatedAt, task.FieldCreatedAt)
	builder = whereOptTime(builder, query.Deadline, task.FieldDeadline)
	builder = whereOptTime(builder, query.CancelledAt, task.FieldCancelledAt)
	builder = whereOptTime(builder, query.DispatchedAt, task.FieldDispatchedAt)
	builder = whereOptTime(builder, query.DoneAt, task.FieldDoneAt)

	builder = whereMap(builder, query.Param, task.FieldParam)
	builder = whereMap(builder, query.Meta, task.FieldMeta)

	return builder
}

func whereFieldEq[T interface {
	Where(ps ...predicate.Task) T
}, U any](builder T, query option.Option[U], pred func(v U) predicate.Task) T {
	if query.IsNone() {
		return builder
	}
	return builder.Where(pred(query.Value()))
}

func whereTime[T interface {
	Where(ps ...predicate.Task) T
}](builder T, query option.Option[def.TimeMatcher], fieldName string) T {
	if query.IsNone() {
		return builder
	}
	return builder.Where(timePred(fieldName, query.Value()))
}

func whereOptTime[T interface {
	Where(ps ...predicate.Task) T
}](builder T, query option.Option[option.Option[def.TimeMatcher]], fieldName string) T {
	if query.IsNone() {
		return builder
	}
	if query.Value().IsNone() {
		return builder.Where(sql.FieldIsNull(fieldName))
	}
	return whereTime[T](builder, query.Value(), fieldName)
}

func timePred(fieldName string, matcher def.TimeMatcher) predicate.Task {
	switch matcher.MatchType.Get() {
	default:
		panic(fmt.Errorf("unknown matcher type: %s", matcher.MatchType.Get()))
	case def.TimeMatcherNonNull:
		return sql.FieldNotNull(fieldName)
	case def.TimeMatcherEqual:
		return sql.FieldEQ(fieldName, matcher.Value)
	case def.TimeMatcherBefore:
		return sql.FieldLT(fieldName, matcher.Value)
	case def.TimeMatcherBeforeEqual:
		return sql.FieldLTE(fieldName, matcher.Value)
	case def.TimeMatcherAfter:
		return sql.FieldGT(fieldName, matcher.Value)
	case def.TimeMatcherAfterEqual:
		return sql.FieldGTE(fieldName, matcher.Value)
	}
}

func whereMap[T interface {
	Where(ps ...predicate.Task) T
}](builder T, query option.Option[[]def.MapMatcher], fieldName string) T {
	if query.IsNone() {
		return builder
	}
	return builder.Where(jsonMatcher(fieldName, query.Value()))
}

func jsonMatcher(field string, matchers []def.MapMatcher) predicate.Task {
	return func(s *sql.Selector) {
		for _, kv := range matchers {
			var pred *sql.Predicate
			switch kv.MatchType.Get() {
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
