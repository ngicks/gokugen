package gokugen_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ngicks/gokugen"
	gosched "github.com/ngicks/gokugen"
)

type set struct {
	m map[string]struct{}
}

func (s *set) Add(v ...string) {
	for _, vv := range v {
		s.m[vv] = struct{}{}
	}
}

func (s *set) Remove(v string) (removed bool) {
	_, removed = s.m[v]
	delete(s.m, v)
	return
}

func (s *set) Keys() (keys []string) {
	for k := range s.m {
		keys = append(keys, k)
	}
	return
}

func TestWorkRegistry(t *testing.T) {
	wr := gosched.NewWorkRegistry()

	var fn1 gokugen.WorkFnWParam = func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time, param any) error {
		fmt.Println("foo")
		return nil
	}

	var fn2 gokugen.WorkFnWParam = func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time, param any) error {
		fmt.Println("bar")
		return nil
	}

	if val, ok := wr.Load("aaa"); val != nil || ok {
		t.Errorf("must be empty fn")
	}

	{
		actual, loaded := wr.LoadOrStore("foo", fn1)
		if reflect.ValueOf(actual).Pointer() != reflect.ValueOf(fn1).Pointer() || loaded {
			t.Errorf("failed to store fn to an unused key")
		}
	}
	{
		actual, loaded := wr.LoadOrStore("foo", fn2)
		if reflect.ValueOf(actual).Pointer() != reflect.ValueOf(fn1).Pointer() || !loaded {
			t.Errorf("failed to load fn from an already used key")
		}
	}
	{
		actual, loaded := wr.Load("foo")
		if reflect.ValueOf(actual).Pointer() != reflect.ValueOf(fn1).Pointer() || !loaded {
			t.Errorf("failed to load fn from an already used key")
		}
	}
	{
		wr.Delete("foo")
		actual, loaded := wr.Load("foo")
		if actual != nil || loaded {
			t.Errorf("failed to load fn from an already used key")
		}
		wr.Delete("foo")
		wr.Delete("foo")
		wr.Delete("foo")
		wr.Delete("barbarbar")
	}
	{
		wr.LoadOrStore("foo", fn2)
		actual, loaded := wr.LoadAndDelete("foo")
		if reflect.ValueOf(actual).Pointer() != reflect.ValueOf(fn2).Pointer() || !loaded {
			t.Errorf("failed to load fn from an already used key")
		}
	}
	{
		actual, loaded := wr.Load("foo")
		if actual != nil || loaded {
			t.Errorf("failed to load fn from an already used key")
		}
	}
	{
		wr.Store("foo", fn1)
		wr.Store("foo", fn2)
		actual, loaded := wr.Load("foo")
		if reflect.ValueOf(actual).Pointer() != reflect.ValueOf(fn2).Pointer() || !loaded {
			t.Errorf("failed to load fn from an already used key")
		}
		wr.Delete("foo")
	}
	{
		wr.Store("foo", fn1)
		wr.Store("bar", fn2)
		wr.Store("baz", fn1)
		wr.Store("qux", fn2)
		wr.Store("quux", nil)

		keySet := &set{
			m: make(map[string]struct{}),
		}

		keySet.Add("foo", "bar", "baz", "qux", "quux")

		sliceContain := func(tgt string, sl []string) bool {
			for _, v := range sl {
				if tgt == v {
					return true
				}
			}
			return false
		}
		wr.Range(func(key string, value gokugen.WorkFnWParam) bool {
			keySet.Remove(key)
			if !sliceContain(key, []string{"foo", "bar", "baz", "qux", "quux"}) {
				t.Errorf("unknown key=%s", key)
			}
			ptr := reflect.ValueOf(value).Pointer()
			switch key {
			case "foo", "baz":
				if ptr != reflect.ValueOf(fn1).Pointer() {
					t.Errorf("unknown value=%v", ptr)
				}
			case "bar", "qux":
				if ptr != reflect.ValueOf(fn2).Pointer() {
					t.Errorf("unknown value=%v", ptr)
				}
			case "quux":
				if value != nil {
					t.Errorf("not nil=%v", ptr)
				}
			}
			return true
		})
		if len(keySet.Keys()) != 0 {
			t.Errorf("keys not visited=%v", keySet.Keys())
		}
	}
}
