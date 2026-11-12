package packaging

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/igh-platform/igh-platform/internal/edge/printer"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/model"
	"github.com/igh-platform/igh-platform/internal/shared/palletcode"
	"github.com/igh-platform/igh-platform/internal/shared/protocol"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 打包机点表的字段名约定。
//
// 打包机的 UDT 布局随现场机型不同（旧系统是逐台写在 palletizers.settings 里），
// 所以不能像卷绕机那样把偏移写死在代码里。约定的是**语义字段名**，
// 现场点表用这些名字声明布局，控制器按名字取值——布局变了只改配置。
//
// 点表示例（DB 号与超时需现场核对）：
//
//	poll_groups:
//	  - name: pallet
//	    db: 760
//	    start: 0
//	    length: 64
//	    points:
//	      - { name: "Palletizer.PalletComplete", type: "int16", offset: 0 }
//	      - { name: "Palletizer.PalletAck",      type: "int16", offset: 2, direction: "write" }
//	      - { name: "Palletizer.LotCode",        type: "string", length: 20, offset: 4 }
//	      - { name: "Palletizer.GradeCode",      type: "string", length: 8,  offset: 24 }
//	      - { name: "Palletizer.BobbinCount",    type: "int16", offset: 32 }
//	      - { name: "Palletizer.Layers",         type: "int16", offset: 34 }
//	      - { name: "Palletizer.TotalWeight",    type: "float32", offset: 36 }
const (
	fieldPalletComplete = "Palletizer.PalletComplete"
	fieldPalletAck      = "Palletizer.PalletAck"
	fieldLotCode        = "Palletizer.LotCode"
	fieldGradeCode      = "Palletizer.GradeCode"
	fieldBobbinCount    = "Palletizer.BobbinCount"
	fieldLayers         = "Palletizer.Layers"
	fieldTotalWeight    = "Palletizer.TotalWeight"
	fieldProductDate    = "Palletizer.ProductDate"
)

// PLCPort 是控制器用到的 PLC 能力。
//
// 收窄成接口而不是直接依赖 plc.Manager：握手是纯逻辑（读信号 → 组事件 → 回写），
// 用假实现就能测，不需要连真机。
type PLCPort interface {
	Device(code string) (config.PLCDevice, bool)
	Write(deviceCode string, dbNumber, start int, data []byte) error
}

// PalletizerController 把"打包机堆完一垛"翻译成业务事件。
//
// 握手时序（与旧系统一致）：
//
//	PLC 置  Palletizer.PalletComplete = 1   本垛堆完
//	PC   读  批号/等级/锭数/层数/重量
//	PC   生成栈板编码、落本地记录、打标签、发出 packing_complete
//	PC   写  Palletizer.PalletAck = 1       已处理
//	PLC 清  PalletComplete = 0
//
// 不做"猜"：任何必需字段缺失都不生成事件，只告警。
// 栈板编码错了会打到实体标签上，发现时整垛已入库。
type PalletizerController struct {
	db  *gorm.DB
	prt *printer.Client
	plc PLCPort

	mu sync.Mutex
	// bindings 是打包机编码 → 绑定信息（PLC 设备、打包机号、厂编码）
	bindings map[string]config.EdgePalletizer
	// handled 记录最近处理过的 (设备, 打包机号) 时间，抑制重复触发
	handled map[string]time.Time
	// configDir 用于定位 labels/templates 下的标签模板
	configDir string
	// labelFile 是栈板标签模板文件名（相对 labels/templates）
	labelFile string

	// emit 可替换，测试时用来断言产生了什么事件
	emit func(types.EdgeEvent)
	// now 可替换，测试时固定生产日期
	now func() time.Time
	// printFn 可替换，测试时用来断言"打了什么标签"
	printFn func(printerCode, host string, port int, l printer.PalletLabel) error
}

func NewPalletizerController(db *gorm.DB, prt *printer.Client, plcMgr PLCPort) *PalletizerController {
	return &PalletizerController{
		db:        db,
		prt:       prt,
		plc:       plcMgr,
		bindings:  make(map[string]config.EdgePalletizer),
		handled:   make(map[string]time.Time),
		labelFile: "pallet.zpl",
		now:       time.Now,
	}
}

// SetConfigDir 指定配置目录，用于定位 labels/templates。
func (c *PalletizerController) SetConfigDir(dir string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.configDir = dir
}

// SetBindings 注入现场配置里的打包机绑定。
func (c *PalletizerController) SetBindings(list []config.EdgePalletizer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bindings = make(map[string]config.EdgePalletizer, len(list))
	for _, b := range list {
		if b.Device == "" || b.Number <= 0 {
			logger.L.Warnw("palletizer binding incomplete, ignored",
				"code", b.Code, "device", b.Device, "number", b.Number)
			continue
		}
		c.bindings[b.Device] = b
	}
	logger.L.Infow("palletizer bindings loaded", "count", len(c.bindings))
}

// SetEmit 注入事件上报回调。
func (c *PalletizerController) SetEmit(fn func(types.EdgeEvent)) { c.emit = fn }

// HandlePoints 处理一次 PLC 点表读数。
//
// 只看完整快照 values：PalletComplete 与 Ack 可能不在同一次变化里，
// 只看 changed 会漏。
func (c *PalletizerController) HandlePoints(deviceCode string, data map[string]any) {
	values, _ := data["values"].(map[string]any)
	if len(values) == 0 {
		values, _ = data["changed"].(map[string]any)
	}
	if len(values) == 0 || c.plc == nil {
		return
	}

	c.mu.Lock()
	binding, ok := c.bindings[deviceCode]
	c.mu.Unlock()
	if !ok {
		return // 该设备不是打包机
	}

	dev, ok := c.plc.Device(deviceCode)
	if !ok {
		return
	}

	complete := c.fieldInt(dev, values, fieldPalletComplete)
	ack := c.fieldInt(dev, values, fieldPalletAck)
	if complete != 1 || ack == 1 {
		return
	}

	// 同一垛可能连续多轮读到 complete=1（PLC 还没清），限流避免重复出垛
	key := fmt.Sprintf("%s/%d", deviceCode, binding.Number)
	c.mu.Lock()
	if last, seen := c.handled[key]; seen && time.Since(last) < 5*time.Second {
		c.mu.Unlock()
		return
	}
	c.handled[key] = time.Now()
	c.mu.Unlock()

	evt, err := c.buildEvent(dev, values, binding)
	if err != nil {
		logger.L.Errorw("palletizer: 无法生成垛事件，不回写 Ack",
			"device", deviceCode, "palletizer", binding.Number, "error", err)
		// 不回写 Ack：让 PLC 保持等待，由上层告警处理，
		// 绝不假装成功（与打标握手同一原则）。
		return
	}

	if c.emit != nil {
		c.emit(*evt)
	}

	if err := c.writeAck(dev, deviceCode); err != nil {
		logger.L.Errorw("palletizer: 回写 Ack 失败", "device", deviceCode, "error", err)
	}
}

// buildEvent 组装一垛的业务事件。
func (c *PalletizerController) buildEvent(dev config.PLCDevice, values map[string]any, b config.EdgePalletizer) (*types.EdgeEvent, error) {
	lotCode := c.fieldString(dev, values, fieldLotCode)
	if lotCode == "" {
		return nil, fmt.Errorf("点表缺少 %s（批号），无法出垛", fieldLotCode)
	}
	bobbinCount := c.fieldInt(dev, values, fieldBobbinCount)
	if bobbinCount <= 0 {
		return nil, fmt.Errorf("点表缺少 %s 或值非法（%d），无法出垛", fieldBobbinCount, bobbinCount)
	}

	now := c.now()
	productDate := c.productDate(dev, values, now)

	dailyID, err := c.nextDailyID(b.Number, productDate)
	if err != nil {
		return nil, err
	}

	code, err := palletcode.Build(b.PlantCode, productDate, b.Number, dailyID)
	if err != nil {
		return nil, err
	}

	grade := c.fieldString(dev, values, fieldGradeCode)
	layers := c.fieldInt(dev, values, fieldLayers)

	payload := map[string]any{
		"pallet_code":   code,
		"lot_code":      lotCode,
		"grade":         grade,
		"bobbin_count":  bobbinCount,
		"layers":        layers,
		"palletizer_id": b.Number,
		"daily_id":      dailyID,
		"product_date":  productDate.Format("2006-01-02"),
	}
	// 重量为 0 表示"未上报"，不带这个字段，
	// 否则中心会把它当成真实重量 0.000 参与平均。
	if w, ok := c.fieldFloat(dev, values, fieldTotalWeight); ok && w > 0 {
		payload["total_weight"] = w
	}

	// 丝饼列表来自本地上料记录：包装工位上料时把这些锭置为 current_position='packing'。
	// 取不到就不带，中心侧只记锭数——不猜，猜错会让追溯查到别的锭。
	bobbinIDs := c.bobbinsInPacking(lotCode, bobbinCount)
	if len(bobbinIDs) > 0 {
		payload["bobbin_ids"] = bobbinIDs
	}

	// 先在边端落一垛，再发事件。
	//
	// 顺序不能反：nextDailyID 是按本地已落的垛数算的，
	// 不落库的话同一天每一垛都会算出 daily_id=1、编码相同，
	// 第二垛起会被中心当作重复静默丢弃（实体栈板已经堆出来了，数据却没了）。
	//
	// 落库失败就不发事件、也不回写 Ack：让 PLC 保持等待并告警，
	// 与"缺少批号"同一原则——绝不假装成功。
	if err := c.persistLocal(code, lotCode, grade, bobbinCount, layers,
		payload, bobbinIDs, b.Number, dailyID, productDate); err != nil {
		return nil, err
	}

	// 打印栈板标签。
	//
	// 打印失败**不影响**事件上报与 Ack：垛已经堆出来了、数据已经落了，
	// 只是纸上少一张标签。现场可以补打，而丢数据补不回来。
	// 失败会把 label_time 留空，界面据此显示"未打标"，便于发现与补打。
	if err := c.printLabel(b, lotCode, code, grade, bobbinCount, layers, payload, productDate); err != nil {
		logger.L.Errorw("palletizer: 栈板标签打印失败，数据已记录、可补打",
			"device", dev.Code, "palletizer", b.Number, "pallet_code", code, "error", err)
	}

	return &types.EdgeEvent{
		EventType: protocol.EventPackingComplete,
		Payload:   payload,
	}, nil
}

// printLabel 打印一张栈板标签，成功后回写 label_time。
//
// 没有绑定打印机（label_printer 留空）时直接跳过——
// 有些工位是人工贴标，不该因此报错。
func (c *PalletizerController) printLabel(
	b config.EdgePalletizer, lotCode, palletCode, grade string,
	bobbinCount, layers int, payload map[string]any, productDate time.Time,
) error {
	if b.LabelPrinter == "" || c.prt == nil {
		return nil
	}
	p, ok := c.prt.ByCode(b.LabelPrinter)
	if !ok {
		return fmt.Errorf("打印机 %q 未配置", b.LabelPrinter)
	}

	label := printer.PalletLabel{
		PalletCode:  palletCode,
		LotCode:     lotCode,
		GradeCode:   grade,
		BobbinCount: bobbinCount,
		Layers:      layers,
		ProductDate: productDate.Format("2006-01-02"),
		Palletizer:  b.Number,
		Timestamp:   c.now().Format("2006/01/02 15:04"),
		BarCode:     palletCode,
	}
	if w, ok := payload["total_weight"].(float64); ok && w > 0 {
		label.TotalWeight = w
		label.HasWeight = true
	}

	tmpl := c.loadLabelTemplate()

	c.mu.Lock()
	fn := c.printFn
	c.mu.Unlock()
	if fn != nil {
		if err := fn(p.Code, p.Host, p.Port, label); err != nil {
			return err
		}
	} else {
		z := printer.NewZebra(p.Host, p.Port, 0)
		if err := z.Print(tmpl, label); err != nil {
			return err
		}
	}

	// 只有真的打成功了才记时间：留空表示"未打标"，界面据此提示补打
	now := c.now()
	if err := c.db.Model(&model.Pallet{}).Where("code = ?", palletCode).
		Update("label_time", now).Error; err != nil {
		logger.L.Warnw("palletizer: 回写 label_time 失败", "pallet_code", palletCode, "error", err)
	}
	return nil
}

// loadLabelTemplate 读取标签模板。
//
// 模板放在 configs/labels/templates/ 下，随配置一起分发。
// 读不到就用内置默认模板并告警：打不出一张格式合理的标签是现场问题，
// 但也不该因为一个文件缺失让整垛数据出不去。
func (c *PalletizerController) loadLabelTemplate() string {
	c.mu.Lock()
	dir, file := c.configDir, c.labelFile
	c.mu.Unlock()

	if dir == "" || file == "" {
		return printer.DefaultPalletLabelTemplate()
	}
	path := filepath.Join(dir, "labels", "templates", file)
	raw, err := os.ReadFile(path)
	if err != nil {
		logger.L.Warnw("palletizer: 读取标签模板失败，使用内置模板",
			"path", path, "error", err)
		return printer.DefaultPalletLabelTemplate()
	}
	return string(raw)
}

// persistLocal 把这一垛写进边端本地库。
//
// 本地记录有两个作用：
//  1. 提供 nextDailyID 的计数基准（断网时也要连续出垛，不能依赖中心）
//  2. 断网期间在边端页面上仍能看到垛与锭的关联
//
// 用 OnConflict DoNothing：重启后重复处理同一垛不会报错，
// 编码本身就承担幂等键的作用。
func (c *PalletizerController) persistLocal(
	code, lotCode, grade string, bobbinCount, layers int,
	payload map[string]any, bobbinIDs []int64,
	palletizerNumber, dailyID int, productDate time.Time,
) error {
	var lot model.Lot
	if err := c.db.Where("code = ?", lotCode).First(&lot).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询批次 %s 失败: %w", lotCode, err)
	}

	weight, _ := payload["total_weight"].(float64)
	palletizer := int64(palletizerNumber)
	daily := dailyID
	date := truncateToDate(productDate)

	pallet := model.Pallet{
		Code:         code,
		LotID:        lot.ID,
		GradeCode:    grade,
		Layers:       layers,
		BobbinCount:  bobbinCount,
		PalletizerID: &palletizer,
		DailyID:      &daily,
		ProductDate:  &date,
		Status:       "created",
		CreatedAt:    c.now(),
	}
	if weight > 0 {
		pallet.TotalWeight = &weight
	}

	return c.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&pallet).Error; err != nil {
			return fmt.Errorf("落本地栈板失败: %w", err)
		}
		if len(bobbinIDs) == 0 {
			return nil
		}
		links := make([]model.PalletBobbin, 0, len(bobbinIDs))
		for _, id := range bobbinIDs {
			links = append(links, model.PalletBobbin{
				PalletID: pallet.ID, BobbinID: id, CreatedAt: c.now(),
			})
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&links).Error; err != nil {
			return fmt.Errorf("落栈板-丝饼关联失败: %w", err)
		}
		return tx.Model(&model.Bobbin{}).Where("id IN ?", bobbinIDs).
			Updates(map[string]any{
				"status":           "packed",
				"current_position": "pallet",
				"updated_at":       c.now(),
			}).Error
	})
}

// nextDailyID 计算该打包机当天的第几垛。
//
// 以本地已落库的栈板为基准 +1：边端断网时也要能连续出垛，
// 不能依赖中心。同一 (打包机, 生产日期) 范围内取最大值。
//
// 用区间比较而不是 DATE(product_date) = '...'：
// 后者在两种数据库上的类型推导不一样，区间比较两边都稳。
func (c *PalletizerController) nextDailyID(palletizerNumber int, productDate time.Time) (int, error) {
	start := truncateToDate(productDate)
	end := start.AddDate(0, 0, 1)

	var maxID *int
	err := c.db.Model(&model.Pallet{}).
		Where("palletizer_id = ? AND product_date >= ? AND product_date < ?",
			palletizerNumber, start, end).
		Select("MAX(daily_id)").Scan(&maxID).Error
	if err != nil {
		return 0, fmt.Errorf("统计当天垛数失败: %w", err)
	}
	if maxID == nil {
		return 1, nil
	}
	return *maxID + 1, nil
}

// productDate 取生产日期。
//
// 优先用 PLC 上报（点表里配了才读得到），否则用当前日期。
// 跨零点打包时两者不同，标签日期必须跟产品走。
func (c *PalletizerController) productDate(dev config.PLCDevice, values map[string]any, fallback time.Time) time.Time {
	raw := c.fieldString(dev, values, fieldProductDate)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return truncateToDate(fallback)
	}
	for _, layout := range []string{"2006-01-02", "060102", "20060102"} {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t
		}
	}
	logger.L.Warnw("palletizer: 生产日期格式无法识别，改用当天",
		"device", dev.Code, "raw", raw)
	return truncateToDate(fallback)
}

// bobbinsInPacking 取该批次在包装工位上待装的丝饼。
//
// 与分拣的 findBobbin 同一思路：靠 current_position 标记，
// 不依赖任何位置表。数量不足返回已找到的，由中心侧按实际关联。
func (c *PalletizerController) bobbinsInPacking(lotCode string, limit int) []int64 {
	var lot model.Lot
	if err := c.db.Where("code = ?", lotCode).First(&lot).Error; err != nil {
		return nil
	}
	var bobbins []model.Bobbin
	c.db.Where("lot_id = ? AND current_position = ?", lot.ID, "packing").
		Order("id").Limit(limit).Find(&bobbins)

	ids := make([]int64, 0, len(bobbins))
	for _, b := range bobbins {
		ids = append(ids, b.ID)
	}
	return ids
}

// writeAck 回写"已处理"信号。
func (c *PalletizerController) writeAck(dev config.PLCDevice, deviceCode string) error {
	for _, g := range dev.PollGroups {
		for _, p := range config.ExpandGroup(g) {
			if p.Winder != 0 || !strings.HasSuffix(p.Field, fieldPalletAck) {
				continue
			}
			buf := make([]byte, 2)
			buf[0] = 0
			buf[1] = 1
			return c.plc.Write(deviceCode, p.DB, g.Start+p.Offset, buf)
		}
	}
	return fmt.Errorf("点表里找不到 %s", fieldPalletAck)
}

// fieldInt 按字段名后缀取整数值。
func (c *PalletizerController) fieldInt(dev config.PLCDevice, values map[string]any, suffix string) int {
	for _, g := range dev.PollGroups {
		for _, p := range config.ExpandGroup(g) {
			if p.Winder != 0 || !strings.HasSuffix(p.Field, suffix) {
				continue
			}
			if n, ok := toInt(values[p.Key()]); ok {
				return n
			}
		}
	}
	return 0
}

// fieldString 按字段名后缀取字符串值。
func (c *PalletizerController) fieldString(dev config.PLCDevice, values map[string]any, suffix string) string {
	for _, g := range dev.PollGroups {
		for _, p := range config.ExpandGroup(g) {
			if p.Winder != 0 || !strings.HasSuffix(p.Field, suffix) {
				continue
			}
			if s, ok := values[p.Key()].(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

// fieldFloat 按字段名后缀取浮点值。
func (c *PalletizerController) fieldFloat(dev config.PLCDevice, values map[string]any, suffix string) (float64, bool) {
	for _, g := range dev.PollGroups {
		for _, p := range config.ExpandGroup(g) {
			if p.Winder != 0 || !strings.HasSuffix(p.Field, suffix) {
				continue
			}
			switch v := values[p.Key()].(type) {
			case float64:
				return v, true
			case float32:
				return float64(v), true
			}
		}
	}
	return 0, false
}

func truncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int8:
		return int(n), true
	case uint8:
		return int(n), true
	case int16:
		return int(n), true
	case uint16:
		return int(n), true
	case int32:
		return int(n), true
	case uint32:
		return int(n), true
	case int64:
		return int(n), true
	case uint64:
		return int(n), true
	case float32:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
