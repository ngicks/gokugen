package scheduler

import "testing"

func TestState(t *testing.T) {
	t.Run("workingState", func(t *testing.T) {
		s := workingState{}

		if s.IsWorking() {
			t.Fatalf("initial state: must not be working")
		}

		if !s.setWorking() {
			t.Fatalf("swapped faild")
		}
		if !s.IsWorking() {
			t.Fatalf("setWorking with no arg must set it to wroking")
		}
		if !s.setWorking(false) {
			t.Fatalf("swapped faild")
		}
		if s.IsWorking() {
			t.Fatalf("setWorking with false set it to non-wroking")
		}
		if s.setWorking(false) {
			t.Fatalf("swapped faild")
		}
		if !s.setWorking(true) {
			t.Fatalf("swapped faild")
		}
		if !s.IsWorking() {
			t.Fatalf("setWorking with true set it to wroking")
		}
		if s.setWorking(true) {
			t.Fatalf("swapped faild")
		}
	})
	t.Run("endState", func(t *testing.T) {
		s := endState{}

		if s.IsEnded() {
			t.Fatalf("initial state: must not be working")
		}

		if !s.setEnded() {
			t.Fatalf("swapped faild")
		}
		if !s.IsEnded() {
			t.Fatalf("setWorking with no arg must set it to wroking")
		}
		if s.setEnded() {
			t.Fatalf("swapped faild")
		}
	})
}
