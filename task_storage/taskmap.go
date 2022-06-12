package taskstorage

import (
	"sync"

	"github.com/ngicks/gokugen"
)

type TaskMap struct {
	m *sync.Map
}

func NewTaskMap() *TaskMap {
	return &TaskMap{
		m: new(sync.Map),
	}
}

func (m *TaskMap) LoadOrStore(k string, v gokugen.Task) (actual gokugen.Task, loaded bool) {
	act, loaded := m.m.LoadOrStore(k, v)
	actual = act.(gokugen.Task)
	return
}

func (m *TaskMap) Has(k string) (has bool) {
	_, has = m.m.Load(k)
	return
}

func (m *TaskMap) Clone() *TaskMap {
	cloned := make(map[string]gokugen.Task)
	m.m.Range(func(key, value any) bool {
		cloned[key.(string)] = value.(gokugen.Task)
		return true
	})

	synMap := new(sync.Map)
	for k, v := range cloned {
		synMap.Store(k, v)
	}
	return &TaskMap{
		m: synMap,
	}
}

func (m *TaskMap) LoadAndDelete(k string) (task gokugen.Task, loaded bool) {
	v, loaded := m.m.LoadAndDelete(k)
	if !loaded {
		return
	}
	task = v.(gokugen.Task)
	return
}

func (m *TaskMap) Delete(k string) (deleted bool) {
	_, deleted = m.m.LoadAndDelete(k)
	return
}

func (m *TaskMap) AllIds() []string {
	ids := make([]string, 0)
	m.m.Range(func(key, value any) bool {
		ids = append(ids, key.(string))
		return true
	})
	return ids
}
