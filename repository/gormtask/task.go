package gormtask

import (
	"time"

	"github.com/google/uuid"
	"github.com/ngicks/gokugen/scheduler"
	"gorm.io/gorm"
)

type GormTask struct {
	Id            string         `json:"id" gorm:"primaryKey,not null"` // Id is an id of the task.
	WorkId        string         `json:"work_id" gorm:"not null"`       // WorkId is work function id.
	Param         string         `json:"param"`
	ScheduledAt   time.Time      `json:"scheduled_at" gorm:"not null,index:sched,sort:asc"`
	Priority      int            `json:"priority" gorm:"not null,index:sched,sort:desc"`
	CreatedAt     time.Time      `json:"created_at" gorm:"not null,autoCreateTime:milli"`
	CancelledAt   *time.Time     `json:"cancelled_at,omitempty"`
	DispatchedAt  *time.Time     `json:"dispatched_at,omitempty"`
	DoneAt        *time.Time     `json:"done_at,omitempty"`
	Err           string         `json:"err"`
	UpdatedAt     int64          `gorm:"autoUpdateTime:milli"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index:deleted"`
	UpdatedFields []string       `json:"-" gorm:"-:all"`
}

func FromTask(t scheduler.Task) GormTask {
	return GormTask{
		Id:           t.Id,
		WorkId:       t.WorkId,
		Param:        string(t.Param),
		ScheduledAt:  t.ScheduledAt,
		Priority:     t.Priority,
		CreatedAt:    t.CreatedAt,
		CancelledAt:  t.CancelledAt,
		DispatchedAt: t.DispatchedAt,
		DoneAt:       t.DoneAt,
		Err:          t.Err,
	}
}

func (t GormTask) GenerateId() GormTask {
	t.Id = uuid.NewString()
	return t
}

func (t GormTask) ToTask() scheduler.Task {
	return scheduler.Task{
		Id:           t.Id,
		WorkId:       t.WorkId,
		Param:        []byte(t.Param),
		ScheduledAt:  t.ScheduledAt,
		Priority:     t.Priority,
		CreatedAt:    t.CreatedAt,
		CancelledAt:  t.CancelledAt,
		DispatchedAt: t.DispatchedAt,
		DoneAt:       t.DoneAt,
		Err:          t.Err,
	}
}
