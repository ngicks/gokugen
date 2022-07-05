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
	Store(key string, value Transformer)
	Load(key string) (value Transformer, ok bool)
}

type TransformerRegistryImpl struct {
	syncparam.Map[string, Transformer]
}

// AddType stores TransformedAny instantiated with T into the registry.
// AddType is an external function, not a method for *AnyUnmarshaller,
// since Go curently does not allow us to add type parameters to methods individually.
func AddType[T any](key string, registry TransformerRegistry) {
	registry.Store(key, TransformedAny[T])
}

type Storer interface {
	Store(key string, value cron.WorkFnWParam)
}

// RegisterDefault registers default transformer TransformedAny[T] to registry.
// typed workFn will be called with transformed T value processed by those defaut ones.
//
// WorkRegistry of registry must implement Storer interface. If not, returns false and no registration occurs.
func RegisterDefault[T any](
	key string,
	registry *ParamTransformer,
	workFn func(taskCtx context.Context, scheduled time.Time, param T) (any, error),
) (stored bool) {
	storer, ok := registry.inner.(Storer)
	if !ok {
		return false
	}

	AddType[T](key, registry.transformerRegistry)

	storer.Store(key, func(taskCtx context.Context, scheduled time.Time, param any) (any, error) {
		return workFn(taskCtx, scheduled, param.(T))
	})

	return true
}

type ParamTransformer struct {
	inner               cron.WorkRegistry
	transformerRegistry TransformerRegistry
}

func NewParamTransformer(
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

	tansformer, ok := p.transformerRegistry.Load(key)
	if !ok {
		return
	}

	value = func(taskCtx context.Context, scheduled time.Time, param any) (any, error) {
		unmarshaled, err := tansformer(param)
		if err != nil {
			return nil, err
		}
		return work(taskCtx, scheduled, unmarshaled)
	}

	return
}
