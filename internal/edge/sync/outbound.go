package sync

import (
	"github.com/igh-platform/igh-platform/internal/edge/handler"
	"github.com/igh-platform/igh-platform/internal/edge/queue"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

type Outbound struct {
	transport *handler.Transport
	queue     *queue.Queue
}

func NewOutbound(transport *handler.Transport, q *queue.Queue) *Outbound {
	return &Outbound{transport: transport, queue: q}
}

func (o *Outbound) Flush() int {
	events := o.queue.Drain()
	if len(events) == 0 {
		return 0
	}

	result, err := o.transport.UploadBatch(events)
	if err != nil {
		// 服务端没确认落库（网络失败，或返回 503 部分失败）。
		// 整批重新入队：已成功的事件靠 (edge_id, event_id) 幂等去重，重传安全。
		for _, e := range events {
			o.queue.Enqueue(e)
		}
		logger.L.Warnw("batch upload failed, re-queued", "count", len(events), "error", err)
		return 0
	}

	// 服务端返回 200 但仍有 rejected：其中 retryable=false 的是永久错误
	// （事件本身不合法），重试无意义，只能丢弃并告警，避免毒丸堵死队列。
	if dropped := countPermanent(result); dropped > 0 {
		logger.L.Errorw("batch upload dropped permanently-invalid events",
			"count", dropped, "total", len(events), "edge_id", o.transport.EdgeID())
	}
	if dup := countData(result, "duplicate"); dup > 0 {
		logger.L.Debugw("server reported duplicates", "count", dup)
	}

	logger.L.Infow("synced events to server", "count", len(events),
		"accepted", countData(result, "accepted"))
	return len(events)
}

// countData 取服务端返回的计数字段（JSON 解出来是 float64）。
func countData(result map[string]any, key string) int {
	data, ok := result["data"].(map[string]any)
	if !ok {
		return 0
	}
	switch n := data[key].(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

// countPermanent 统计不可重试的拒绝事件数。
func countPermanent(result map[string]any) int {
	data, ok := result["data"].(map[string]any)
	if !ok {
		return 0
	}
	items, ok := data["rejected_items"].([]any)
	if !ok {
		return 0
	}
	n := 0
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		if retryable, _ := m["retryable"].(bool); !retryable {
			n++
		}
	}
	return n
}
