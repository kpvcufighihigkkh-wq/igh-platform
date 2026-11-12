package service

import (
	"errors"
	"fmt"

	"github.com/igh-platform/igh-platform/internal/shared/model"
	"gorm.io/gorm"
)

// 本文件实现"按丝锭编码全链追溯"。
//
// 追溯链：
//
//	bobbins.code
//	  → doffings        落筒（哪条线、哪个机位、哪一次落筒）
//	  → lots            批次（规格、工单、客户标准）
//	  → pallet_bobbins  上了哪一垛（层/位）
//	  → pallets         栈板（编码、等级、重量、打印时间）
//	  → packing_orders  包装单（打包机、目标数量）
//	  → picking_orders  拣货单（仓库、数量）
//
// 这条链是断链检测的重点：任何一环取不到都要如实报告，
// 而不是返回一个"看起来完整、实际缺数据"的结果。

// ErrTraceNotFound 表示按给定编码查不到丝锭。
var ErrTraceNotFound = errors.New("丝锭编码不存在")

// BobbinTrace 是一条丝锭的完整追溯结果。
type BobbinTrace struct {
	Bobbin model.Bobbin `json:"bobbin"`

	// Doffing 是产生这个丝锭的落筒。为空表示历史数据未关联（早期数据）。
	Doffing *model.Doffing `json:"doffing,omitempty"`
	// Lot 是所属批次。
	Lot *model.Lot `json:"lot,omitempty"`
	// Line 是产线。
	Line *model.ProductionLine `json:"line,omitempty"`

	// Pallet 是它所在的栈板。为空表示还没上垛（在制、在库或已出库）。
	Pallet *PalletPlacement `json:"pallet,omitempty"`

	// Missing 记录追溯链上缺失的环节，供前端显式提示。
	//
	// 为什么单独列出来：追溯的价值在于"能证明货是从哪来的"。
	// 如果某一环没数据却静默省略，看报告的人会误以为这一环是正常的。
	Missing []string `json:"missing,omitempty"`
}

// PalletPlacement 是丝锭在栈板上的位置及其后续单据。
type PalletPlacement struct {
	Pallet   model.Pallet `json:"pallet"`
	Layer    int          `json:"layer"`
	Position int          `json:"position"`

	PackingOrder *model.PackingOrder `json:"packing_order,omitempty"`
	PickingOrder *model.PickingOrder `json:"picking_order,omitempty"`
}

// TraceService 提供追溯查询。
type TraceService struct {
	db *gorm.DB
}

func NewTraceService(db *gorm.DB) *TraceService {
	return &TraceService{db: db}
}

// ByBobbinCode 按丝锭编码查完整链路。
func (s *TraceService) ByBobbinCode(code string) (*BobbinTrace, error) {
	if code == "" {
		return nil, fmt.Errorf("丝锭编码为空")
	}

	var bobbin model.Bobbin
	if err := s.db.Where("code = ?", code).First(&bobbin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTraceNotFound
		}
		return nil, err
	}

	t := &BobbinTrace{Bobbin: bobbin}

	// 1. 落筒
	if bobbin.DoffingID > 0 {
		var d model.Doffing
		if err := s.db.First(&d, bobbin.DoffingID).Error; err == nil {
			t.Doffing = &d
		} else {
			t.Missing = append(t.Missing, "落筒记录（doffing_id 指向的记录不存在）")
		}
	} else {
		t.Missing = append(t.Missing, "落筒记录（丝锭未关联落筒）")
	}

	// 2. 批次与产线
	lotID := bobbin.LotID
	if t.Doffing != nil && t.Doffing.LotID > 0 {
		lotID = t.Doffing.LotID
	}
	if lotID > 0 {
		var lot model.Lot
		if err := s.db.First(&lot, lotID).Error; err == nil {
			t.Lot = &lot
		} else {
			t.Missing = append(t.Missing, "批次记录")
		}
	} else {
		t.Missing = append(t.Missing, "批次记录（丝锭未关联批次）")
	}
	if t.Doffing != nil && t.Doffing.LineID > 0 {
		var line model.ProductionLine
		if err := s.db.First(&line, t.Doffing.LineID).Error; err == nil {
			t.Line = &line
		} else {
			t.Missing = append(t.Missing, "产线记录")
		}
	}

	// 3. 栈板位置
	//
	// 一个丝锭理论上只该在一垛上。查不到不报缺失：
	// "还没上垛"是正常状态（在制、在库），不是数据问题。
	var pb model.PalletBobbin
	if err := s.db.Where("bobbin_id = ?", bobbin.ID).First(&pb).Error; err == nil {
		var p model.Pallet
		if err := s.db.First(&p, pb.PalletID).Error; err == nil {
			t.Pallet = &PalletPlacement{
				Pallet: p, Layer: pb.Layer, Position: pb.Position,
			}
			s.fillOrders(t.Pallet)
		} else {
			t.Missing = append(t.Missing, "栈板记录（pallet_bobbins 指向的栈板不存在）")
		}
	}

	return t, nil
}

// fillOrders 补上栈板关联的包装单与拣货单。
func (s *TraceService) fillOrders(p *PalletPlacement) {
	if p.Pallet.PackingOrderID != nil {
		var po model.PackingOrder
		if err := s.db.First(&po, *p.Pallet.PackingOrderID).Error; err == nil {
			p.PackingOrder = &po
			if po.PickingOrderID != nil {
				var pick model.PickingOrder
				if err := s.db.First(&pick, *po.PickingOrderID).Error; err == nil {
					p.PickingOrder = &pick
				}
			}
		}
	}
	// 包装单没直接挂拣货单时，按批次+等级反查最近一张——
	// 这是弱关联，只用于展示线索，不当作确定事实。
	// 所以宁可留空，也不要猜：猜错会让追溯报告指向错误的订单。
}

// ByPalletCode 按栈板编码查这一垛上的全部丝锭。
//
// 反向查询：客户拿到的是整垛的栈板标签，而不是某一只丝锭。
func (s *TraceService) ByPalletCode(code string) (*PalletTrace, error) {
	if code == "" {
		return nil, fmt.Errorf("栈板编码为空")
	}
	var p model.Pallet
	if err := s.db.Where("code = ?", code).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTraceNotFound
		}
		return nil, err
	}

	t := &PalletTrace{Pallet: p}

	var bobbins []model.Bobbin
	err := s.db.Table("bobbins").
		Joins("JOIN pallet_bobbins pb ON pb.bobbin_id = bobbins.id").
		Where("pb.pallet_id = ?", p.ID).
		Order("bobbins.place_in_winder").
		Find(&bobbins).Error
	if err != nil {
		return nil, err
	}
	t.Bobbins = bobbins

	if p.LotID > 0 {
		var lot model.Lot
		if err := s.db.First(&lot, p.LotID).Error; err == nil {
			t.Lot = &lot
		} else {
			t.Missing = append(t.Missing, "批次记录")
		}
	}
	placement := &PalletPlacement{Pallet: p}
	s.fillOrders(placement)
	t.PackingOrder = placement.PackingOrder
	t.PickingOrder = placement.PickingOrder

	// 栈板上的锭数与关联记录数对不上，是必须暴露的数据问题：
	// 说明 pallet_bobbins 有缺口，追溯链不完整。
	if p.BobbinCount > 0 && len(bobbins) != p.BobbinCount {
		t.Missing = append(t.Missing, fmt.Sprintf(
			"丝锭关联不完整（栈板记录 %d 锭，实际关联 %d 锭）", p.BobbinCount, len(bobbins)))
	}

	return t, nil
}

// PalletTrace 是一垛的追溯结果。
type PalletTrace struct {
	Pallet  model.Pallet   `json:"pallet"`
	Lot     *model.Lot     `json:"lot,omitempty"`
	Bobbins []model.Bobbin `json:"bobbins"`

	PackingOrder *model.PackingOrder `json:"packing_order,omitempty"`
	PickingOrder *model.PickingOrder `json:"picking_order,omitempty"`

	Missing []string `json:"missing,omitempty"`
}
