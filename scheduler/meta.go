package scheduler

import (
	"sync"
)

var (
	metaNameLock sync.RWMutex
	metaMap      = make(map[string]bool)
)

func RegisterMeta(name string) {
	metaNameLock.Lock()
	defer metaNameLock.Unlock()
	if metaMap[name] {
		panic(
			"github.com/ngicks/gokugen/scheduler: " +
				"RegisterMeta is called twice for the meta named " + name,
		)
	}
	metaMap[name] = true
}
