package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// PackingCompleteRequest 是一次"堆垛完成"的完整输入。
//
// 用结构体而不是长参数表：字段还要继续加（打包机号、生产日期、锭列表），
// 位置参数每加一个都要改所有调用点，且同类参数（int, int）极易传错顺序。
type PackingCompleteRequest struct {
	// PalletCode 是栈板编码，同时用作幂等键（表上有唯一索引）
	PalletCode string
	LotID      int64
	GradeCode  string
	// BobbinCount 是这垛的锭数，用于进度累加
	BobbinCount int
	Layers      int
	TotalWeight float64
	// PackingOrderID 为空表示这垛不归属任何包装单（手工堆垛/补录）
	PackingOrderID *int64
	// BobbinIDs 是实际上了这垛的丝饼 id。为空时只记录锭数、不建关联，
	// 不能凭空猜：猜错会导致追溯查到别的锭。
	BobbinIDs []int64
	// PalletizerID / ProductDate / DailyID 用于补齐栈板主数据，可为空
	PalletizerID *int64
	ProductDate  *time.Time
	DailyID      *int
}

// ProcessPackingComplete 落一垛。
//
// 幂等语义：以 PalletCode 为键，边端至少一次投递重传同一垛时直接返回已有记录，
// 不重复创建、也不重复累加包装单进度。
//
// 旧实现有三个缺陷（已修）：
//  1. packed_quantity 先 +bobbinCount，随后又用 order.PackedQuantity+bobbinCount
//     判完成 —— 第二次读取到的已是累加后的值，等于加了两次，包装单会提前完成
//  2. 没有幂等：重传撞 code 唯一索引，整条事务回滚，事件被当成失败处理
//  3. 不写 pallet_bobbins：只留一个数字，事后无法追溯是哪几锭
func (s *OrderService) ProcessPackingComplete(req PackingCompleteRequest) (*model.Pallet, error) {
	if req.PalletCode == "" {
		return nil, fmt.Errorf("栈板编码为空")
	}

	var pallet *model.Pallet
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 幂等：已存在就直接返回，不重复做任何副作用
		var existing model.Pallet
		if err := tx.Where("code = ?", req.PalletCode).First(&existing).Error; err == nil {
			pallet = &existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		weight := req.TotalWeight
		pallet = &model.Pallet{
			Code:           req.PalletCode,
			LotID:          req.LotID,
			GradeCode:      req.GradeCode,
			Layers:         req.Layers,
			BobbinCount:    req.BobbinCount,
			PackingOrderID: req.PackingOrderID,
			PalletizerID:   req.PalletizerID,
			ProductDate:    req.ProductDate,
			DailyID:        req.DailyID,
			Status:         "created",
		}
		// 0 权重视为"未上报"，存 NULL 而不是 0.000，
		// 否则报表里会把它当成真实重量参与平均。
		if req.TotalWeight != 0 {
			pallet.TotalWeight = &weight
		}
		if err := tx.Create(pallet).Error; err != nil {
			return err
		}

		// 建立栈板-丝饼关联，并把丝饼置为"已包装"。
		// 用 OnConflict DoNothing：同一锭重复上报不该让整垛失败。
		if len(req.BobbinIDs) > 0 {
			now := time.Now()
			links := make([]model.PalletBobbin, 0, len(req.BobbinIDs))
			for _, id := range req.BobbinIDs {
				if id <= 0 {
					continue
				}
				links = append(links, model.PalletBobbin{
					PalletID:  pallet.ID,
					BobbinID:  id,
					CreatedAt: now,
				})
			}
			if len(links) > 0 {
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
					Create(&links).Error; err != nil {
					return err
				}
				if err := tx.Model(&model.Bobbin{}).
					Where("id IN ?", req.BobbinIDs).
					Updates(map[string]any{
						"status":           "packed",
						"current_position": "pallet",
						"updated_at":       now,
					}).Error; err != nil {
					return err
				}
			}
		}

		if req.PackingOrderID != nil {
			return completeProgress(tx, *req.PackingOrderID, req.BobbinCount)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("process packing: %w", err)
	}
	return pallet, nil
}

// completeProgress 累加包装单进度并在达到目标时结束它。
//
// 判完成必须用**累加后回读**的值，不能拿累加前的快照再加一次。
func completeProgress(tx *gorm.DB, orderID int64, delta int) error {
	if err := tx.Model(&model.PackingOrder{}).Where("id = ?", orderID).
		Update("packed_quantity", gorm.Expr("packed_quantity + ?", delta)).Error; err != nil {
		return err
	}

	var order model.PackingOrder
	if err := tx.First(&order, orderID).Error; err != nil {
		return err
	}
	if order.Status == "completed" {
		return nil
	}
	if order.PackedQuantity >= order.TargetQuantity {
		now := time.Now()
		return tx.Model(&model.PackingOrder{}).Where("id = ?", orderID).
			Updates(map[string]any{"status": "completed", "completed_at": now}).Error
	}
	return nil
}
