package gokugen

import (
	"sync"
)

type WorkRegistry struct {
	m sync.Map
}

func NewWorkRegistry() *WorkRegistry {
	return &WorkRegistry{}
}

func (m *WorkRegistry) Delete(key string) {
	m.m.Delete(key)
}

func (m *WorkRegistry) Load(key string) (value WorkFnWParam, ok bool) {
	v, ok := m.m.Load(key)
	if !ok {
		return
	}
	return v.(WorkFnWParam), ok
}

func (m *WorkRegistry) LoadAndDelete(key string) (value WorkFnWParam, loaded bool) {
	v, ok := m.m.LoadAndDelete(key)
	return v.(WorkFnWParam), ok
}

func (m *WorkRegistry) LoadOrStore(key string, value WorkFnWParam) (actual WorkFnWParam, loaded bool) {
	v, loaded := m.m.LoadOrStore(key, value)
	return v.(WorkFnWParam), loaded
}

func (m *WorkRegistry) Range(f func(key string, value WorkFnWParam) bool) {
	walker := func(k, v any) bool {
		return f(k.(string), v.(WorkFnWParam))
	}
	m.m.Range(walker)
}

func (m *WorkRegistry) Store(key string, value WorkFnWParam) {
	m.m.Store(key, value)
}
