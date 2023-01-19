package repository

import (
	"github.com/google/uuid"
	"github.com/ngicks/gokugen/repository/gormtask"
	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/gommon/pkg/common"
	"github.com/ngicks/type-param-common/util"
	"gorm.io/gorm"
)

type GormCore interface {
	AddTask(param scheduler.TaskParam) (gormtask.GormTask, error)
	Cancel(id string) (cancelled bool, err error)
	GetById(id string) (gormtask.GormTask, error)
	GetNext() (gormtask.GormTask, error)
	MarkAsDispatched(id string) error
	MarkAsDone(id string, err error) error
	Update(id string, param scheduler.TaskParam) (updated bool, err error)
}

var _ GormCore = &DefaultGormCore{}

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

func (g *DefaultGormCore) AddTask(param scheduler.TaskParam) (gormtask.GormTask, error) {
	orgTask := param.ToTask(false)
	t := gormtask.FromTask(orgTask)
	t.Id = uuid.NewString()

	result := g.db.Create(&t)
	if result.Error != nil {
		return gormtask.GormTask{}, result.Error
	}
	return t, nil
}

func (g *DefaultGormCore) GetById(id string) (gormtask.GormTask, error) {
	t := gormtask.GormTask{}
	result := g.db.Where("id = ?", id).Limit(1).Find(&t)
	if result.Error != nil {
		return gormtask.GormTask{}, result.Error
	}
	if result.RowsAffected > 0 {
		return t, nil
	}
	return gormtask.GormTask{}, &scheduler.RepositoryError{Kind: scheduler.IdNotFound}
}

type chainType int

const (
	not   chainType = -1
	where chainType = 1
	or    chainType = 2
	notOr chainType = -2
)

type pair struct {
	l        string
	r        any
	selected bool
	where    chainType
}

func (g *DefaultGormCore) update(id string, selected []pair, grouped []pair, task gormtask.GormTask) (updated bool, err error) {
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
			selects = append(selects, selected[i].l)
		}
	}
	for i := 0; i < len(grouped); i++ {
		if grouped[i].selected {
			selects = append(selects, grouped[i].l)
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

func composeWhere(db, tx *gorm.DB, pairs []pair) *gorm.DB {
	for _, pair := range pairs {
		query := pair.l
		if pair.r == nil {
			query += " IS ?"
		} else {
			query += " = ?"
		}
		switch pair.where {
		case where:
			tx = tx.Where(query, pair.r)
		case not:
			tx = tx.Not(query, pair.r)
		case or:
			tx = tx.Or(query, pair.r)
		case notOr:
			tx = tx.Or(db.Not(query, pair.r))
		}
	}
	return tx
}

func (g *DefaultGormCore) Update(id string, param scheduler.TaskParam) (updated bool, err error) {
	selected := []pair{
		{"dispatched_at", nil, false, where},
		{"cancelled_at", nil, false, where},
		{"done_at", nil, false, where},
	}

	grouped := make([]pair, 0, 4)
	if !param.ScheduledAt.IsZero() {
		grouped = append(grouped, pair{"scheduled_at", param.ScheduledAt, true, notOr})
	}
	if param.WorkId != "" {
		grouped = append(grouped, pair{"work_id", param.WorkId, true, notOr})
	}
	if param.Param != nil {
		grouped = append(grouped, pair{"param", string(param.Param), true, notOr})
	}
	if param.Priority != nil {
		grouped = append(grouped, pair{"priority", *param.Priority, true, notOr})
	}

	updated, err = g.update(id, selected, grouped, gormtask.FromTask(param.ToTask(false)))
	if err != nil {
		return false, err
	}

	if !updated {
		task, err := g.GetById(id)
		if err != nil {
			return false, err
		}
		errKind := errKind(task.ToTask(), errKindOption{})
		if errKind != "" {
			return false, &scheduler.RepositoryError{Id: id, Kind: errKind}
		}
	}
	return updated, err
}

func (g *DefaultGormCore) Cancel(id string) (cancelled bool, err error) {
	cancelled, err = g.update(
		id,
		[]pair{
			{"dispatched_at", nil, false, where},
			{l: "cancelled_at", r: nil, selected: true, where: where},
			{"done_at", nil, false, where},
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
		errKind := errKind(task.ToTask(), errKindOption{skipCancelledAt: true})
		if errKind != "" {
			return false, &scheduler.RepositoryError{Id: id, Kind: errKind}
		}
	}
	return cancelled, err
}
func (g *DefaultGormCore) MarkAsDispatched(id string) error {
	updated, err := g.update(
		id,
		[]pair{
			{"dispatched_at", nil, true, where},
			{"cancelled_at", nil, false, where},
			{"done_at", nil, false, where},
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
		errKind := errKind(task.ToTask(), errKindOption{})
		if errKind != "" {
			return &scheduler.RepositoryError{Id: id, Kind: errKind}
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
		[]pair{
			{"dispatched_at", nil, false, not},
			{"cancelled_at", nil, false, where},
			{"done_at", nil, true, where},
			{"err", "", true, where},
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
		errKind := errKind(
			task.ToTask(),
			errKindOption{returnOnEmptyDispatchedAt: true, skipDispatchedAt: true},
		)
		if errKind != "" {
			return &scheduler.RepositoryError{Id: id, Kind: errKind}
		}
	}
	return updateErr
}

func (g *DefaultGormCore) GetNext() (gormtask.GormTask, error) {
	var t gormtask.GormTask
	result := g.db.
		Where("cancelled_at IS NULL").
		Where("dispatched_at IS NULL").
		Where("done_at IS NULL").
		Order("scheduled_at asc, priority desc").
		Limit(1).
		Find(&t)

	if result.Error != nil {
		return gormtask.GormTask{}, result.Error
	}
	if result.RowsAffected > 0 {
		return t, nil
	}
	return gormtask.GormTask{}, &scheduler.RepositoryError{Kind: scheduler.Empty}
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
