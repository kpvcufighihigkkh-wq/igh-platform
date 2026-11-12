package queue

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

type QueueItem struct {
	ID        int64     `gorm:"primaryKey"`
	EventID   string    `gorm:"size:128;index"`
	Seq       int64     `gorm:"index"`
	EventType string    `gorm:"size:32"`
	EdgeID    string    `gorm:"size:64"`
	Payload   string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"index"`
}

type Queue struct {
	db     *gorm.DB
	mu     sync.Mutex
	closed bool
	// seq 是本地单调递增序号，用于生成稳定的事件 ID 与排查丢件。
	seq int64
}

func New(db *gorm.DB) *Queue {
	db.AutoMigrate(&QueueItem{})
	q := &Queue{db: db}
	// 恢复序号：取已有最大值，保证重启后继续递增
	var maxSeq int64
	db.Model(&QueueItem{}).Select("COALESCE(MAX(seq),0)").Scan(&maxSeq)
	q.seq = maxSeq
	depth := q.Depth()
	if depth > 0 {
		logger.L.Infow("recovered offline queue", "count", depth)
	}
	return q
}

// Enqueue 写入一条待上传事件。
//
// 关键：EventID 一旦生成就不再改变。outbound 上传失败后会把同一个事件对象
// 重新入队，此时 EventID 保持不变，服务端据此幂等去重。
func (q *Queue) Enqueue(event types.EdgeEvent) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}

	if event.EventID == "" {
		q.seq++
		event.EventID = fmt.Sprintf("%s-%d-%d", event.EdgeID, q.seq, time.Now().UnixNano())
		event.Seq = q.seq
	}

	payload, _ := json.Marshal(event.Payload)
	item := QueueItem{
		EventID:   event.EventID,
		Seq:       event.Seq,
		EventType: event.EventType,
		EdgeID:    event.EdgeID,
		Payload:   string(payload),
		CreatedAt: event.Timestamp,
	}
	if err := q.db.Create(&item).Error; err != nil {
		logger.L.Errorw("enqueue failed", "error", err, "event_id", event.EventID)
	}
}

func (q *Queue) Drain() []types.EdgeEvent {
	q.mu.Lock()
	defer q.mu.Unlock()

	var items []QueueItem
	q.db.Order("id ASC").Limit(500).Find(&items)
	if len(items) == 0 {
		return nil
	}

	events := make([]types.EdgeEvent, len(items))
	ids := make([]int64, len(items))
	for i, item := range items {
		var payload map[string]any
		json.Unmarshal([]byte(item.Payload), &payload)
		events[i] = types.EdgeEvent{
			EdgeID:    item.EdgeID,
			EventID:   item.EventID,
			Seq:       item.Seq,
			EventType: item.EventType,
			Timestamp: item.CreatedAt,
			Payload:   payload,
		}
		ids[i] = item.ID
	}

	q.db.Delete(&QueueItem{}, ids)
	return events
}

func (q *Queue) Depth() int {
	var count int64
	q.db.Model(&QueueItem{}).Count(&count)
	return int(count)
}

func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
}
