package taskstorage

import "sync"

type SyncStateStore struct {
	m *sync.Map
}

func NewSyncStateStore() *SyncStateStore {
	return &SyncStateStore{
		m: new(sync.Map),
	}
}

func (s *SyncStateStore) Put(v string, state TaskState) {
	s.m.Store(v, state)
}

func (s *SyncStateStore) Remove(v string) (removed bool) {
	_, removed = s.m.LoadAndDelete(v)
	return
}

func (s *SyncStateStore) Has(v string) (has bool) {
	_, has = s.m.Load(v)
	return
}

func (s *SyncStateStore) Len() int {
	var count int
	s.m.Range(func(key, value any) bool { count++; return true })
	return count
}

type TaskStateSet struct {
	Key   string
	Value TaskState
}

func (s *SyncStateStore) GetAll() []TaskStateSet {
	sl := make([]TaskStateSet, 0)
	s.m.Range(func(k, v any) bool {
		sl = append(sl, TaskStateSet{Key: k.(string), Value: v.(TaskState)})
		return true
	})
	return sl
}

func (s *SyncStateStore) Clone() *SyncStateStore {
	m := new(sync.Map)
	s.m.Range(func(k, v any) bool {
		m.Store(k.(string), v.(TaskState))
		return true
	})
	return &SyncStateStore{
		m: m,
	}
}
