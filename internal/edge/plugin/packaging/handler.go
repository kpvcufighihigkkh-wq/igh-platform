package packaging

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/shared/model"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

type handler struct {
	db *gorm.DB
}

func (h *handler) listPickingOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	var total int64
	q := h.db.Model(&model.PickingOrder{})
	if lotID := c.Query("lot_id"); lotID != "" {
		q = q.Where("lot_id = ?", lotID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)

	var orders []model.PickingOrder
	q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders)
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{"total": total, "items": orders}})
}

func (h *handler) createPickingOrder(c *gin.Context) {
	var order model.PickingOrder
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}
	order.Status = "pending"
	order.CreatedAt = time.Now()
	if err := h.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, types.Response{Code: 0, Data: order})
}

func (h *handler) startPickingOrder(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	now := time.Now()
	result := h.db.Model(&model.PickingOrder{}).Where("id = ? AND status = ?", id, "pending").Updates(map[string]any{
		"status":     "in_progress",
		"started_at": now,
	})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: "order not in pending status"})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0})
}

func (h *handler) completePickingOrder(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	now := time.Now()
	result := h.db.Model(&model.PickingOrder{}).Where("id = ? AND status = ?", id, "in_progress").Updates(map[string]any{
		"status":       "completed",
		"completed_at": now,
	})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: "order not in progress"})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0})
}

func (h *handler) listPackingOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	var total int64
	q := h.db.Model(&model.PackingOrder{})
	if lotID := c.Query("lot_id"); lotID != "" {
		q = q.Where("lot_id = ?", lotID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)

	var orders []model.PackingOrder
	q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders)
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{"total": total, "items": orders}})
}

func (h *handler) createPackingOrder(c *gin.Context) {
	var order model.PackingOrder
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}
	order.Status = "pending"
	order.PackedQuantity = 0
	order.CreatedAt = time.Now()
	if err := h.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, types.Response{Code: 0, Data: order})
}

func (h *handler) startPackingOrder(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	now := time.Now()
	result := h.db.Model(&model.PackingOrder{}).Where("id = ? AND status = ?", id, "pending").Updates(map[string]any{
		"status":     "in_progress",
		"started_at": now,
	})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: "order not in pending status"})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0})
}

func (h *handler) listPallets(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	var total int64
	q := h.db.Model(&model.Pallet{})
	if lotID := c.Query("lot_id"); lotID != "" {
		q = q.Where("lot_id = ?", lotID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)

	var pallets []model.Pallet
	q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&pallets)
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{"total": total, "items": pallets}})
}

type completePackingReq struct {
	PalletCode     string   `json:"pallet_code" binding:"required"`
	LotID          int64    `json:"lot_id" binding:"required"`
	GradeCode      string   `json:"grade_code" binding:"required"`
	BobbinCount    int      `json:"bobbin_count" binding:"required,min=1"`
	Layers         int      `json:"layers" binding:"required,min=1"`
	TotalWeight    float64  `json:"total_weight"`
	PackingOrderID *int64   `json:"packing_order_id"`
}

// completePacking 在边端落一垛。
//
// 与中心 OrderService.ProcessPackingComplete 保持同一套语义，
// 否则断网期间边端记的进度和联网后中心记的对不上：
//   - 幂等：同一个 code 重复提交直接返回已有记录
//   - 进度按**锭数**累加（旧实现按垛 +1，与中心不一致）
//   - 判完成用累加后回读的值，不能拿累加前的快照再加一次
func (h *handler) completePacking(c *gin.Context) {
	var req completePackingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}

	var pallet model.Pallet
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var existing model.Pallet
		if err := tx.Where("code = ?", req.PalletCode).First(&existing).Error; err == nil {
			pallet = existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		weight := &req.TotalWeight
		if req.TotalWeight == 0 {
			weight = nil
		}
		pallet = model.Pallet{
			Code:           req.PalletCode,
			LotID:          req.LotID,
			GradeCode:      req.GradeCode,
			Layers:         req.Layers,
			BobbinCount:    req.BobbinCount,
			TotalWeight:    weight,
			PackingOrderID: req.PackingOrderID,
			Status:         "created",
			CreatedAt:      time.Now(),
		}
		if err := tx.Create(&pallet).Error; err != nil {
			return err
		}

		if req.PackingOrderID != nil {
			if err := tx.Model(&model.PackingOrder{}).Where("id = ?", *req.PackingOrderID).
				Update("packed_quantity",
					gorm.Expr("packed_quantity + ?", req.BobbinCount)).Error; err != nil {
				return err
			}

			var order model.PackingOrder
			if err := tx.First(&order, *req.PackingOrderID).Error; err != nil {
				return err
			}
			if order.Status != "completed" && order.PackedQuantity >= order.TargetQuantity {
				now := time.Now()
				return tx.Model(&model.PackingOrder{}).Where("id = ?", order.ID).Updates(map[string]any{
					"status":       "completed",
					"completed_at": now,
				}).Error
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, types.Response{Code: 0, Data: pallet})
}

// intakeReq 是包装工位扫码上料的请求。
//
// 现场动作：操作工把丝饼取下、逐个扫码，扫完一盘后打包机开始堆垛。
// 上料的唯一作用是给这些丝饼打上"在包装工位"的标记——
// 打包机出垛时靠这个标记回查本垛是哪几锭（PalletizerController.bobbinsInPacking）。
type intakeReq struct {
	BobbinIDs []int64 `json:"bobbin_ids" binding:"required,min=1"`
	// PalletizerID 只用于写进 position_detail，便于事后核对是哪台机进的料
	PalletizerID int `json:"palletizer_id"`
	// Replace 为 true 时先清空本工位已有的上料标记再写入。
	// 换批次/换垛时用：否则上一盘的残留会被算进下一垛。
	Replace bool `json:"replace"`
}

// intakeBobbins 把丝饼标记为"在包装工位"。
//
// 为什么必须有这一步：打包机出垛时按 current_position='packing' 回查锭列表，
// 没有这一步就永远查不到，栈板只能记一个锭数、不知道是哪几锭，追溯断链。
//
// 校验逐条给出而不是遇错即停：现场扫码是连续动作，
// 一次把"哪些锭有问题"全列出来才能一次补齐。
func (h *handler) intakeBobbins(c *gin.Context) {
	var req intakeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}

	var accepted []int64
	var problems []string
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if req.Replace {
			if err := tx.Model(&model.Bobbin{}).
				Where("current_position = ?", "packing").
				Update("current_position", "warehouse").Error; err != nil {
				return err
			}
		}

		for _, id := range req.BobbinIDs {
			if id <= 0 {
				problems = append(problems, fmt.Sprintf("丝饼 id %d 非法", id))
				continue
			}
			var b model.Bobbin
			if err := tx.First(&b, id).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					problems = append(problems, fmt.Sprintf("丝饼 %d 不存在", id))
					continue
				}
				return err
			}
			// 已经上了垛的不该再进料——那会导致同一锭出现在两块栈板上
			if b.CurrentPosition == "pallet" {
				problems = append(problems, fmt.Sprintf("丝饼 %d 已上垛", id))
				continue
			}

			detail := map[string]any{}
			if req.PalletizerID > 0 {
				detail["palletizer"] = req.PalletizerID
			}
			raw, _ := json.Marshal(detail)
			if err := tx.Model(&model.Bobbin{}).Where("id = ?", id).Updates(map[string]any{
				"current_position": "packing",
				"position_detail":  string(raw),
				"updated_at":       time.Now(),
			}).Error; err != nil {
				return err
			}
			accepted = append(accepted, id)
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{
		"accepted": accepted,
		"count":    len(accepted),
		"problems": problems,
	}})
}

// listPackingBobbins 列出当前在包装工位待装的丝饼。
//
// 现场用来核对"扫进来的锭是不是都在"，也是出垛时回查的同一批数据。
func (h *handler) listPackingBobbins(c *gin.Context) {
	var bobbins []model.Bobbin
	q := h.db.Where("current_position = ?", "packing")
	if lotID := c.Query("lot_id"); lotID != "" {
		q = q.Where("lot_id = ?", lotID)
	}
	q.Order("id").Find(&bobbins)
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{
		"items": bobbins,
		"total": len(bobbins),
	}})
}

func (h *handler) listLots(c *gin.Context) {
	status := c.DefaultQuery("status", "active")
	var lots []model.Lot
	q := h.db.Model(&model.Lot{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Order("id DESC").Limit(100).Find(&lots)
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{
		"total": len(lots), "items": lots,
	}})
}

// Suppress unused import warning
var _ = fmt.Sprintf
