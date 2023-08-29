package cron

import (
	sortabletask "github.com/ngicks/gokugen/internal/sortable_task"
	"github.com/ngicks/gokugen/mutator"
)

type wrappedTask struct {
	*sortabletask.IndexedTask
	key      serializable
	mutators mutator.Mutators
}
