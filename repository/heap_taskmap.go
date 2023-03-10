package repository

import (
	"strings"
	"time"

	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/type-param-common/util"
)

type TaskMap struct {
	Scheduled map[string]scheduler.Task
	Cancelled map[string]scheduler.Task
	Done      map[string]scheduler.Task
	Deleted   map[string]scheduler.Task
}

func (tm TaskMap) IsInitialized() bool {
	return tm.Scheduled != nil && tm.Cancelled != nil && tm.Done != nil && tm.Deleted != nil
}

// taskMap is map-like part of HeapRepository.
// HeapRepository delegates functionalities, such as O(1) access to tasks,
// finding tasks or state changes, to taskMap.
type taskMap struct {
	Scheduled map[string]*wrappedTask
	Done      map[string]*wrappedTask
	Cancelled map[string]*wrappedTask
	Deleted   map[string]*wrappedTask
}

func newTaskMap(size ...int) taskMap {
	var mapSize [4]int
	if len(size) != 0 {
		copy(mapSize[:], size)
	}

	return taskMap{
		Scheduled: make(map[string]*wrappedTask, mapSize[0]),
		Done:      make(map[string]*wrappedTask, mapSize[1]),
		Cancelled: make(map[string]*wrappedTask, mapSize[2]),
		Deleted:   make(map[string]*wrappedTask, mapSize[3]),
	}
}

func fromExternal(tm TaskMap) taskMap {
	ret := newTaskMap(len(tm.Scheduled), len(tm.Done), len(tm.Cancelled), len(tm.Deleted))

	for id, task := range tm.Scheduled {
		ret.Scheduled[id] = &wrappedTask{Task: task, Index: -1}
	}
	for id, task := range tm.Cancelled {
		ret.Cancelled[id] = &wrappedTask{Task: task, Index: -1}
	}
	for id, task := range tm.Done {
		ret.Done[id] = &wrappedTask{Task: task, Index: -1}
	}
	for id, task := range tm.Deleted {
		ret.Deleted[id] = &wrappedTask{Task: task, Index: -1}
	}
	return ret
}

func (tm *taskMap) Init() {

}

func (tm taskMap) Dump() TaskMap {
	return TaskMap{
		Scheduled: cloneUnwrapping(tm.Scheduled),
		Cancelled: cloneUnwrapping(tm.Cancelled),
		Done:      cloneUnwrapping(tm.Done),
		Deleted:   cloneUnwrapping(tm.Deleted),
	}
}

func cloneUnwrapping(src map[string]*wrappedTask) map[string]scheduler.Task {
	out := make(map[string]scheduler.Task, len(src))
	for id, task := range src {
		out[id] = task.Task
	}
	return out
}

func copyUnwrapping(dst map[string]scheduler.Task, src map[string]*wrappedTask) {
	for id, task := range src {
		dst[id] = task.Task
	}
}

func (tm taskMap) IsInitialized() bool {
	return tm.Scheduled != nil && tm.Cancelled != nil && tm.Done != nil && tm.Deleted != nil
}

func (tm *taskMap) Get(id string) (task *wrappedTask, ok bool) {
	if t, ok := tm.Scheduled[id]; ok {
		return t, true
	}
	if t, ok := tm.Done[id]; ok {
		return t, true
	}
	if t, ok := tm.Cancelled[id]; ok {
		return t, true
	}
	// ignore deleted elements.
	return nil, false
}

func (tm *taskMap) Add(task *wrappedTask) {
	tm.Scheduled[task.Id] = task
}

func (tm *taskMap) SetCancelled(id string, now time.Time) {
	t, ok := tm.Scheduled[id]
	if ok {
		t.CancelledAt = util.Escape(now)
		tm.Cancelled[id] = t
		delete(tm.Scheduled, id)
	}
}
func (tm *taskMap) SetDone(id string, now time.Time, err error) {
	t, ok := tm.Scheduled[id]
	if ok {
		t.DoneAt = util.Escape(now)
		if err != nil {
			t.Err = err.Error()
		}
		tm.Done[id] = t
		delete(tm.Scheduled, id)
	}
}

func (tm *taskMap) Delete(id string) {
	t, ok := tm.Get(id)
	if !ok {
		return
	}
	delete(tm.Scheduled, id)
	delete(tm.Cancelled, id)
	delete(tm.Done, id)
	tm.Deleted[id] = t
}

func (tm *taskMap) RemoveDone() map[string]*wrappedTask {
	out := tm.Done
	tm.Done = make(map[string]*wrappedTask)
	return out
}

func (tm *taskMap) RemoveCancelled() map[string]*wrappedTask {
	out := tm.Cancelled
	tm.Cancelled = make(map[string]*wrappedTask)
	return out
}

func (tm *taskMap) RemoveDeleted() map[string]*wrappedTask {
	out := tm.Deleted
	tm.Deleted = make(map[string]*wrappedTask)
	return out
}

func (tm *taskMap) Find(t scheduler.TaskMatcher) []scheduler.Task {
	matched := make([]scheduler.Task, 0)

	var maps []map[string]*wrappedTask
	if t.DoneAt != nil {
		maps = []map[string]*wrappedTask{tm.Done}
	} else if t.CancelledAt != nil {
		maps = []map[string]*wrappedTask{tm.Cancelled}
	} else {
		maps = []map[string]*wrappedTask{tm.Done, tm.Cancelled, tm.Scheduled}
	}

	for _, container := range maps {
		for _, task := range container {
			if task.Match(t) {
				matched = append(matched, task.Task)
			}
		}
	}

	return matched
}

func (tm *taskMap) FindMetaContain(matcher []scheduler.KeyValuePairMatcher) []scheduler.Task {
	out := make([]scheduler.Task, 0)

	if len(matcher) == 0 {
		return out
	}

	for _, container := range [...]map[string]*wrappedTask{
		tm.Done,
		tm.Cancelled,
		tm.Scheduled,
	} {
		for _, task := range container {
			var i int
			for i = 0; i < len(matcher); i++ {
				if v, ok := task.Meta[matcher[i].Key]; ok {
					matched := false
					switch matcher[i].MatchTy {
					case scheduler.HasKey:
						matched = true
					case scheduler.Exact:
						matched = v == matcher[i].Value
					case scheduler.Forward:
						matched = strings.HasPrefix(v, matcher[i].Value)
					case scheduler.Backward:
						matched = strings.HasSuffix(v, matcher[i].Value)
					case scheduler.Partial:
						matched = strings.Contains(v, matcher[i].Value)
					}
					if matched {
						out = append(out, task.Task)
					}
				}
			}
		}
	}

	return out
}
