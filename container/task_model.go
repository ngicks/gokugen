package container

import (
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/ngicks/gokugen/scheduler"
	tpc "github.com/ngicks/type-param-common"
)

var _ tpc.Lessable[*TaskModel] = &TaskModel{}
var _ scheduler.Task = &TaskModel{}

type TaskModel struct {
	id          string
	taskId      string
	scheduledAt time.Time
	param       []byte
	isDone      int32
	isCancelled int32
}

func NewTask(taskId string, param []byte, scheduledAt time.Time) *TaskModel {
	return &TaskModel{
		id:          uuid.NewString(),
		taskId:      taskId,
		scheduledAt: scheduledAt,
		param:       param,
	}
}

func (m TaskModel) Id() string {
	return m.id
}
func (m TaskModel) TaskId() string {
	return m.taskId
}
func (m TaskModel) ScheduledAt() time.Time {
	return m.scheduledAt
}
func (m TaskModel) Param() []byte {
	return m.param
}

func (m *TaskModel) IsCancelled() bool {
	return atomic.LoadInt32(&m.isCancelled) == 1
}

func (m *TaskModel) IsDone() bool {
	return atomic.LoadInt32(&m.isDone) == 1
}

// Inner implements Lessable[Model, Model]
func (m *TaskModel) Inner() *TaskModel {
	return m
}

// LessThan implements Lessable[Model, Model]
func (m *TaskModel) LessThan(l tpc.Lessable[*TaskModel]) bool {
	return m.ScheduledAt().Before(l.Inner().ScheduledAt())
}

func (m *TaskModel) setCancelled() (cancelled bool) {
	return atomic.CompareAndSwapInt32(&m.isCancelled, 0, 1)
}
