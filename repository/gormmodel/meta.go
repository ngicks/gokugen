package gormmodel

import (
	"sort"
)

type MetaKeyValue struct {
	TaskId string `json:"task_id" gorm:"primaryKey"`
	Key    string `json:"key" gorm:"primaryKey"`
	Value  string `json:"value"`
}

type Meta []MetaKeyValue

func FromMeta(taskId string, m map[string]string) Meta {
	out := make([]MetaKeyValue, len(m))

	idx := 0
	for k, v := range m {
		out[idx] = MetaKeyValue{TaskId: taskId, Key: k, Value: v}
		idx++
	}

	// The iteration order of a map[T]U is intentionally unstable.
	// Make it stable here.
	sort.Slice(out, func(i, j int) bool {
		return out[i].Key < out[j].Key
	})

	return out
}

func (m Meta) ToMeta() map[string]string {
	out := make(map[string]string, len(m))
	for _, kv := range m {
		out[kv.Key] = kv.Value
	}
	return out
}
