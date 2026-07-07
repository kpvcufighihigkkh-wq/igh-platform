package service

import (
	"fmt"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"gorm.io/gorm"
)

type WarehouseService struct {
	db *gorm.DB
}

func NewWarehouseService(db *gorm.DB) *WarehouseService {
	return &WarehouseService{db: db}
}

// --- Warehouses ---

func (s *WarehouseService) ListWarehouses() ([]model.Warehouse, error) {
	var whs []model.Warehouse
	err := s.db.Find(&whs).Error
	return whs, err
}

func (s *WarehouseService) GetWarehouse(id int64) (*model.Warehouse, error) {
	var wh model.Warehouse
	err := s.db.First(&wh, id).Error
	return &wh, err
}

func (s *WarehouseService) CreateWarehouse(wh *model.Warehouse) error {
	if err := s.db.Create(wh).Error; err != nil {
		return err
	}
	return s.initSlots(wh)
}

func (s *WarehouseService) initSlots(wh *model.Warehouse) error {
	var slots []model.WarehouseSlot
	for r := 1; r <= wh.Rows; r++ {
		for c := 1; c <= wh.Cols; c++ {
			for l := 1; l <= wh.Layers; l++ {
				slots = append(slots, model.WarehouseSlot{
					WarehouseID: wh.ID,
					RowNo:       r,
					ColNo:       c,
					LayerNo:     l,
					Status:      "empty",
					UpdatedAt:   time.Now(),
				})
			}
		}
	}
	return s.db.CreateInBatches(slots, 500).Error
}

// --- Slots ---

func (s *WarehouseService) GetSlots(warehouseID int64, status string) ([]model.WarehouseSlot, error) {
	var slots []model.WarehouseSlot
	q := s.db.Where("warehouse_id = ?", warehouseID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Order("row_no, col_no, layer_no").Find(&slots).Error
	return slots, err
}

func (s *WarehouseService) GetSlotStats(warehouseID int64) (map[string]int, error) {
	type result struct {
		Status string
		Count  int
	}
	var results []result
	err := s.db.Model(&model.WarehouseSlot{}).
		Select("status, count(*) as count").
		Where("warehouse_id = ?", warehouseID).
		Group("status").Find(&results).Error
	if err != nil {
		return nil, err
	}
	stats := make(map[string]int)
	for _, r := range results {
		stats[r.Status] = r.Count
	}
	return stats, nil
}

// --- Inbound ---

func (s *WarehouseService) Inbound(warehouseCode string, row, col, layer int, occupantType string, occupantID, lotID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var slot model.WarehouseSlot
		err := tx.Joins("JOIN warehouses ON warehouses.id = warehouse_slots.warehouse_id").
			Where("warehouses.code = ? AND warehouse_slots.row_no = ? AND warehouse_slots.col_no = ? AND warehouse_slots.layer_no = ?",
				warehouseCode, row, col, layer).
			First(&slot).Error
		if err != nil {
			return fmt.Errorf("slot not found: %w", err)
		}
		if slot.Status != "empty" && slot.Status != "reserved" {
			return fmt.Errorf("slot %d-%d-%d is %s, cannot inbound", row, col, layer, slot.Status)
		}
		return tx.Model(&slot).Updates(map[string]any{
			"status":        "occupied",
			"occupant_type": occupantType,
			"occupant_id":   occupantID,
			"lot_id":        lotID,
			"updated_at":    time.Now(),
		}).Error
	})
}

// --- Outbound ---

func (s *WarehouseService) Outbound(warehouseCode string, row, col, layer int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var slot model.WarehouseSlot
		err := tx.Joins("JOIN warehouses ON warehouses.id = warehouse_slots.warehouse_id").
			Where("warehouses.code = ? AND warehouse_slots.row_no = ? AND warehouse_slots.col_no = ? AND warehouse_slots.layer_no = ?",
				warehouseCode, row, col, layer).
			First(&slot).Error
		if err != nil {
			return fmt.Errorf("slot not found: %w", err)
		}
		if slot.Status != "occupied" {
			return fmt.Errorf("slot %d-%d-%d is %s, cannot outbound", row, col, layer, slot.Status)
		}
		return tx.Model(&slot).Updates(map[string]any{
			"status":        "empty",
			"occupant_type": "",
			"occupant_id":   nil,
			"lot_id":        nil,
			"updated_at":    time.Now(),
		}).Error
	})
}

// --- Allocation (find empty slot) ---

func (s *WarehouseService) AllocateSlot(warehouseID int64) (*model.WarehouseSlot, error) {
	var slot model.WarehouseSlot
	err := s.db.Where("warehouse_id = ? AND status = ?", warehouseID, "empty").
		Order("layer_no, row_no, col_no").
		First(&slot).Error
	if err != nil {
		return nil, fmt.Errorf("no empty slot available")
	}
	// Reserve it
	s.db.Model(&slot).Updates(map[string]any{"status": "reserved", "updated_at": time.Now()})
	return &slot, nil
}
