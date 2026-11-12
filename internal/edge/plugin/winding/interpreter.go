package winding

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/model"
	"gorm.io/gorm"
)

type interpreter struct {
	db          *gorm.DB
	mu          sync.Mutex
	lastDoffCnt map[string]uint32
	lastAlarm   map[string]byte
	lineID      int64
	lotID       int64
	resolved    bool
}

func newInterpreter(db *gorm.DB) *interpreter {
	return &interpreter{
		db:          db,
		lastDoffCnt: make(map[string]uint32),
		lastAlarm:   make(map[string]byte),
	}
}

func (it *interpreter) resolve() {
	if it.resolved {
		return
	}
	var line model.ProductionLine
	if it.db.Where("status = ?", "running").First(&line).Error == nil {
		it.lineID = line.ID
	}
	var lot model.Lot
	if it.db.Where("status = ?", "active").First(&lot).Error == nil {
		it.lotID = lot.ID
	}
	if it.lineID > 0 && it.lotID > 0 {
		it.resolved = true
	}
}

func (it *interpreter) HandleEvent(deviceCode string, data map[string]any) {
	tag, _ := data["tag"].(string)
	rawData, ok := data["data"].([]byte)
	if !ok || len(rawData) == 0 {
		return
	}

	it.mu.Lock()
	it.resolve()
	lineID := it.lineID
	lotID := it.lotID
	it.mu.Unlock()

	if lineID == 0 || lotID == 0 {
		return
	}

	switch tag {
	case "production":
		it.handleProduction(deviceCode, rawData, lineID, lotID)
	case "status":
		it.handleStatus(deviceCode, rawData)
	}
}

func (it *interpreter) handleProduction(deviceCode string, data []byte, lineID, lotID int64) {
	if len(data) < 8 {
		return
	}

	doffCnt := binary.BigEndian.Uint32(data[0:4])
	if doffCnt == 0 {
		it.mu.Lock()
		it.lastDoffCnt[deviceCode] = 0
		it.mu.Unlock()
		return
	}

	it.mu.Lock()
	prev, exists := it.lastDoffCnt[deviceCode]
	if !exists {
		it.lastDoffCnt[deviceCode] = doffCnt
		it.mu.Unlock()
		return
	}
	if doffCnt <= prev {
		it.mu.Unlock()
		return
	}
	it.lastDoffCnt[deviceCode] = doffCnt
	it.mu.Unlock()

	for i := prev + 1; i <= doffCnt; i++ {
		it.createDoffing(deviceCode, int(i), lineID, lotID)
	}
}

func (it *interpreter) createDoffing(deviceCode string, doffNo int, lineID, lotID int64) {
	now := time.Now()
	bobbinCount := 24

	err := it.db.Transaction(func(tx *gorm.DB) error {
		doffing := model.Doffing{
			LineID:      lineID,
			WinderName:  deviceCode,
			DoffNo:      doffNo,
			LotID:       lotID,
			EndTime:     &now,
			BobbinCount: bobbinCount,
			CreatedAt:   now,
		}
		if err := tx.Create(&doffing).Error; err != nil {
			return err
		}

		bobbins := make([]model.Bobbin, bobbinCount)
		for i := range bobbins {
			bobbins[i] = model.Bobbin{
				DoffingID:       doffing.ID,
				PlaceInWinder:   i + 1,
				LotID:           lotID,
				Status:          "in_process",
				CurrentPosition: "winder",
				CreatedAt:       now,
				UpdatedAt:       now,
			}
		}
		return tx.Create(&bobbins).Error
	})

	if err != nil {
		if logger.L != nil {
			logger.L.Warnw("sim: create doffing failed", "device", deviceCode, "doff_no", doffNo, "error", err)
		}
		return
	}

	if logger.L != nil {
		logger.L.Infow("sim: doffing created",
			"device", deviceCode,
			"doff_no", doffNo,
			"bobbins", bobbinCount,
		)
	}
}

func (it *interpreter) handleStatus(deviceCode string, data []byte) {
	if len(data) < 11 {
		return
	}

	alarmCode := data[10]

	it.mu.Lock()
	prev, exists := it.lastAlarm[deviceCode]
	if !exists {
		it.lastAlarm[deviceCode] = alarmCode
		it.mu.Unlock()
		return
	}
	if alarmCode == prev {
		it.mu.Unlock()
		return
	}
	it.lastAlarm[deviceCode] = alarmCode
	it.mu.Unlock()

	if alarmCode == 0 {
		return
	}

	now := time.Now()
	alarm := model.Alarm{
		SourceType: "plc",
		SourceID:   deviceCode,
		AlarmCode:  fmt.Sprintf("SIM-%d", alarmCode),
		Level:      "warning",
		Message:    fmt.Sprintf("Simulation alarm %d from %s", alarmCode, deviceCode),
		Status:     "active",
		CreatedAt:  now,
	}
	if err := it.db.Create(&alarm).Error; err != nil {
		if logger.L != nil {
			logger.L.Warnw("sim: create alarm failed", "device", deviceCode, "error", err)
		}
		return
	}

	if logger.L != nil {
		logger.L.Infow("sim: alarm created", "device", deviceCode, "code", alarmCode)
	}
}
