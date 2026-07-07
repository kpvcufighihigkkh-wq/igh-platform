package service

import (
	"fmt"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"gorm.io/gorm"
)

type OrderService struct {
	db *gorm.DB
}

func NewOrderService(db *gorm.DB) *OrderService {
	return &OrderService{db: db}
}

// --- Picking Orders ---

func (s *OrderService) ListPickingOrders(lotID int64, status string, page, pageSize int) ([]model.PickingOrder, int64, error) {
	var orders []model.PickingOrder
	var total int64
	q := s.db.Model(&model.PickingOrder{})
	if lotID > 0 {
		q = q.Where("lot_id = ?", lotID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders).Error
	return orders, total, err
}

func (s *OrderService) CreatePickingOrder(order *model.PickingOrder) error {
	return s.db.Create(order).Error
}

func (s *OrderService) StartPickingOrder(id int64) error {
	now := time.Now()
	return s.db.Model(&model.PickingOrder{}).Where("id = ? AND status = ?", id, "pending").
		Updates(map[string]any{"status": "in_progress", "started_at": now}).Error
}

func (s *OrderService) CompletePickingOrder(id int64) error {
	now := time.Now()
	return s.db.Model(&model.PickingOrder{}).Where("id = ? AND status = ?", id, "in_progress").
		Updates(map[string]any{"status": "completed", "completed_at": now}).Error
}

// --- Packing Orders ---

func (s *OrderService) ListPackingOrders(lotID int64, status string, page, pageSize int) ([]model.PackingOrder, int64, error) {
	var orders []model.PackingOrder
	var total int64
	q := s.db.Model(&model.PackingOrder{})
	if lotID > 0 {
		q = q.Where("lot_id = ?", lotID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders).Error
	return orders, total, err
}

func (s *OrderService) CreatePackingOrder(order *model.PackingOrder) error {
	return s.db.Create(order).Error
}

func (s *OrderService) StartPackingOrder(id int64) error {
	now := time.Now()
	return s.db.Model(&model.PackingOrder{}).Where("id = ? AND status = ?", id, "pending").
		Updates(map[string]any{"status": "in_progress", "started_at": now}).Error
}

// --- Pallets ---

func (s *OrderService) ListPallets(lotID int64, status string, page, pageSize int) ([]model.Pallet, int64, error) {
	var pallets []model.Pallet
	var total int64
	q := s.db.Model(&model.Pallet{})
	if lotID > 0 {
		q = q.Where("lot_id = ?", lotID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&pallets).Error
	return pallets, total, err
}

func (s *OrderService) ProcessPackingComplete(palletCode string, lotID int64, gradeCode string, bobbinCount, layers int, totalWeight float64, packingOrderID *int64) (*model.Pallet, error) {
	pallet := &model.Pallet{
		Code:           palletCode,
		LotID:          lotID,
		GradeCode:      gradeCode,
		Layers:         layers,
		BobbinCount:    bobbinCount,
		TotalWeight:    &totalWeight,
		PackingOrderID: packingOrderID,
		Status:         "created",
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(pallet).Error; err != nil {
			return err
		}
		// Update packing order progress
		if packingOrderID != nil {
			if err := tx.Model(&model.PackingOrder{}).Where("id = ?", *packingOrderID).
				Updates(map[string]any{
					"packed_quantity": gorm.Expr("packed_quantity + ?", bobbinCount),
				}).Error; err != nil {
				return err
			}
			// Check if packing order is complete
			var order model.PackingOrder
			tx.First(&order, *packingOrderID)
			if order.PackedQuantity+bobbinCount >= order.TargetQuantity {
				now := time.Now()
				tx.Model(&order).Updates(map[string]any{"status": "completed", "completed_at": now})
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("process packing: %w", err)
	}
	return pallet, nil
}
