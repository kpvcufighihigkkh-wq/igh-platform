package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EdgeEventService 负责边端事件的**幂等接收**。
//
// 为什么需要它：
//
//	边端采用"至少一次"投递（网络失败会重试、离线队列会补传），
//	因此服务端必须能识别重复事件，否则一次重试就会产生重复的落筒/托盘/订单。
//
// 语义：
//   - Ingest 返回 (record, isNew, err)
//   - isNew == false 表示是重复事件，调用方应直接返回成功，且**不得再次分发**
//
// 唯一键：(edge_id, event_id)。event_id 为空时，退化为 edge_id:seq。
type EdgeEventService struct {
	db *gorm.DB
}

func NewEdgeEventService(db *gorm.DB) *EdgeEventService {
	return &EdgeEventService{db: db}
}

// ErrMissingEventKey 表示事件既没有 event_id 也没有 seq，无法去重。
var ErrMissingEventKey = errors.New("edge event: event_id and seq are both empty")

// Ingest 幂等写入一条事件。
//
// 返回 isNew=true 表示首次收到，调用方应继续处理；
// 返回 isNew=false 表示重复上报，调用方应直接返回成功。
func (s *EdgeEventService) Ingest(ev types.EdgeEvent) (*model.EdgeEventRecord, bool, error) {
	eventID := ev.EventID
	if eventID == "" {
		if ev.Seq == 0 {
			return nil, false, ErrMissingEventKey
		}
		eventID = fmt.Sprintf("%s:%d", ev.EdgeID, ev.Seq)
	}
	if ev.EdgeID == "" {
		return nil, false, errors.New("edge event: edge_id is required")
	}

	payload, err := json.Marshal(ev.Payload)
	if err != nil {
		return nil, false, fmt.Errorf("marshal payload: %w", err)
	}

	occurredAt := ev.Timestamp
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}

	record := &model.EdgeEventRecord{
		EdgeID:     ev.EdgeID,
		EventID:    eventID,
		Seq:        ev.Seq,
		EventType:  ev.EventType,
		OccurredAt: occurredAt,
		Payload:    string(payload),
		Status:     "received",
		ReceivedAt: time.Now(),
	}

	// ON CONFLICT DO NOTHING + RowsAffected 是原子操作，
	// 不会出现"先查再插"的竞态（并发重试也能正确去重）。
	res := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(record)
	if res.Error != nil {
		return nil, false, fmt.Errorf("insert edge event: %w", res.Error)
	}

	isNew := res.RowsAffected == 1
	if !isNew {
		logger.L.Debugw("duplicate edge event ignored",
			"edge_id", ev.EdgeID, "event_id", eventID, "type", ev.EventType)
		// 重复事件：把已存在的记录读回来，便于调用方观察
		var existing model.EdgeEventRecord
		if err := s.db.Where("edge_id = ? AND event_id = ?", ev.EdgeID, eventID).
			First(&existing).Error; err == nil {
			record = &existing
		}
	}
	return record, isNew, nil
}

// MarkProcessed 标记事件处理结果。
func (s *EdgeEventService) MarkProcessed(rec *model.EdgeEventRecord, status string, processErr error) {
	if rec == nil {
		return
	}
	now := time.Now()
	updates := map[string]any{
		"status":       status,
		"processed_at": now,
	}
	if processErr != nil {
		updates["error"] = processErr.Error()
	}
	if err := s.db.Model(&model.EdgeEventRecord{}).
		Where("id = ?", rec.ID).
		Updates(updates).Error; err != nil {
		logger.L.Errorw("mark edge event processed failed",
			"edge_id", rec.EdgeID, "event_id", rec.EventID, "error", err)
	}
}

// MarkProcessedByKey 按 (edge_id, event_id) 标记，适用于没有记录对象的场景。
func (s *EdgeEventService) MarkProcessedByKey(edgeID, eventID, status string, processErr error) error {
	now := time.Now()
	updates := map[string]any{
		"status":       status,
		"processed_at": now,
	}
	if processErr != nil {
		updates["error"] = processErr.Error()
	}
	res := s.db.Model(&model.EdgeEventRecord{}).
		Where("edge_id = ? AND event_id = ?", edgeID, eventID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("edge event not found: %s/%s", edgeID, eventID)
	}
	return nil
}

// Count 返回某边端的事件总数（诊断用）。
func (s *EdgeEventService) Count(edgeID string) (int64, error) {
	var n int64
	q := s.db.Model(&model.EdgeEventRecord{})
	if edgeID != "" {
		q = q.Where("edge_id = ?", edgeID)
	}
	err := q.Count(&n).Error
	return n, err
}
