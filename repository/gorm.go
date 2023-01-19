package repository

import (
	"fmt"
	"time"

	"github.com/ngicks/gokugen/repository/gormtask"
	"github.com/ngicks/gokugen/scheduler"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// GormRepository is a small wrapper around GormCore and HookTimer
// that implements scheduler.TaskRepository.
// It connects HookTimer to Core so that timer can be updated correctly,
// and it expands Core's simpler interface to a bit complex one to fit
// into scheduler.TaskRepository.
//
// GormCore is quite same as scheduler.TaskRepository at the moment.
// Over time, GormRepository will strips down Core to make it simple and small.
type GormRepository struct {
	Core      GormCore
	HookTimer HookTimer
}

func NewGorm(core GormCore, hookTimer HookTimer) *GormRepository {
	return &GormRepository{
		Core:      core,
		HookTimer: hookTimer,
	}
}

func NewSqlite3(dbPath string, opts ...gorm.Option) (*GormRepository, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), opts...)
	if err != nil {
		return nil, err
	}

	// Migrate the schema
	err = db.AutoMigrate(&gormtask.GormTask{})
	if err != nil {
		return nil, err
	}

	core := NewDefaultGormCore(db)
	hookTimer, err := NewHookTimer(core)
	if err != nil {
		return nil, err
	}

	return &GormRepository{
		Core:      core,
		HookTimer: hookTimer,
	}, nil
}

func (r *GormRepository) AddTask(param scheduler.TaskParam) (scheduler.Task, error) {
	if !param.IsInitialized() {
		return scheduler.Task{}, fmt.Errorf("input param is not initialized")
	}

	t, err := r.Core.AddTask(param)
	if err != nil {
		return scheduler.Task{}, err
	}

	r.HookTimer.AddTask(param)

	return t.ToTask(), nil
}

func (r *GormRepository) Cancel(id string) (cancelled bool, err error) {
	cancelled, err = r.Core.Cancel(id)
	if err == nil {
		r.HookTimer.Cancel(id)
	}
	return cancelled, err
}
func (r *GormRepository) GetById(id string) (scheduler.Task, error) {
	t, err := r.Core.GetById(id)
	if err != nil {
		return scheduler.Task{}, err
	}
	return t.ToTask(), nil
}
func (r *GormRepository) GetNext() (scheduler.Task, error) {
	t, err := r.Core.GetNext()
	if err != nil {
		return scheduler.Task{}, err
	}
	return t.ToTask(), nil
}
func (r *GormRepository) MarkAsDispatched(id string) error {
	err := r.Core.MarkAsDispatched(id)
	if err == nil {
		r.HookTimer.MarkAsDispatched(id)
	}
	return err
}
func (r *GormRepository) MarkAsDone(id string, err error) error {
	return r.Core.MarkAsDone(id, err)
}
func (r *GormRepository) Update(id string, param scheduler.TaskParam) (updated bool, err error) {
	updated, err = r.Core.Update(id, param)
	if err == nil {
		r.HookTimer.Update(id, param)
	}
	return updated, err
}

func (r *GormRepository) StartTimer() {
	r.HookTimer.StartTimer()
}

func (r *GormRepository) StopTimer() {
	r.HookTimer.StopTimer()
}

func (r *GormRepository) TimerChannel() <-chan time.Time {
	return r.HookTimer.TimerChannel()
}
