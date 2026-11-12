package service

import (
	"fmt"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"gorm.io/gorm"
)

type MaintenanceService struct {
	db *gorm.DB
}

func NewMaintenanceService(db *gorm.DB) *MaintenanceService {
	return &MaintenanceService{db: db}
}

// --- Equipment ---

func (s *MaintenanceService) ListEquipment(eqType string) ([]model.Equipment, error) {
	var items []model.Equipment
	q := s.db.Model(&model.Equipment{})
	if eqType != "" {
		q = q.Where("type = ?", eqType)
	}
	err := q.Order("code").Find(&items).Error
	return items, err
}

func (s *MaintenanceService) CreateEquipment(eq *model.Equipment) error {
	return s.db.Create(eq).Error
}

func (s *MaintenanceService) IncrementCycle(code string) error {
	return s.db.Model(&model.Equipment{}).Where("code = ?", code).
		Update("total_cycles", gorm.Expr("total_cycles + 1")).Error
}

// --- Plans ---

func (s *MaintenanceService) ListPlans(equipmentID int64) ([]model.MaintenancePlan, error) {
	var plans []model.MaintenancePlan
	q := s.db.Model(&model.MaintenancePlan{})
	if equipmentID > 0 {
		q = q.Where("equipment_id = ?", equipmentID)
	}
	err := q.Find(&plans).Error
	return plans, err
}

func (s *MaintenanceService) CreatePlan(plan *model.MaintenancePlan) error {
	if plan.TriggerType == "interval" && plan.IntervalHours != nil {
		due := time.Now().Add(time.Duration(*plan.IntervalHours) * time.Hour)
		plan.NextDueAt = &due
	}
	return s.db.Create(plan).Error
}

func (s *MaintenanceService) GetOverduePlans() ([]model.MaintenancePlan, error) {
	var plans []model.MaintenancePlan
	now := time.Now()
	err := s.db.Where("next_due_at IS NOT NULL AND next_due_at <= ?", now).Find(&plans).Error
	return plans, err
}

func (s *MaintenanceService) GetUpcomingPlans(withinHours int) ([]model.MaintenancePlan, error) {
	var plans []model.MaintenancePlan
	now := time.Now()
	deadline := now.Add(time.Duration(withinHours) * time.Hour)
	err := s.db.Where("next_due_at > ? AND next_due_at <= ?", now, deadline).Find(&plans).Error
	return plans, err
}

// --- Records ---

func (s *MaintenanceService) RecordMaintenance(planID *int64, equipmentID, operatorID int64, content string, durationMin *int) error {
	record := &model.MaintenanceRecord{
		PlanID:      planID,
		EquipmentID: equipmentID,
		OperatorID:  operatorID,
		Content:     content,
		DurationMin: durationMin,
		DoneAt:      time.Now(),
	}
	if err := s.db.Create(record).Error; err != nil {
		return err
	}
	// Update plan next_due_at
	if planID != nil {
		var plan model.MaintenancePlan
		if s.db.First(&plan, *planID).Error == nil && plan.TriggerType == "interval" && plan.IntervalHours != nil {
			now := time.Now()
			next := now.Add(time.Duration(*plan.IntervalHours) * time.Hour)
			// 写失败必须返回：保养记录已落库、但计划的"下次到期时间"没更新，
			// 计划会一直显示已逾期，现场按它做保养就会重复做。
			if err := s.db.Model(&plan).Updates(map[string]any{
				"last_done_at": now, "next_due_at": next,
			}).Error; err != nil {
				return fmt.Errorf("更新保养计划下次到期时间失败: %w", err)
			}
		}
	}
	return nil
}

func (s *MaintenanceService) ListRecords(equipmentID int64, page, pageSize int) ([]model.MaintenanceRecord, int64, error) {
	var records []model.MaintenanceRecord
	var total int64
	q := s.db.Model(&model.MaintenanceRecord{})
	if equipmentID > 0 {
		q = q.Where("equipment_id = ?", equipmentID)
	}
	q.Count(&total)
	err := q.Order("done_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error
	return records, total, err
}
