package workregistry

import (
	"strings"
	"sync"

	"github.com/ngicks/gokugen/cron"
)

// Prefix is simple prefixer of WorkRegistry.
// It stores registries with prefix associated to them.
//
// Main purpose of Prefix is mixing Cli and other manually registered functions.
// e.g. prefix cli commands with `cli:` like `cli:ls` and other function with `gofunc:`
type Prefix struct {
	mu     sync.Mutex
	others map[string]cron.WorkRegistry
}

type PrefixOption func(p *Prefix) *Prefix

// PrefixAddMap is option to NewPrefix.
// PrefixAddMap adds regitries by passing map.
// Key must be prefix string. Value must be work registry paired to prefix.
func PrefixAddMap(m map[string]cron.WorkRegistry) PrefixOption {
	return func(p *Prefix) *Prefix {
		for k, v := range m {
			p.others[k] = v
		}
		return p
	}
}

func PrefixAddRegistry(prefix string, registry cron.WorkRegistry) PrefixOption {
	return func(p *Prefix) *Prefix {
		p.others[prefix] = registry
		return p
	}
}

func NewPrefix(options ...PrefixOption) *Prefix {
	p := &Prefix{
		others: make(map[string]cron.WorkRegistry),
	}
	for _, opt := range options {
		p = opt(p)
	}
	return p
}

// Load loads work function associated with given prefixedKey.
// Load assumes that key is prefixed with known keyword.
// It removes prefix and then loads work function from a registry paired with the prefix.
func (p *Prefix) Load(prefixedKey string) (value cron.WorkFnWParam, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for prefix, registry := range p.others {
		if strings.HasPrefix(prefixedKey, prefix) {
			return registry.Load(prefixedKey[len(prefix):])
		}
	}
	return
}
