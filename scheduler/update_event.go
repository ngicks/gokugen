package scheduler

type UpdateType string

const (
	CancelTask       UpdateType = "cancel_task"
	UpdateParam      UpdateType = "update_param"
	MarkAsDispatched UpdateType = "mark_as_dispatched"
	MarkAsDone       UpdateType = "mark_as_done"
)

type updateEvent struct {
	id         string
	updateType UpdateType
	err        error
	task       Task
	param      TaskParam
	responseCh chan error
}
