package heap_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ngicks/gokugen/heap"
)

func TestMinHeap(t *testing.T) {
	// Seeing basic delegation.
	// If these codes work, it should work for any type.
	t.Run("number heap", func(t *testing.T) {
		h := heap.NewNumber[int]()
		ans := []int{3, 4, 4, 5, 6}
		h.Push(5)
		h.Push(4)
		h.Push(6)
		h.Push(3)
		h.Push(4)

		for _, i := range ans {
			popped := h.Pop()
			if popped != i {
				t.Errorf("pop = %v expected %v", popped, i)
			}
		}
		if h.Len() != 0 {
			t.Errorf("expect empty but size = %v", h.Len())
		}
	})

	t.Run("struct heap", func(t *testing.T) {
		type testStruct struct {
			t time.Time
		}
		less := func(i, j *testStruct) bool {
			return i.t.Before(j.t)
		}

		h := heap.NewHeap(less)
		ans := []*testStruct{
			{t: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
			{t: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC)},
			{t: time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)},
			{t: time.Date(2021, 4, 1, 0, 0, 0, 0, time.UTC)},
			{t: time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC)},
		}
		h.Push(ans[2])
		h.Push(ans[1])
		h.Push(ans[3])
		h.Push(ans[0])
		h.Push(ans[4])

		for _, i := range ans {
			popped := h.Pop()
			if popped.t != i.t {
				t.Errorf("pop = %v expected %v", popped.t, i.t)
			}
		}
		if h.Len() != 0 {
			t.Errorf("expect empty but size = %v", h.Len())
		}
	})

	t.Run("Exclude", func(t *testing.T) {
		h := heap.NewNumber[int]()
		h.Push(7)
		h.Push(4)
		h.Push(1)
		h.Push(6)
		h.Push(5)
		h.Push(3)
		h.Push(2)

		removed := h.Exclude(func(ent int) bool { return ent%2 == 0 }, -1, 100)

		fmt.Println(removed)

		for i := 1; i <= 7; i = i + 2 {
			popped := h.Pop()
			if popped != i {
				t.Errorf("pop = %v expected %v", popped, i)
			}
		}
		if h.Len() != 0 {
			t.Errorf("expect empty but size = %v", h.Len())
		}

		h.Push(1)
		h.Push(3)
		h.Push(5)
		h.Push(7)
		h.Push(9)
		h.Push(11)
		h.Push(13)

		removed = h.Exclude(func(ent int) bool { return ent%2 != 0 }, 0, 3)

		if len(removed) != 3 {
			t.Fatalf("removed len must be %d, but is %d: %v", 3, len(removed), removed)
		}
		for h.Len() != 0 {
			h.Pop()
		}

		h.Push(1)
		h.Push(3)
		h.Push(5)
		h.Push(7)
		h.Push(9)
		h.Push(11)
		h.Push(13)

		removed = h.Exclude(func(ent int) bool { return ent%2 != 0 }, 3, 6)

		if len(removed) != 3 {
			t.Fatalf("removed len must be %d, but is %d: %v", 3, len(removed), removed)
		}
	})
}
