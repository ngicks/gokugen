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

func TestHeapAcceptance_milli_sec_precise(t *testing.T) {
	heap := NewHeapRepository()
	heap.isMilliSecPrecise = true
	acceptancetest.TestRepository(t, heap)
}
