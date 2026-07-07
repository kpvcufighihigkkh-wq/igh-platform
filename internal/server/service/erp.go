package service

import (
	"encoding/json"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"gorm.io/gorm"
)

type ERPConfig struct {
	Mode     string `yaml:"mode"`
	PushOnly struct {
		Target     string   `yaml:"target"`
		Tables     []string `yaml:"tables"`
		Retry      int      `yaml:"retry"`
		TimeoutSec int      `yaml:"timeout_sec"`
	} `yaml:"push_only"`
}

type ERPService struct {
	db     *gorm.DB
	config ERPConfig
}

func NewERPService(db *gorm.DB, cfg ERPConfig) *ERPService {
	return &ERPService{db: db, config: cfg}
}

func (s *ERPService) IsEnabled() bool {
	return s.config.Mode != "" && s.config.Mode != "disabled"
}

func (s *ERPService) EnqueueSync(target string, recordID int64, payload any) error {
	if !s.IsEnabled() {
		return nil
	}
	data, _ := json.Marshal(payload)
	record := &model.ERPSyncQueue{
		Target:    target,
		RecordID:  recordID,
		Payload:   string(data),
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	return s.db.Create(record).Error
}

func (s *ERPService) ListQueue(status string, page, pageSize int) ([]model.ERPSyncQueue, int64, error) {
	var items []model.ERPSyncQueue
	var total int64
	q := s.db.Model(&model.ERPSyncQueue{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return items, total, err
}

func (s *ERPService) RetryFailed() (int, error) {
	result := s.db.Model(&model.ERPSyncQueue{}).
		Where("status = ? AND retry_count < ?", "failed", s.config.PushOnly.Retry).
		Updates(map[string]any{"status": "pending"})
	return int(result.RowsAffected), result.Error
}

func (s *ERPService) ProcessQueue() {
	if !s.IsEnabled() {
		return
	}
	var items []model.ERPSyncQueue
	s.db.Where("status = ?", "pending").Order("created_at").Limit(50).Find(&items)

	for _, item := range items {
		err := s.pushToERP(item)
		if err != nil {
			s.db.Model(&item).Updates(map[string]any{
				"status":        "failed",
				"retry_count":   gorm.Expr("retry_count + 1"),
				"error_message": err.Error(),
			})
			logger.L.Warnw("erp sync failed", "target", item.Target, "id", item.RecordID, "error", err)
		} else {
			now := time.Now()
			s.db.Model(&item).Updates(map[string]any{"status": "sent", "sent_at": now})
		}
	}
}

func (s *ERPService) pushToERP(item model.ERPSyncQueue) error {
	// TODO: actual push implementation based on config.Mode
	// push_only: write to MSSQL target
	// bidirectional: POST to API endpoint
	logger.L.Infow("erp push", "target", item.Target, "record_id", item.RecordID)
	return nil
}
