package repository

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ngicks/gokugen/repository/gormmodel"
	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/gommon/pkg/common"
	"github.com/ngicks/type-param-common/util"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _ scheduler.RepositoryLike = (*DefaultGormCore)(nil)

type DefaultGormCore struct {
	db        *gorm.DB
	timer     common.Timer
	nowGetter common.NowGetter

	softDelete bool
}

type defaultGormOption func(g *DefaultGormCore)

// SetSoftDelete returns option func that sets g's soft deletion mode.
// true for soft deletion, and vice versa.
func SetSoftDelete(softDelete bool) defaultGormOption {
	return func(g *DefaultGormCore) {
		g.softDelete = softDelete
	}
}

func NewDefaultGormCore(db *gorm.DB, opts ...defaultGormOption) *DefaultGormCore {
	g := &DefaultGormCore{
		db:        db,
		timer:     common.NewTimerReal(),
		nowGetter: common.NowGetterReal{},
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

func (g *DefaultGormCore) AddTask(param scheduler.TaskParam) (scheduler.Task, error) {
	t := gormmodel.FromTask(param.ToTask(false))
	t.Id = uuid.NewString()

	result := g.db.Create(&t)
	if result.Error != nil {
		return scheduler.Task{}, result.Error
	}
	return t.ToTask(), nil
}

func (g *DefaultGormCore) GetById(id string) (scheduler.Task, error) {
	t := gormmodel.Task{}
	result := g.db.Where("id = ?", id).Limit(1).Find(&t)
	if result.Error != nil {
		return scheduler.Task{}, result.Error
	}
	if result.RowsAffected > 0 {
		return t.ToTask(), nil
	}
	return scheduler.Task{}, &scheduler.RepositoryError{Kind: scheduler.IdNotFound}
}

type chainType int

const (
	notAnd chainType = -1
	and    chainType = 1
	or     chainType = 2
	notOr  chainType = -2
)

type columnSelection struct {
	column   string
	value    any
	selected bool
	chainTy  chainType
}

type assoc struct {
	name    string
	replace any
}

func (g *DefaultGormCore) update(
	id string,
	selected, grouped []columnSelection,
	task gormmodel.Task,
) (updated bool, err error) {
	hasId := g.db.Select("id").Where("id = ?", id).Find(&gormmodel.Task{})
	if hasId.Error != nil {
		return false, hasId.Error
	}
	if hasId.RowsAffected != 1 {
		return false, &scheduler.RepositoryError{Kind: scheduler.IdNotFound}
	}

	tx := g.db.Model(&gormmodel.Task{Id: id})

	selects := make([]string, 0, len(selected)+len(grouped))
	for i := 0; i < len(selected); i++ {
		if selected[i].selected {
			selects = append(selects, selected[i].column)
		}
	}
	for i := 0; i < len(grouped); i++ {
		if grouped[i].selected {
			selects = append(selects, grouped[i].column)
		}
	}

	tx = tx.Select(selects)
	tx = composeWhere(g.db, tx, selected)

	if grouped != nil {
		innerTx := g.db.Where("")
		innerTx = composeWhere(g.db, innerTx, grouped)
		tx = tx.Where(innerTx)
	}

	result := tx.Updates(task)

	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected > 0 {
		updated = true
	}
	return updated, nil
}

func composeWhere(db, tx *gorm.DB, wheres []columnSelection) *gorm.DB {
	for _, where := range wheres {
		query := where.column
		if where.value == nil {
			query += " IS ?"
		} else {
			query += " = ?"
		}
		switch where.chainTy {
		case and:
			tx = tx.Where(query, where.value)
		case notAnd:
			tx = tx.Not(query, where.value)
		case or:
			tx = tx.Or(query, where.value)
		case notOr:
			tx = tx.Or(db.Not(query, where.value))
		}
	}
	return tx
}

func (g *DefaultGormCore) Update(id string, param scheduler.TaskParam) (updated bool, err error) {
	var selected []columnSelection
	if param.HasOnlyMeta() {
		selected = []columnSelection{}
	} else {
		selected = []columnSelection{
			{"dispatched_at", nil, false, and},
			{"cancelled_at", nil, false, and},
			{"done_at", nil, false, and},
		}
	}

	task := gormmodel.FromTask(param.ToTask(false))

	grouped := make([]columnSelection, 0, 5)
	if !param.ScheduledAt.IsZero() {
		grouped = append(grouped, columnSelection{column: "scheduled_at", value: task.ScheduledAt, selected: true, chainTy: notOr})
	}
	if param.WorkId != "" {
		grouped = append(grouped, columnSelection{column: "work_id", value: task.WorkId, selected: true, chainTy: notOr})
	}
	if param.Param != nil {
		grouped = append(grouped, columnSelection{column: "param", value: task.Param, selected: true, chainTy: notOr})
	}
	if param.Priority != nil {
		grouped = append(grouped, columnSelection{column: "priority", value: param.Priority, selected: true, chainTy: notOr})
	}
	if param.Meta != nil {
		grouped = append(grouped, columnSelection{column: "meta", value: task.Meta, selected: true, chainTy: notOr})
	}

	updated, err = g.update(id, selected, grouped, task)
	if err != nil {
		return false, err
	}

	if !updated {
		task, err := g.GetById(id)
		if err != nil {
			return false, err
		}
		if err := ErrKindUpdate(task); err != nil {
			return false, err
		}
	}
	return updated, err
}

func (g *DefaultGormCore) Find(t scheduler.TaskMatcher) ([]scheduler.Task, error) {
	matcher := gormmodel.FromTaskMatcher(t)

	selections := make([]columnSelection, 0, 11)

	addWhere := func(columnName string, value any) {
		selections = append(selections, columnSelection{columnName, value, false, and})
	}

	matcher.VisitNonZero(gormmodel.TaskVisitor{
		Id:           func(v string) { addWhere("`id`", v) },
		WorkId:       func(v string) { addWhere("`work_id`", v) },
		Param:        func(v string) { addWhere("`param`", v) },
		ScheduledAt:  func(v time.Time) { addWhere("`scheduled_at`", v) },
		CreatedAt:    func(v time.Time) { addWhere("`created_at`", v) },
		CancelledAt:  func(v *time.Time) { addWhere("`cancelled_at`", v) },
		DispatchedAt: func(v *time.Time) { addWhere("`dispatched_at`", v) },
		DoneAt:       func(v *time.Time) { addWhere("`done_at`", v) },
		Err:          func(v string) { addWhere("`err`", v) },
		Meta:         func(v datatypes.JSON) { addWhere("`meta`", v) },
		Priority:     func(v *int) { addWhere("`priority`", v) },
	})

	out := make([]gormmodel.Task, 0)

	tx := g.db.Model(&gormmodel.Task{})
	tx = composeWhere(g.db, tx, selections)

	result := tx.Find(&out)
	if result.Error != nil {
		return nil, result.Error
	}

	matched := make([]scheduler.Task, len(out))
	for i := 0; i < len(out); i++ {
		matched[i] = out[i].ToTask()
	}

	return matched, nil
}

func (g *DefaultGormCore) FindMetaContain(matcher []scheduler.KeyValuePairMatcher) ([]scheduler.Task, error) {
	if len(matcher) == 0 {
		return []scheduler.Task{}, nil
	}

	tx := g.db.Model(&gormmodel.Task{})

	for _, kv := range matcher {
		switch kv.MatchTy {
		case scheduler.HasKey:
			tx = tx.Where(datatypes.JSONQuery("meta").HasKey(kv.Key))
		case scheduler.Exact:
			tx = tx.Where(datatypes.JSONQuery("meta").Equals(kv.Value, kv.Key))
		case scheduler.Forward, scheduler.Backward, scheduler.Partial:
			// TODO: handle correctly after this commit lands
			// https://github.com/go-gorm/datatypes/commit/b3e966cc69f8d5c3e1aa45c61bd00226bd3ad0f5
			return nil, fmt.Errorf(
				"%w: not supported for %s",
				scheduler.ErrNotSupported, kv.MatchTy,
			)
		default:
			continue
		}
	}

	matched := make([]gormmodel.Task, 0)
	err := tx.Find(&matched).Error
	if err != nil {
		return nil, err
	}

	out := make([]scheduler.Task, len(matched))
	for i := 0; i < len(matched); i++ {
		out[i] = matched[i].ToTask()
	}
	return out, nil
}

func (g *DefaultGormCore) Cancel(id string) (cancelled bool, err error) {
	cancelled, err = g.update(
		id,
		[]columnSelection{
			{"dispatched_at", nil, false, and},
			{column: "cancelled_at", value: nil, selected: true, chainTy: and},
			{"done_at", nil, false, and},
		},
		nil,
		gormmodel.Task{CancelledAt: util.Escape(g.nowGetter.GetNow())},
	)
	if err != nil {
		return false, err
	}
	if !cancelled {
		task, err := g.GetById(id)
		if err != nil {
			return false, err
		}
		if err := ErrKindCancel(task); err != nil {
			return false, err
		}
	}
	return cancelled, err
}
func (g *DefaultGormCore) MarkAsDispatched(id string) error {
	updated, err := g.update(
		id,
		[]columnSelection{
			{"dispatched_at", nil, true, and},
			{"cancelled_at", nil, false, and},
			{"done_at", nil, false, and},
		},
		nil,
		gormmodel.Task{DispatchedAt: util.Escape(g.nowGetter.GetNow())},
	)

	if err != nil {
		return err
	}
	if !updated {
		task, err := g.GetById(id)
		if err != nil {
			return err
		}
		if err := ErrKindMarkAsDispatch(task); err != nil {
			return err
		}
	}

	return err
}
func (g *DefaultGormCore) MarkAsDone(id string, err error) error {
	var errStr string
	if err != nil {
		errStr = err.Error()
	}

	updated, updateErr := g.update(
		id,
		[]columnSelection{
			{"dispatched_at", nil, false, notAnd},
			{"cancelled_at", nil, false, and},
			{"done_at", nil, true, and},
			{"err", "", true, and},
		},
		nil,
		gormmodel.Task{DoneAt: util.Escape(g.nowGetter.GetNow()), Err: errStr},
	)

	if updateErr != nil {
		return updateErr
	}
	if !updated {
		task, err := g.GetById(id)
		if err != nil {
			return err
		}
		if err := ErrKindMarkAsDone(task); err != nil {
			return err
		}
	}
	return updateErr
}

func (g *DefaultGormCore) GetNext() (scheduler.Task, error) {
	var t gormmodel.Task
	result := g.db.
		Where("cancelled_at IS NULL").
		Where("dispatched_at IS NULL").
		Where("done_at IS NULL").
		Order("scheduled_at asc, priority desc").
		Limit(1).
		Find(&t)

	if result.Error != nil {
		return scheduler.Task{}, result.Error
	}
	if result.RowsAffected > 0 {
		return t.ToTask(), nil
	}
	return scheduler.Task{}, &scheduler.RepositoryError{Kind: scheduler.Empty}
}

func (g *DefaultGormCore) GetNextMany() ([]gormmodel.Task, error) {
	t := make([]gormmodel.Task, 0, 100)
	result := g.db.
		Where("cancelled_at IS NULL").
		Where("dispatched_at IS NULL").
		Where("done_at IS NULL").
		Order("scheduled_at asc, priority desc").
		Limit(10).
		Find(&t)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected > 0 {
		return t, nil
	}
	return nil, &scheduler.RepositoryError{Kind: scheduler.Empty}
}

var _ scheduler.DispatchedReverter = (*DefaultGormCore)(nil)

func (g *DefaultGormCore) RevertDispatched() error {
	err := g.db.
		Where("dispatched_at IS NOT NULL AND cancelled_at IS NULL AND done_at IS NULL").
		Update("dispatched_at", nil).
		Error
	if err != nil {
		return err
	}
	return nil
}

const (
	cancelledDeleteQuery = "cancelled_at IS NOT NULL AND cancelled_at < ?"
	doneDeletedQuery     = "done_at IS NOT NULL AND done_at < ?"
)

var _ scheduler.BeforeDeleter = (*DefaultGormCore)(nil)

func (g *DefaultGormCore) DeleteBefore(before time.Time, returning bool) (scheduler.Deleted, error) {
	var (
		deleted         scheduler.Deleted
		cancelled, done []gormmodel.Task
	)

	if returning {
		cancelled = make([]gormmodel.Task, 0)
		done = make([]gormmodel.Task, 0)

		if err := g.db.Where(cancelledDeleteQuery, before).Find(&cancelled).Error; err != nil {
			return scheduler.Deleted{}, err
		}
		if err := g.db.Where(doneDeletedQuery, before).Find(&done).Error; err != nil {
			return scheduler.Deleted{}, err
		}
	}

	tx := g.db
	if !g.softDelete {
		tx = tx.Unscoped()
	}

	result := tx.
		Where(cancelledDeleteQuery, before).
		Or(doneDeletedQuery, before).
		Delete(&gormmodel.Task{})

	if returning && result.Error == nil {
		deleted.Cancelled = make(map[string]scheduler.Task, len(cancelled))
		deleted.Done = make(map[string]scheduler.Task, len(done))

		for _, task := range cancelled {
			deleted.Cancelled[task.Id] = task.ToTask()
		}
		for _, task := range done {
			deleted.Done[task.Id] = task.ToTask()
		}
	}

	return deleted, result.Error
}

// HardDelete deletes soft-deleted tasks.
func (g *DefaultGormCore) HardDelete(returning bool) ([]gormmodel.Task, error) {
	deleted := make([]gormmodel.Task, 0)

	tx := g.db
	if returning {
		tx = tx.Clauses(clause.Returning{})
	}

	err := tx.
		Unscoped().
		Where("deleted_at IS NOT NULL").
		Delete(&deleted).
		Error
	if err != nil {
		return nil, err
	}
	return deleted, nil
}
