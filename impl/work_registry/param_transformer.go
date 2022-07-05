package workregistry

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/ngicks/gokugen"
	"github.com/ngicks/gokugen/cron"
	syncparam "github.com/ngicks/type-param-common/sync-param"
)

type Transformer = func(naive any) (transformed any, err error)

// TransformedAny is reference Transformer implementation.
// If naive is T it simply returns naive.
// If naive is []byte then unmarshal it, as a json string, to T.
// Else, TransformedAny treats naive as naively unmarshalled json value
// (e.g. map[any]any, []any or other literal values).
// It copies values from naive to transformed through reflection.
func TransformedAny[T any](naive any) (transformed any, err error) {
	if _, ok := naive.(T); ok {
		return naive, nil
	}

	var val T

	if bin, ok := naive.([]byte); ok {
		err = json.Unmarshal(bin, &val)
		if err != nil {
			return
		}
		transformed = val
		return
	}

	err = mapstructure.Decode(naive, &val)
	if err != nil {
		return
	}
	return val, nil
}

type TransformerRegistry interface {
	Load(key string) (value Transformer, ok bool)
}

type TransformerRegistryImpl struct {
	syncparam.Map[string, Transformer]
}

// AddType stores TransformedAny instantiated with T into the registry.
// AddType is an external function, not a method for *AnyUnmarshaller,
// since Go curently does not allow us to add type parameters to methods individually.
func AddType[T any](key string, registry *TransformerRegistryImpl) {
	registry.Store(key, TransformedAny[T])
}

type Storer interface {
	Store(key string, value cron.WorkFnWParam)
}

type ParamTransformer struct {
	inner               cron.WorkRegistry
	transformerRegistry TransformerRegistry
}

func NewParamUnmarshaller(
	inner cron.WorkRegistry,
	marshallerRegistry TransformerRegistry,
) *ParamTransformer {
	return &ParamTransformer{
		inner:               inner,
		transformerRegistry: marshallerRegistry,
	}
}

func (p *ParamTransformer) Load(key string) (value gokugen.WorkFnWParam, ok bool) {
	work, ok := p.inner.Load(key)
	if !ok {
		return
	}

	unmarshaller, ok := p.transformerRegistry.Load(key)
	if !ok {
		return
	}

	value = func(taskCtx context.Context, scheduled time.Time, param any) (any, error) {
		unmarshaled, err := unmarshaller(param)
		if err != nil {
			return nil, err
		}
		return work(taskCtx, scheduled, unmarshaled)
	}

	return
}
