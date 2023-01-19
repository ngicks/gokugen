package repository

import (
	"testing"

	"github.com/ngicks/gokugen/scheduler"
	acceptancetest "github.com/ngicks/gokugen/scheduler/acceptance_test"
)

var _ scheduler.TaskRepository = &HeapRepository{}

func TestHeapAcceptance(t *testing.T) {
	heap := NewHeapRepository()
	acceptancetest.TestRepository(t, heap)
}
