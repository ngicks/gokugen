package ent

import (
	"context"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/gokugen/repository/ent/gen"
	"github.com/ngicks/gokugen/repository/ent/gen/task"
)

var _ def.EndedDeleter = (*EntRepository)(nil)

func (r *EntRepository) DeleteEnded(
	ctx context.Context,
	returning bool,
	limit int,
) error {
	_, err := r.client.Task.Delete().
		Where(task.StateIn(task.StateCancelled, task.StateDone, task.StateErr)).
		Exec(ctx)
	if gen.IsNotFound(err) {
		return &def.RepositoryError{Kind: def.Exhausted, Raw: err}
	}
	return err
}
