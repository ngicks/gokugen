package workregistry

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/ngicks/gokugen"
	"github.com/ngicks/gokugen/cron"
	"github.com/ngicks/type-param-common/set"
)

var _ cron.WorkRegistry = &Cli{}

// Cli builds work function from given string assuming it is a cli command.
// Cli also holds whiteList and blackList.
type Cli struct {
	mu        sync.Mutex
	whiteList set.Set[string]
	backList  set.Set[string]
}

type CliOption func(c *Cli) *Cli

func CliWhiteList(whilteList []string) CliOption {
	return func(c *Cli) *Cli {
		for _, w := range whilteList {
			c.whiteList.Add(w)
		}
		return c
	}
}
func CliBlackList(backList []string) func(c *Cli) *Cli {
	return func(c *Cli) *Cli {
		for _, w := range backList {
			c.backList.Add(w)
		}
		return c
	}
}

func NewCli(options ...CliOption) *Cli {
	c := new(Cli)
	for _, opt := range options {
		c = opt(c)
	}
	return c
}

func (c *Cli) isKeyAllowed(key string) (ok bool) {
	if c.backList.Len() == 0 && c.whiteList.Len() == 0 {
		return true
	}
	if c.backList.Len() != 0 {
		if c.backList.Has(key) {
			return false
		} else {
			return false
		}
	}
	if c.whiteList.Len() != 0 {
		if c.whiteList.Has(key) {
			return true
		} else {
			return false
		}
	}
	// unreachable?
	return false
}

func (c *Cli) Load(key string) (value gokugen.WorkFnWParam, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ok = c.isKeyAllowed(key)
	if !ok {
		return
	}
	value = buildCliWorkFn(key)
	return
}

func buildCliWorkFn(command string) cron.WorkFnWParam {
	return func(taskCtx context.Context, scheduled time.Time, param any) (v any, err error) {

		wg := sync.WaitGroup{}
		wg.Add(1)

		cmd := exec.CommandContext(taskCtx, command, serializeParam(param)...)
		b, err := cmd.Output()
		if b != nil {
			v = string(b)
		}

		wg.Wait()
		return
	}
}

func serializeParam(input any) (params []string) {
	if input == nil {
		return []string{}
	}
	switch x := input.(type) {
	case []string:
		return x
	case string:
		return []string{x}
	case []byte:
		return []string{string(x)}
	default:
		str := fmt.Sprintf("%s", x)
		if str != "" {
			return []string{str}
		} else {
			return []string{}
		}
	}
}
