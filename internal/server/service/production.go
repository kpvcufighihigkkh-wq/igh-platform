package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/shared/bobbincode"
	"github.com/igh-platform/igh-platform/internal/shared/geometry"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProductionService struct {
	db *gorm.DB
	// geom 提供现场可配的几何表（装载位置查表）。
	// 允许为 nil：未接入几何表时退化为"只记录台车，不算容器位置"。
	geom *SiteGeometry
}

func NewProductionService(db *gorm.DB) *ProductionService {
	return &ProductionService{db: db}
}

// SetGeometry 注入几何表提供方。
func (s *ProductionService) SetGeometry(g *SiteGeometry) {
	s.geom = g
}

// --- Lots ---

type CreateLotRequest struct {
	Lot         model.Lot              `json:"lot"`
	Weights     []model.LotWeight      `json:"weights"`
	GradeRanges []model.LotGradeRange  `json:"grade_ranges"`
}

func (s *ProductionService) ListLots(status string, page, pageSize int) ([]model.Lot, int64, error) {
	var lots []model.Lot
	var total int64
	q := s.db.Model(&model.Lot{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&lots).Error
	return lots, total, err
}

func (s *ProductionService) GetLot(id int64) (*model.Lot, error) {
	var lot model.Lot
	err := s.db.Preload("Weights").Preload("GradeRanges").First(&lot, id).Error
	return &lot, err
}

func (s *ProductionService) CreateLot(req *CreateLotRequest) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&req.Lot).Error; err != nil {
			return err
		}
		for i := range req.Weights {
			req.Weights[i].LotID = req.Lot.ID
		}
		for i := range req.GradeRanges {
			req.GradeRanges[i].LotID = req.Lot.ID
		}
		if len(req.Weights) > 0 {
			if err := tx.Create(&req.Weights).Error; err != nil {
				return err
			}
		}
		if len(req.GradeRanges) > 0 {
			if err := tx.Create(&req.GradeRanges).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *ProductionService) UpdateLot(req *CreateLotRequest) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&req.Lot).Error; err != nil {
			return err
		}
		// 删除旧明细的错误必须检查：在事务里被吞掉，
		// 结果是旧重量区间还在、新的一批又插进来，同一批次出现两套互斥的区间。
		if err := tx.Where("lot_id = ?", req.Lot.ID).Delete(&model.LotWeight{}).Error; err != nil {
			return fmt.Errorf("清除旧重量区间失败: %w", err)
		}
		if err := tx.Where("lot_id = ?", req.Lot.ID).Delete(&model.LotGradeRange{}).Error; err != nil {
			return fmt.Errorf("清除旧等级区间失败: %w", err)
		}
		for i := range req.Weights {
			req.Weights[i].ID = 0
			req.Weights[i].LotID = req.Lot.ID
		}
		for i := range req.GradeRanges {
			req.GradeRanges[i].ID = 0
			req.GradeRanges[i].LotID = req.Lot.ID
		}
		if len(req.Weights) > 0 {
			if err := tx.Create(&req.Weights).Error; err != nil {
				return err
			}
		}
		if len(req.GradeRanges) > 0 {
			if err := tx.Create(&req.GradeRanges).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *ProductionService) EndLot(id int64) error {
	return s.db.Model(&model.Lot{}).Where("id = ?", id).Updates(map[string]any{
		"end_lot": true, "status": "completed", "updated_at": time.Now(),
	}).Error
}

func (s *ProductionService) TogglePackingLock(id int64) error {
	return s.db.Exec("UPDATE lots SET packing_lock = NOT packing_lock, updated_at = ? WHERE id = ?", time.Now(), id).Error
}

// --- Doffings ---

func (s *ProductionService) ListDoffings(lotID int64, page, pageSize int) ([]model.Doffing, int64, error) {
	var doffings []model.Doffing
	var total int64
	q := s.db.Model(&model.Doffing{})
	if lotID > 0 {
		q = q.Where("lot_id = ?", lotID)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&doffings).Error
	return doffings, total, err
}

// ProcessDoffing 写入一次落筒及其全部锭。
//
// 幂等：唯一键 (line_id, winder_name, doff_no, lot_id) 对应旧库的
// doffings_un，边端重传同一条事件时直接返回已有记录，不重复插锭。
func (s *ProductionService) ProcessDoffing(lineID, lotID int64, winderName string, doffNo, bobbinCount int, endTime time.Time) (*model.Doffing, error) {
	doffing := &model.Doffing{
		LineID:      lineID,
		WinderName:  winderName,
		DoffNo:      doffNo,
		LotID:       lotID,
		EndTime:     &endTime,
		BobbinCount: bobbinCount,
	}

	// 丝锭编码要用批号而不是 lot_id：编码是给人和扫码枪看的，
	// 印在标签上的必须是现场认识的批号。
	var lotCode string
	if err := s.db.Model(&model.Lot{}).Where("id = ?", lotID).
		Pluck("code", &lotCode).Error; err != nil {
		return nil, fmt.Errorf("查询批号失败（lot_id=%d）: %w", lotID, err)
	}
	// 产线编码同理，而且是唯一性的一部分：批号不隐含产线，
	// 同一批次可以并行在多条线上生产，两边机位号都从 1 开始。
	var lineCode string
	if err := s.db.Model(&model.ProductionLine{}).Where("id = ?", lineID).
		Pluck("code", &lineCode).Error; err != nil {
		return nil, fmt.Errorf("查询产线编码失败（line_id=%d）: %w", lineID, err)
	}
	ident, err := bobbincode.ParseWinderName(winderName)
	if err != nil {
		return nil, err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(doffing)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// 已存在：读回原记录，并跳过锭的创建。
			//
			// 条件必须与 uk_doffing 完全一致（含 end_time）：
			// 少一个字段就可能读到**另一次**落筒——落筒号会循环，
			// 同一 (产线,机位,落筒号,批次) 在不同时刻是不同落筒。
			var existing model.Doffing
			if err := tx.Where("line_id = ? AND winder_name = ? AND doff_no = ? AND lot_id = ? AND end_time = ?",
				lineID, winderName, doffNo, lotID, endTime).First(&existing).Error; err != nil {
				return err
			}
			*doffing = existing
			return nil
		}
		// Create bobbins for this doffing
		for i := 1; i <= bobbinCount; i++ {
			// 编码里带落筒时刻：落筒号会循环，只靠它无法区分
			// 循环前后的两次落筒。详见 bobbincode.Build 的说明。
			code, err := bobbincode.Build(lotCode, lineCode, ident.Winder, ident.Side, doffNo, i, endTime)
			if err != nil {
				return fmt.Errorf("生成丝锭编码失败（落筒 %s #%d 锭位 %d）: %w",
					winderName, doffNo, i, err)
			}
			bobbin := &model.Bobbin{
				Code:            code,
				DoffingID:       doffing.ID,
				PlaceInWinder:   i,
				LotID:           lotID,
				Status:          "in_process",
				CurrentPosition: "winder",
			}
			if err := tx.Create(bobbin).Error; err != nil {
				return err
			}
		}
		return nil
	})

	return doffing, err
}

// --- Bobbins ---

func (s *ProductionService) ListBobbins(lotID int64, status string, page, pageSize int) ([]model.Bobbin, int64, error) {
	var bobbins []model.Bobbin
	var total int64
	q := s.db.Model(&model.Bobbin{})
	if lotID > 0 {
		q = q.Where("lot_id = ?", lotID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&bobbins).Error
	return bobbins, total, err
}

func (s *ProductionService) GetBobbin(id int64) (*model.Bobbin, error) {
	var bobbin model.Bobbin
	err := s.db.First(&bobbin, id).Error
	return &bobbin, err
}

// GetBobbinByCode 按业务编码查丝锭。
//
// 编码是现场唯一可读的标识（印在丝锭标签上），客户投诉时拿到的就是它。
// 注意用 First 而不是兜底的 Find：查不到必须返回 ErrRecordNotFound，
// 让调用方回 404，而不是返回一个零值丝锭让人误以为查到了。
func (s *ProductionService) GetBobbinByCode(code string) (*model.Bobbin, error) {
	var bobbin model.Bobbin
	err := s.db.Where("code = ?", code).First(&bobbin).Error
	return &bobbin, err
}

func (s *ProductionService) UpdateBobbinPosition(id int64, position string, detail map[string]any) error {
	detailJSON, _ := json.Marshal(detail)
	return s.db.Model(&model.Bobbin{}).Where("id = ?", id).Updates(map[string]any{
		"current_position": position,
		"position_detail":  detailJSON,
		"updated_at":       time.Now(),
	}).Error
}

// --- Trolleys ---

func (s *ProductionService) ListTrolleys(lotID int64, status string, page, pageSize int) ([]model.Trolley, int64, error) {
	var trolleys []model.Trolley
	var total int64
	q := s.db.Model(&model.Trolley{})
	if lotID > 0 {
		q = q.Where("lot_id = ?", lotID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&trolleys).Error
	return trolleys, total, err
}

// LoadTrolley 装载一车：记录台车，并把锭的位置更新为"在台车上的哪个位置"。
//
// edgeCode 用于取该边端当前生效的装载位置查表；lineID 用于取线路的装载模式。
// 位置计算规则与旧系统一致（load_bobbins + calc_place）：
//
//	calc_place(锭在卷绕机上的位置, 'A'|'B', 装载模式)
//
// 其中 A/B 由同一车两个落筒的顺序决定：先落筒 → A，后落筒 → B。
//
// 位置写进 position_detail（JSON），不新增表字段：
//
//	{"trolley_id": 3, "place": 15}
//
// 查表失败时仍然完成装载，只是不写 place，并把问题记进日志——
// 一车没装上比位置算错要好处理得多。
func (s *ProductionService) LoadTrolley(edgeCode string, number int, rfid string, lotID, lineID int64, doffingIDs []int64) (*model.Trolley, error) {
	idsJSON, _ := json.Marshal(doffingIDs)
	trolley := &model.Trolley{
		Number:      number,
		RFID:        rfid,
		LotID:       &lotID,
		DoffingIDs:  string(idsJSON),
		LoadingTime: time.Now(),
		LineID:      &lineID,
		Status:      "loaded",
	}

	// 装载位置查表 + 线路装载模式
	var seq *geometry.LoadingSequences
	pattern := s.loadingPattern(lineID)
	switch {
	case s.geom == nil:
		logger.L.Warnw("load trolley: geometry not wired, skip place", "line_id", lineID)
	case edgeCode == "":
		logger.L.Warnw("load trolley: empty edge code, skip place", "line_id", lineID)
	case pattern == "":
		logger.L.Warnw("load trolley: line has no loading_pattern, skip place",
			"line_id", lineID, "edge", edgeCode)
	default:
		var err error
		seq, err = s.geom.LoadingSequences(edgeCode)
		if err != nil {
			logger.L.Warnw("load trolley: load geometry failed", "edge", edgeCode, "error", err)
		}
		if seq == nil {
			logger.L.Warnw("load trolley: no loading table available, skip place",
				"line_id", lineID, "edge", edgeCode, "pattern", pattern)
		}
	}

	// 同一车上每个落筒的侧别：第 1 个落筒为 A，第 2 个为 B
	sideOf := func(i int) (geometry.SpindleSide, bool) {
		switch i {
		case 0:
			return geometry.SpindleA, true
		case 1:
			return geometry.SpindleB, true
		default:
			return 0, false
		}
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(trolley).Error; err != nil {
			return err
		}

		for i, doffID := range doffingIDs {
			side, ok := sideOf(i)

			var bobbins []model.Bobbin
			if err := tx.Where("doffing_id = ? AND current_position = ?", doffID, "winder").
				Find(&bobbins).Error; err != nil {
				return err
			}

			for _, b := range bobbins {
				detail := map[string]any{"trolley_id": trolley.ID}

				if seq != nil && ok {
					place, err := seq.Place(pattern, b.PlaceInWinder, side)
					if err != nil {
						logger.L.Warnw("load trolley: place lookup failed",
							"edge", edgeCode, "pattern", pattern, "doffing", doffID,
							"place_in_winder", b.PlaceInWinder, "side", string(side), "error", err)
					} else {
						detail["place"] = place
					}
				}

				raw, _ := json.Marshal(detail)
				if err := tx.Model(&model.Bobbin{}).Where("id = ?", b.ID).Updates(map[string]any{
					"current_position": "trolley",
					"position_detail":  string(raw),
					"updated_at":       time.Now(),
				}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})

	return trolley, err
}

// loadingPattern 从线路主数据里取装载模式（旧库 spinning_sides.loading_side）。
//
// 存在 production_lines.config 里，例如 {"loading_pattern":"fdy"}。
// 取不到时返回空串，调用方因此不做位置计算——不猜模式。
func (s *ProductionService) loadingPattern(lineID int64) string {
	var line model.ProductionLine
	if err := s.db.First(&line, lineID).Error; err != nil || len(line.Config) == 0 {
		return ""
	}
	var cfg struct {
		LoadingPattern string `json:"loading_pattern"`
	}
	if err := json.Unmarshal(line.Config, &cfg); err != nil {
		return ""
	}
	return cfg.LoadingPattern
}

// --- Winder Status ---

type WinderStatus struct {
	Code             string     `json:"code"`
	Name             string     `json:"name"`
	Status           string     `json:"status"`
	ActiveLotID      *int64     `json:"active_lot_id,omitempty"`
	ActiveLotCode    string     `json:"active_lot_code,omitempty"`
	ActiveLotSpec    string     `json:"active_lot_spec,omitempty"`
	TodayDoffCount   int        `json:"today_doff_count"`
	TodayBobbinCount int        `json:"today_bobbin_count"`
	LastDoffingTime  *time.Time `json:"last_doffing_time,omitempty"`
}

func (s *ProductionService) WinderStatusByLine(lineCode string) ([]WinderStatus, error) {
	var equipment []model.Equipment
	if err := s.db.Where("type = ? AND line_code = ?", "winder", lineCode).
		Order("code").Find(&equipment).Error; err != nil {
		return nil, err
	}
	if len(equipment) == 0 {
		return []WinderStatus{}, nil
	}

	codes := make([]string, len(equipment))
	for i, e := range equipment {
		codes[i] = e.Code
	}

	type todayStat struct {
		WinderName  string
		DoffCount   int
		BobbinCount int
	}
	var stats []todayStat
	today := time.Now().Format("2006-01-02")
	s.db.Model(&model.Doffing{}).
		Select("winder_name, COUNT(*) as doff_count, COALESCE(SUM(bobbin_count),0) as bobbin_count").
		Where("winder_name IN ? AND DATE(created_at) = ?", codes, today).
		Group("winder_name").Find(&stats)
	statsMap := map[string]todayStat{}
	for _, st := range stats {
		statsMap[st.WinderName] = st
	}

	type lastDoff struct {
		WinderName string
		CreatedAt  time.Time
		LotID      int64
	}
	var lastDoffs []lastDoff
	s.db.Raw(`SELECT DISTINCT ON (winder_name) winder_name, created_at, lot_id
		FROM doffings WHERE winder_name IN (?)
		ORDER BY winder_name, created_at DESC`, codes).Scan(&lastDoffs)
	lastMap := map[string]lastDoff{}
	for _, ld := range lastDoffs {
		lastMap[ld.WinderName] = ld
	}

	lotIDs := map[int64]bool{}
	for _, ld := range lastDoffs {
		lotIDs[ld.LotID] = true
	}
	lotMap := map[int64]model.Lot{}
	if len(lotIDs) > 0 {
		ids := make([]int64, 0, len(lotIDs))
		for id := range lotIDs {
			ids = append(ids, id)
		}
		var lots []model.Lot
		s.db.Where("id IN ?", ids).Find(&lots)
		for _, l := range lots {
			lotMap[l.ID] = l
		}
	}

	results := make([]WinderStatus, len(equipment))
	for i, e := range equipment {
		ws := WinderStatus{Code: e.Code, Name: e.Name, Status: e.Status}
		if st, ok := statsMap[e.Code]; ok {
			ws.TodayDoffCount = st.DoffCount
			ws.TodayBobbinCount = st.BobbinCount
		}
		if ld, ok := lastMap[e.Code]; ok {
			ws.LastDoffingTime = &ld.CreatedAt
			ws.ActiveLotID = &ld.LotID
			if lot, ok2 := lotMap[ld.LotID]; ok2 {
				ws.ActiveLotCode = lot.Code
				ws.ActiveLotSpec = lot.Specification
			}
		}
		results[i] = ws
	}
	return results, nil
}

func (s *ProductionService) RecentDoffingsByWinder(winderName string, limit int) ([]model.Doffing, error) {
	var doffings []model.Doffing
	err := s.db.Where("winder_name = ?", winderName).
		Order("created_at DESC").Limit(limit).Find(&doffings).Error
	return doffings, err
}
