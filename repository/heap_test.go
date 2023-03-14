package repository

import (
	"encoding/json"
	"errors"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ngicks/gokugen/scheduler"
	acceptancetest "github.com/ngicks/gokugen/scheduler/acceptance_test"
	"github.com/ngicks/type-param-common/util"
	"github.com/stretchr/testify/require"
)

var _ scheduler.TaskRepository = (*HeapRepository)(nil)

func TestHeapAcceptance(t *testing.T) {
	acceptancetest.TestRepository(
		t,
		func() scheduler.TaskRepository { return NewHeapRepository() },
		acceptancetest.RepositoryTestConfig{
			FindMetaContain: acceptancetest.FindMetaContainTestConfig{
				Forward:  true,
				Backward: true,
				Partial:  true,
			},
			RevertDispatched: true,
			DeleteBefore:     true,
		},
	)
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
	// count must be larger than 10 or zero.
	for _, count := range []int{0, 10, 100, 1000} {
		testHeapCloneN(t, count)
	}
}

func testHeapCloneN(t *testing.T, taskCount int) {
	require := require.New(t)

	heap := NewHeapRepository()

	err := populateHeap(heap, taskCount)
	require.NoError(err)

	dumped := heap.Dump()

	// making sure all fields are populated...
	if taskCount > 0 {
		rv := reflect.ValueOf(dumped)
		for i := 0; i < rv.NumField(); i++ {
			require.Greater(rv.Field(i).Len(), 0)
		}
	}

	marshalled, err := json.Marshal(dumped)
	require.NoError(err)

	var recovered TaskMap
	err = json.Unmarshal(marshalled, &recovered)
	require.NoError(err)

	recoveredHeap := NewHeapRepositoryFromMap(recovered)

	// wrappedTask.Index can generate diff.
	// It is an index of slice used in min-heap. Index can be different one.
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

func populateHeap(heap *HeapRepository, len int) error {
	tasks, err := addRandomTask(heap, len)
	if err != nil {
		return err
	}

	for i := 0; i < len; i++ {
		if i%6 == 0 {
			_, _ = heap.Delete(tasks[i].Id)
		} else if i%5 == 0 {
			_ = heap.MarkAsDispatched(tasks[i].Id)
			_ = heap.MarkAsDone(tasks[i].Id, errors.New("foobar"))
		} else if i%4 == 0 {
			_ = heap.MarkAsDispatched(tasks[i].Id)
			_ = heap.MarkAsDone(tasks[i].Id, nil)
		} else if i%3 == 0 {
			_, _ = heap.Cancel(tasks[i].Id)
		} else if i%2 == 0 {
			_ = heap.MarkAsDispatched(tasks[i].Id)
		}
	}

	return nil
}
