package workregistry

import (
	"encoding/json"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/ngicks/gokugen"
	"github.com/ngicks/gokugen/cron"
	syncparam "github.com/ngicks/type-param-common/sync-param"
)

type Unmarshaller = func(naive any) (unmarshalled any, err error)

// UnmarshalAny is reference Unmarshaller implementation.
// If naive is T it simply returns naive.
// If naive is []byte then unmarshal it as a json string.
// Else, UnmarshalAny treats naive as naively unmarshalled json value
// (e.g. map[any]any, []any or other literal values).
// It copies values from naive to unmarshaled through reflection.
func UnmarshalAny[T any](naive any) (unmarshalled any, err error) {
	if _, ok := naive.(T); ok {
		return naive, nil
	}

	var val T

	if bin, ok := naive.([]byte); ok {
		err = json.Unmarshal(bin, &val)
		if err != nil {
			return
		}
		unmarshalled = val
		return
	}

	err = mapstructure.Decode(naive, &val)
	if err != nil {
		return
	}
	return
}

type UnmarshallerRegistry interface {
	Load(key string) (value Unmarshaller, ok bool)
}

type AnyUnmarshaller struct {
	syncparam.Map[string, Unmarshaller]
}

// AddType stores UnmarshalAny, which is instantiated with T, into registry.
// AddType is external function, not a method for *AnyUnmarshaller,
// since Go curently does not allow us to add type parameters to methods.
func AddType[T any](key string, registry *AnyUnmarshaller) {
	registry.Store(key, UnmarshalAny[T])
}

type ParamUnmarshaller struct {
	inner              cron.WorkRegistry
	marshallerRegistry UnmarshallerRegistry
}

func (p *ParamUnmarshaller) Load(key string) (value gokugen.WorkFnWParam, ok bool) {
	work, ok := p.inner.Load(key)
	if !ok {
		return
	}

	unmarshaller, ok := p.marshallerRegistry.Load(key)
	if !ok {
		return
	}

	value = func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time, param any) (any, error) {
		unmarshaled, err := unmarshaller(param)
		if err != nil {
			return nil, err
		}
		return work(ctxCancelCh, taskCancelCh, scheduled, unmarshaled)
	}

	return
}
