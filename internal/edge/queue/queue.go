package queue

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

type Queue struct {
	dir    string
	items  []types.EdgeEvent
	mu     sync.Mutex
	closed bool
}

func New(dir string) *Queue {
	os.MkdirAll(dir, 0755)
	q := &Queue{dir: dir}
	q.loadFromDisk()
	return q
}

func (q *Queue) Enqueue(event types.EdgeEvent) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.items = append(q.items, event)
	q.persistToDisk()
}

func (q *Queue) Drain() []types.EdgeEvent {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil
	}
	events := q.items
	q.items = nil
	q.persistToDisk()
	return events
}

func (q *Queue) Depth() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.persistToDisk()
}

func (q *Queue) persistToDisk() {
	path := filepath.Join(q.dir, "queue.json")
	data, err := json.Marshal(q.items)
	if err != nil {
		logger.L.Errorw("persist queue failed", "error", err)
		return
	}
	os.WriteFile(path, data, 0644)
}

func (q *Queue) loadFromDisk() {
	path := filepath.Join(q.dir, "queue.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var items []types.EdgeEvent
	if json.Unmarshal(data, &items) == nil {
		q.items = items
		logger.L.Infow("recovered offline queue", "count", len(items))
	}
}
