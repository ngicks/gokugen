package repository

import (
	"encoding/json"
	"errors"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ngicks/gokugen/scheduler"
	acceptancetest "github.com/ngicks/gokugen/scheduler/acceptance_test"
	"github.com/ngicks/type-param-common/util"
	"github.com/stretchr/testify/require"
)

var _ scheduler.TaskRepository = &HeapRepository{}

func TestHeapAcceptance(t *testing.T) {
	heap := NewHeapRepository()
	acceptancetest.TestRepository(t, heap)
}

func addRandomTask(repo scheduler.TaskRepository, n int) (added []scheduler.Task, err error) {
	now := time.Now()

	for i := 0; i < n; i++ {
		t, err := repo.AddTask(scheduler.TaskParam{
			ScheduledAt: now.Add(time.Duration(rand.Int63())),
			WorkId:      strconv.FormatInt(int64(i), 10),
			Priority:    util.Escape(rand.Int()),
		})
		if err != nil {
			return nil, err
		}
		added = append(added, t)
	}

	return added, nil
}

func TestHeapClone(t *testing.T) {
	for _, count := range []int{0, 10, 100} {
		testHeapCloneN(t, count)
	}
}

func testHeapCloneN(t *testing.T, taskCount int) {
	require := require.New(t)

	heap := NewHeapRepository()

	tasks, err := addRandomTask(heap, taskCount)
	require.NoError(err)

	if taskCount != 0 {
		heap.MarkAsDispatched(tasks[taskCount/2].Id)
		heap.Cancel(tasks[taskCount/3].Id)
		heap.MarkAsDispatched(tasks[taskCount/4].Id)
		heap.MarkAsDone(tasks[taskCount/4].Id, nil)
		heap.MarkAsDispatched(tasks[taskCount/5].Id)
		heap.MarkAsDone(tasks[taskCount/5].Id, errors.New("foobar"))
	}

	dumped := heap.Dump()

	marshalled, err := json.Marshal(dumped)
	require.NoError(err)

	var recovered TaskMap
	err = json.Unmarshal(marshalled, &recovered)
	require.NoError(err)

	recoveredHeap := NewHeapRepositoryFromMap(recovered)

	// wrappedTask.Index can generate diff. It is an index of slice used in min-heap. Index can be different one.
	if diff := cmp.Diff(heap.taskMap.Dump(), recoveredHeap.taskMap.Dump()); diff != "" {
		t.Fatalf("not equal. diff = %s", diff)
	}

	for {
		if heap.heap.Len() == 0 || recoveredHeap.heap.Len() == 0 {
			if heap.heap.Len() != 0 || recoveredHeap.heap.Len() != 0 {
				t.Fatalf("wrong len")
			}
			break
		}

		popOrg, popRcv := heap.heap.Pop(), recoveredHeap.heap.Pop()

		if diff := cmp.Diff(*popOrg, *popRcv); diff != "" {
			t.Fatalf("not qual. diff = %s", diff)
		}
	}
}
