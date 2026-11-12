package plugin

import (
	"fmt"
	"sync"
)

var (
	mu       sync.RWMutex
	registry = map[string]StationPlugin{}
)

func Register(p StationPlugin) {
	mu.Lock()
	defer mu.Unlock()
	registry[p.Type()] = p
}

func Get(stationType string) (StationPlugin, error) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[stationType]
	if !ok {
		return nil, fmt.Errorf("unknown station type: %s", stationType)
	}
	return p, nil
}

func All() []StationPlugin {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]StationPlugin, 0, len(registry))
	for _, p := range registry {
		out = append(out, p)
	}
	return out
}
