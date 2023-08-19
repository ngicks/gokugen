package repository

import (
	"context"

	"github.com/ngicks/gokugen/def"
)

// HookTimer watches repository mutation and resets its timer if necessary.
type HookTimer interface {
	// SetRepository sets Repository. Calling this twice may cause a runtime panic.
	SetRepository(core def.Repository)
	AddTask(ctx context.Context, param def.TaskUpdateParam)
	UpdateById(ctx context.Context, id string, param def.TaskUpdateParam)
	Cancel(ctx context.Context, id string)
	MarkAsDispatched(ctx context.Context, id string)
	def.Observer
}
