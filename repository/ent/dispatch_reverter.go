package ent

import (
	"context"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/gokugen/repository/ent/gen"
	"github.com/ngicks/gokugen/repository/ent/gen/task"
)

var _ def.DispatchedReverter = (*EntRepository)(nil)

func (r *EntRepository) RevertDispatched(ctx context.Context) error {
	err := r.client.Task.Update().
		Where(task.StateEQ(task.StateDispatched)).
		SetState(task.StateScheduled).
		Exec(ctx)
	if gen.IsNotFound(err) {
		return &def.RepositoryError{Kind: def.Exhausted, Raw: err}
	}
	return err
}

func (r *EntRepository) CancelDispatched(ctx context.Context) error {
	err := r.client.Task.Update().
		Where(task.StateEQ(task.StateDispatched)).
		SetState(task.StateCancelled).
		SetCancelledAt(def.NormalizeTime(r.clock.Now())).
		Exec(ctx)
	if gen.IsNotFound(err) {
		return &def.RepositoryError{Kind: def.Exhausted, Raw: err}
	}
	return err
}
