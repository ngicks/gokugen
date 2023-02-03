package repository

import (
	"github.com/google/uuid"
	"github.com/ngicks/gokugen/repository/gormtask"
	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/gommon/pkg/common"
	"github.com/ngicks/type-param-common/util"
	"gorm.io/gorm"
)

var _ scheduler.RepositoryLike = &DefaultGormCore{}

type DefaultGormCore struct {
	db        *gorm.DB
	timer     common.Timer
	nowGetter common.NowGetter
}

func NewDefaultGormCore(db *gorm.DB) *DefaultGormCore {
	return &DefaultGormCore{
		db:        db,
		timer:     common.NewTimerReal(),
		nowGetter: common.NowGetterReal{},
	}
}

func (g *DefaultGormCore) AddTask(param scheduler.TaskParam) (scheduler.Task, error) {
	t := gormtask.FromTask(param.ToTask(false))
	t.Id = uuid.NewString()

	result := g.db.Create(&t)
	if result.Error != nil {
		return scheduler.Task{}, result.Error
	}
	return t.ToTask(), nil
}

func (g *DefaultGormCore) GetById(id string) (scheduler.Task, error) {
	t := gormtask.GormTask{}
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

func (g *DefaultGormCore) update(
	id string,
	selected, grouped []columnSelection,
	task gormtask.GormTask,
) (updated bool, err error) {
	hasId := g.db.Select("id").Where("id = ?", id).Find(&gormtask.GormTask{})
	if hasId.Error != nil {
		return false, hasId.Error
	}
	if hasId.RowsAffected != 1 {
		return false, &scheduler.RepositoryError{Kind: scheduler.IdNotFound}
	}

	var t gormtask.GormTask
	tx := g.db.
		Model(&t).
		Where("id = ?", id)

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
	return result.RowsAffected > 0, nil
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

	task := gormtask.FromTask(param.ToTask(false))

	grouped := make([]columnSelection, 0, 5)
	if !param.ScheduledAt.IsZero() {
		grouped = append(grouped, columnSelection{"scheduled_at", task.ScheduledAt, true, notOr})
	}
	if param.WorkId != "" {
		grouped = append(grouped, columnSelection{"work_id", task.WorkId, true, notOr})
	}
	if param.Param != nil {
		grouped = append(grouped, columnSelection{"param", task.Param, true, notOr})
	}
	if param.Priority != nil {
		grouped = append(grouped, columnSelection{"priority", task.Priority, true, notOr})
	}
	if param.Meta != nil {
		grouped = append(grouped, columnSelection{"meta", task.Meta, true, notOr})
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

func (g *DefaultGormCore) Cancel(id string) (cancelled bool, err error) {
	cancelled, err = g.update(
		id,
		[]columnSelection{
			{"dispatched_at", nil, false, and},
			{column: "cancelled_at", value: nil, selected: true, chainTy: and},
			{"done_at", nil, false, and},
		},
		nil,
		gormtask.GormTask{CancelledAt: util.Escape(g.nowGetter.GetNow())},
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
		gormtask.GormTask{DispatchedAt: util.Escape(g.nowGetter.GetNow())},
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
		gormtask.GormTask{DoneAt: util.Escape(g.nowGetter.GetNow()), Err: errStr},
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
	var t gormtask.GormTask
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

func (g *DefaultGormCore) GetNextMany() ([]gormtask.GormTask, error) {
	t := make([]gormtask.GormTask, 0, 100)
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
