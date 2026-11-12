package sorting

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/igh-platform/igh-platform/internal/edge/printer"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/geometry"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/model"
	"gorm.io/gorm"
)

// PrinterController 实现外检工位的"PLC 请求 → 打标 → 回写完成"握手。
//
// 协议来源：遗留系统 O17006/src/SortingPrinter.js（逐行核对）。
//
// 握手时序（每台打标机一套，index 从 1 开始）：
//
//	PLC 置  Start[i] = 1        表示"请给第 i 台打标机打这一版"
//	PLC 置  Step     = n        表示当前是模块内第 n 段
//	PLC 置  ModuleRead = 1      表示模块数据（锭列表）已准备好
//	PC   读  四个信号，满足条件即打印
//	PC   写  Done[i] = 1        表示"已打完"
//	PLC 清  Start[i] = 0
//
// 打印哪个锭：由"第 i 个头 + 第 n 段"查锭位映射表得到模块内锭位，
// 再取该锭位对应的丝饼。映射表是现场配置，随中心下发。
// PLCPort 是打印控制器真正用到的 PLC 能力。
//
// 收窄成接口而不是直接依赖 plc.Manager：
// 打标握手是纯逻辑（读信号 → 判断 → 打印 → 回写），
// 用假实现就能测，不需要连真机或真 PLC。
type PLCPort interface {
	// Device 返回设备点表
	Device(code string) (config.PLCDevice, bool)
	// Write 写一段字节到 PLC
	Write(deviceCode string, dbNumber, start int, data []byte) error
}

type PrinterController struct {
	db  *gorm.DB
	prt *printer.Client
	plc PLCPort

	mu sync.Mutex
	// heads 是当前生效的外检锭位映射（来自现场配置）
	heads *geometry.HeadMaps
	// macsa 是打标机连接（按打印机编码索引），复用连接避免每版重建
	macsa map[string]*printer.Macsa
	// macsaAddr 记录打标机地址，配置变化时用于重连
	macsaAddr map[string]string
	// lastPrint 用于抑制同一 (head, step) 的重复触发
	lastPrint map[string]time.Time
	// printFn 可替换，测试时用来断言"打了什么"
	printFn func(printerCode, labelName string, vars []printer.MacsaVariable) error
}

func NewPrinterController(db *gorm.DB, prt *printer.Client, plcMgr PLCPort) *PrinterController {
	return &PrinterController{
		db:        db,
		prt:       prt,
		plc:       plcMgr,
		macsa:     make(map[string]*printer.Macsa),
		macsaAddr: make(map[string]string),
		lastPrint: make(map[string]time.Time),
	}
}

// SetHeadMaps 注入现场配置里的锭位映射。
func (c *PrinterController) SetHeadMaps(hm *geometry.HeadMaps) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.heads = hm
}

// HandlePoints 处理一次打标机相关的点表读数。
//
// 由插件的 OnPLCEvent 调用。data["values"] 是本次读到的完整快照，
// 因为 Start/Done/Step 可能不同时变化，必须看快照而不是变化集。
//
// 取值方式：按设备点表展开后的 PointAddress.Key() 去快照里取，
// 而不是拼 "Start[3]" 这类字符串后缀——点表的键形如
// "laser_start/W3/LaserPrinter.Start"，后缀匹配会永远落空。
func (c *PrinterController) HandlePoints(deviceCode string, data map[string]any) {
	values, _ := data["values"].(map[string]any)
	if len(values) == 0 {
		values, _ = data["changed"].(map[string]any)
	}
	if len(values) == 0 || c.plc == nil {
		return
	}

	dev, ok := c.plc.Device(deviceCode)
	if !ok {
		return
	}

	starts, dones, alarms := c.signalMaps(dev, values)
	step := c.singleField(dev, values, "LaserPrinter.Step")
	moduleRead := c.singleField(dev, values, "LaserPrinter.ModuleRead")
	module := c.singleField(dev, values, "LaserPrinter.Id_Module")

	if step <= 0 || moduleRead != 1 {
		return
	}

	for head, start := range starts {
		if start != 1 || dones[head] != 0 {
			continue
		}
		if alarm := alarms[head]; alarm != 0 {
			logger.L.Warnw("sorting printer in alarm, skip",
				"device", deviceCode, "head", head, "alarm", alarm)
			continue
		}

		key := fmt.Sprintf("%s/%d/%d", deviceCode, head, step)
		c.mu.Lock()
		if last, ok := c.lastPrint[key]; ok && time.Since(last) < 2*time.Second {
			c.mu.Unlock()
			continue // 同一段重复触发，避免重复打印
		}
		c.lastPrint[key] = time.Now()
		c.mu.Unlock()

		if err := c.printOne(deviceCode, head, step, module); err != nil {
			logger.L.Errorw("sorting print failed",
				"device", deviceCode, "head", head, "step", step, "error", err)
			continue
		}

		// 回写完成信号。写失败时不回写，让 PLC 保持等待，
		// 由上层告警——绝不假装成功。
		if err := c.writeDone(deviceCode, head); err != nil {
			logger.L.Errorw("sorting print: write done failed",
				"device", deviceCode, "head", head, "error", err)
		}
	}
}

// printOne 打印"第 head 头第 step 段"对应的那一个丝饼标签。
func (c *PrinterController) printOne(deviceCode string, head, step, module int) error {
	c.mu.Lock()
	heads := c.heads
	c.mu.Unlock()

	if heads == nil {
		return fmt.Errorf("外检锭位映射未配置")
	}

	headCount := heads.HeadCounts()
	if len(headCount) == 0 {
		return fmt.Errorf("外检锭位映射为空")
	}
	// 当前只有一套映射时直接用；多套时按最大头数匹配
	hc := headCount[0]
	if len(headCount) > 1 {
		for _, n := range headCount {
			if n >= head {
				hc = n
				break
			}
		}
	}

	place, err := heads.Position(hc, head, step)
	if err != nil {
		return fmt.Errorf("查锭位失败 (head=%d step=%d): %w", head, step, err)
	}

	bobbin, err := c.findBobbin(module, place)
	if err != nil {
		return err
	}

	label, err := c.buildLabel(bobbin)
	if err != nil {
		return err
	}

	p, err := c.resolvePrinter(deviceCode, head)
	if err != nil {
		return err
	}
	labelName := p.Label
	if labelName == "" {
		return fmt.Errorf("设备 %s 的第 %d 号头未配置标签模板名", deviceCode, head)
	}

	// 测试可替换打印实现，避免握手逻辑必须依赖真机才能验证
	c.mu.Lock()
	fn := c.printFn
	c.mu.Unlock()
	if fn != nil {
		return fn(p.Code, labelName, label)
	}

	m, err := c.macsaConn(p)
	if err != nil {
		return err
	}
	return m.Print(labelName, label)
}

// findBobbin 取模块内指定位置的丝饼。
//
// 新库里没有旧系统的 positions 表，模块内位置由 position_detail 表达：
// 外检上线时丝饼的 position_detail 形如 {"module":3,"place":15}。
//
// 注意：position_detail 是 []byte（各数据库可能存成 BLOB），
// 不能依赖 SQL 的 LIKE 去过滤 JSON 内容，取出后在 Go 里解析更可靠。
func (c *PrinterController) findBobbin(module, place int) (*model.Bobbin, error) {
	var candidates []model.Bobbin
	if err := c.db.Where("current_position = ?", "module").Find(&candidates).Error; err != nil {
		return nil, err
	}

	for i := range candidates {
		var d struct {
			Module int `json:"module"`
			Place  int `json:"place"`
		}
		if err := json.Unmarshal(candidates[i].PositionDetail, &d); err != nil {
			continue
		}
		if d.Place != place {
			continue
		}
		// module > 0 时才校验模块号；未标注模块的按位置匹配
		if module > 0 && d.Module != 0 && d.Module != module {
			continue
		}
		return &candidates[i], nil
	}
	return nil, fmt.Errorf("模块 %d 内没有位置 %d 的丝饼", module, place)
}

// buildLabel 按旧系统的四个模板字段组装标签内容。
//
// 字段号与内容取自 O17006/src/SortingPrinter.js：
//
//	0: 工厂前缀 + 规格 + 班次 + 等级
//	4: 工厂代码 + 生产线 + 卷绕机-锭位
//	3: 日期 + 时间
//	5: 丝饼前缀 + PLC 锭号
func (c *PrinterController) buildLabel(b *model.Bobbin) ([]printer.MacsaVariable, error) {
	var lot model.Lot
	if b.LotID > 0 {
		c.db.First(&lot, b.LotID)
	}
	var doffing model.Doffing
	if b.DoffingID > 0 {
		c.db.First(&doffing, b.DoffingID)
	}

	spec := lot.Specification
	if spec == "" {
		spec = lot.Name
	}

	shift := ""
	if doffing.ShiftID != nil {
		shift = fmt.Sprintf("%d", *doffing.ShiftID)
	}

	endDate, endTime := "", ""
	if doffing.EndTime != nil {
		endDate = doffing.EndTime.Format("2006/01/02")
		endTime = doffing.EndTime.Format("15:04")
	}

	winderNo, _ := splitWinderName(doffing.WinderName)

	return []printer.MacsaVariable{
		{Field: "0", Value: joinNonEmpty(" ", lot.Prefix, spec, shift, b.SortingGrade)},
		{Field: "4", Value: fmt.Sprintf("%d-%d", winderNo, b.PlaceInWinder)},
		{Field: "3", Value: joinNonEmpty("   ", endDate, endTime)},
		{Field: "5", Value: fmt.Sprintf("%d", b.ID)},
	}, nil
}

// splitWinderName 把 "winder-L01/W3/left" 拆成 (机位号, 线号)。
func splitWinderName(name string) (winder int, line string) {
	parts := strings.Split(name, "/")
	if len(parts) < 2 {
		return 0, name
	}
	line = strings.TrimPrefix(parts[0], "winder-")
	fmt.Sscanf(strings.TrimPrefix(parts[1], "W"), "%d", &winder)
	return winder, line
}

func joinNonEmpty(sep string, parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, sep)
}

// resolvePrinter 按"设备 + 头号"找到打标机配置。
//
// 现场在 edge.yaml 的 printers 段按 IP 直配，不依赖任何命名约定：
//
//	printers:
//	  - { code: "sort07", device: "sorting-L04", head: 7,
//	      host: "192.168.4.84", port: 3490, label: "mae3" }
func (c *PrinterController) resolvePrinter(deviceCode string, head int) (*printer.Printer, error) {
	p, ok := c.prt.ByHead(deviceCode, head)
	if !ok {
		return nil, fmt.Errorf("设备 %s 的第 %d 号头没有配置打标机", deviceCode, head)
	}
	return p, nil
}

// macsaConn 取（或建立）打标机连接，按打印机编码缓存。
func (c *PrinterController) macsaConn(p *printer.Printer) (*printer.Macsa, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if m, ok := c.macsa[p.Code]; ok {
		return m, nil
	}

	m, err := printer.NewMacsa(p.Host, p.Port)
	if err != nil {
		return nil, err
	}
	c.macsa[p.Code] = m
	c.macsaAddr[p.Code] = fmt.Sprintf("%s:%d", p.Host, p.Port)
	return m, nil
}

// signalMaps 从点表展开结果中取出三组按打标头编号索引的信号值。
//
// 点表用 repeat 描述"12 个 INT 连续排列"，展开后每个元素带着 Winder 序号；
// 这里用 Winder 作为打标头编号，正好对应旧系统的 Start[1..12]。
func (c *PrinterController) signalMaps(dev config.PLCDevice, values map[string]any) (
	starts, dones, alarms map[int]int) {

	starts = make(map[int]int)
	dones = make(map[int]int)
	alarms = make(map[int]int)

	for _, g := range dev.PollGroups {
		for _, p := range config.ExpandGroup(g) {
			if p.Winder == 0 {
				continue
			}
			v, ok := values[p.Key()]
			if !ok {
				continue
			}
			n, ok := toInt(v)
			if !ok {
				continue
			}
			switch {
			case strings.HasSuffix(p.Field, "LaserPrinter.Start"):
				starts[p.Winder] = n
			case strings.HasSuffix(p.Field, "LaserPrinter.Done"):
				dones[p.Winder] = n
			case strings.HasSuffix(p.Field, "LaserPrinter.Alarm"):
				alarms[p.Winder] = n
			}
		}
	}
	return starts, dones, alarms
}

// singleField 取一个非重复字段的值（Step / ModuleRead / Id_Module）。
func (c *PrinterController) singleField(dev config.PLCDevice, values map[string]any, suffix string) int {
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

// writeDone 回写 Done[i]=1。
//
// 写入地址来自设备的点表：按字段名和打标头序号找到该字段的分组与偏移，
// 而不是硬编码 DB 号和字节位置——点表是现场可配的。
func (c *PrinterController) writeDone(deviceCode string, head int) error {
	return c.writeInt(deviceCode, "LaserPrinter.Done", head, 1)
}

// writeInt 按字段名 + 打标头序号把 int16 值写入 PLC。
//
// 点表里 Done 是一个 repeat 分组（12 或 16 个 INT 连续排列），
// 第 head 个元素的地址由 p.Winder 标识，不能用 "Done[3]" 这种字符串后缀匹配。
func (c *PrinterController) writeInt(deviceCode, fieldName string, head, value int) error {
	if c.plc == nil {
		return fmt.Errorf("PLC 管理器未注入，无法回写 %s", fieldName)
	}

	dev, ok := c.plc.Device(deviceCode)
	if !ok {
		return fmt.Errorf("设备 %s 不存在", deviceCode)
	}

	for _, g := range dev.PollGroups {
		for _, p := range config.ExpandGroup(g) {
			if p.Winder != head || !strings.HasSuffix(p.Field, fieldName) {
				continue
			}
			buf := make([]byte, 2)
			binary.BigEndian.PutUint16(buf, uint16(value))
			// p.Offset 是相对分组起始的偏移，写入要用绝对地址
			return c.plc.Write(deviceCode, p.DB, g.Start+p.Offset, buf)
		}
	}
	return fmt.Errorf("点表里找不到字段 %s 的第 %d 个元素", fieldName, head)
}

func intField(values map[string]any, suffix string) int {
	for k, v := range values {
		if strings.HasSuffix(k, suffix) {
			switch n := v.(type) {
			case int:
				return n
			case int16:
				return int(n)
			case uint16:
				return int(n)
			case int32:
				return int(n)
			case uint32:
				return int(n)
			case int64:
				return int(n)
			case float64:
				return int(n)
			}
		}
	}
	return 0
}
