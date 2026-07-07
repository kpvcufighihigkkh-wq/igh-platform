package service

import (
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"gorm.io/gorm"
)

type AlarmService struct {
	db *gorm.DB
}

func NewAlarmService(db *gorm.DB) *AlarmService {
	return &AlarmService{db: db}
}

func (s *AlarmService) ListAlarms(status, level string, page, pageSize int) ([]model.Alarm, int64, error) {
	var alarms []model.Alarm
	var total int64
	q := s.db.Model(&model.Alarm{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if level != "" {
		q = q.Where("level = ?", level)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&alarms).Error
	return alarms, total, err
}

func (s *AlarmService) CreateAlarm(alarm *model.Alarm) error {
	return s.db.Create(alarm).Error
}

func (s *AlarmService) Acknowledge(id, userID int64) error {
	now := time.Now()
	return s.db.Model(&model.Alarm{}).Where("id = ? AND status = ?", id, "active").
		Updates(map[string]any{
			"status":          "acknowledged",
			"acknowledged_by": userID,
			"acknowledged_at": now,
		}).Error
}

func (s *AlarmService) Resolve(id, userID int64, note string) error {
	now := time.Now()
	return s.db.Model(&model.Alarm{}).Where("id = ? AND status IN ?", id, []string{"active", "acknowledged"}).
		Updates(map[string]any{
			"status":          "resolved",
			"resolved_by":     userID,
			"resolved_at":     now,
			"resolution_note": note,
		}).Error
}

func (s *AlarmService) ActiveCount() (int64, error) {
	var count int64
	err := s.db.Model(&model.Alarm{}).Where("status = ?", "active").Count(&count).Error
	return count, err
}

func (s *AlarmService) ProcessAlarmEvent(sourceType, sourceID, alarmCode, level, message, suggestion string) (*model.Alarm, error) {
	alarm := &model.Alarm{
		SourceType: sourceType,
		SourceID:   sourceID,
		AlarmCode:  alarmCode,
		Level:      level,
		Message:    message,
		Suggestion: suggestion,
		Status:     "active",
	}
	return alarm, s.db.Create(alarm).Error
}

func (s *AlarmService) ProcessAlarmResolved(alarmCode, sourceID string) error {
	return s.db.Model(&model.Alarm{}).
		Where("alarm_code = ? AND source_id = ? AND status IN ?", alarmCode, sourceID, []string{"active", "acknowledged"}).
		Updates(map[string]any{"status": "resolved", "resolved_at": time.Now()}).Error
}
