package winding

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/igh-platform/igh-platform/internal/edge/printer"
	"github.com/igh-platform/igh-platform/internal/shared/bobbincode"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/model"
	"github.com/igh-platform/igh-platform/internal/shared/protocol"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

// bobbinLabelTemplate 是丝锭标签的模板文件名（不含扩展名），
// 对应 <配置目录>/labels/templates/bobbin.zpl。
const bobbinLabelTemplate = "bobbin"

// PointInterpreter 把"点表字段变化"翻译成"业务事件"。
//
// 输入是 plc.Manager 上报的 plc_points 事件：
//
//	{
//	  "event_type": "plc_points",
//	  "group": "winder_data",
//	  "db": 555,
//	  "changed": {
//	    "winder_data/W1/left/DatiBobine_DaWinder.Doff_No": 42,
//	    "winder_data/W1/left/DatiBobine_DaWinder.CodeNumber": "G2091200482S",
//	    ...
//	  }
//	}
//
// 输出是业务事件（doffing），由 sync.Engine 入队并上传中心。
//
// 设计要点：
//   - 变更检测按 (winder, side) 维度，而不是按设备维度
//   - 只有 Doff_No 变化才触发业务事件
//   - 业务事件的 payload 直接来自同一次读取的其它字段，不额外查 PLC
//   - 本地也落一份记录（离线时前端仍可查看），但以中心为准
type PointInterpreter struct {
	db *gorm.DB
	mu sync.Mutex
	// lastDoffNo 记录每个 (device, winder, side) 上一次的 Doff_No
	lastDoffNo map[string]int64
	// pending 暂存同一次读取里的其它字段，供 Doff_No 触发时一并带上
	pending map[string]map[string]any
	// lotCache 缓存批号 → lot 信息
	lotCache map[string]*model.Lot

	// lineCode 是上报中心时的线路编码（来自 edge.line.code）
	lineCode string
	// bobbinsPerDoff 是一次落筒产出的锭数，必须由配置显式给出。
	// 不同机型/线路的锭数不同，这里不做任何默认值猜测：
	// 未配置时拒绝产生落筒事件，宁可报错也不写入错误的锭数。
	bobbinsPerDoff int
	// bobbinLabelPrinter 是丝锭标签打印机编码，空表示不打印。
	bobbinLabelPrinter string
	// prt 是打印机客户端。可为 nil：未配置打印机时不打印，
	// 这是默认状态（现场不一定装了标签机）。
	prt *printer.Client
}

// doffNoKey 返回 SystemSetting 里存最近落筒号的 Key。
// 格式：winding.last_doff_no.<winderID>，每个机位/侧别独立存。
func doffNoKey(winderID string) string {
	return "winding.last_doff_no." + winderID
}

func NewPointInterpreter(db *gorm.DB) *PointInterpreter {
	pi := &PointInterpreter{
		db:         db,
		lastDoffNo: make(map[string]int64),
		pending:    make(map[string]map[string]any),
		lotCache:   make(map[string]*model.Lot),
	}
	// 从本地库恢复上次持久化的落筒号。
	//
	// 目的：消除边端重启后第一次落筒丢失的窗口。
	//
	// 原来的逻辑：首次读到的值当作基线跳过 → 重启后第一次落筒丢失。
	// 现在：恢复后 seen=true、prev=上次的值，只要值变了就触发。
	// 失败则退化为原来行为（不影响后续正常运行）。
	if db != nil {
		pi.loadLastDoffNos()
	}
	return pi
}

// loadLastDoffNos 从 SystemSetting 恢复各机位的最近落筒号。
func (pi *PointInterpreter) loadLastDoffNos() {
	var settings []model.SystemSetting
	if err := pi.db.Where("category = ?", "winding_doff_no").Find(&settings).Error; err != nil {
		logger.L.Warnw("winding: 恢复落筒号失败，退化为基线模式", "error", err)
		return
	}
	for _, s := range settings {
		n, err := strconv.ParseInt(s.Value, 10, 64)
		if err != nil || n <= 0 {
			continue
		}
		// 去掉前缀，还原 winderID
		key := strings.TrimPrefix(s.Key, "winding.last_doff_no.")
		if key == s.Key {
			continue // 格式不符，跳过
		}
		pi.lastDoffNo[key] = n
	}
	if len(pi.lastDoffNo) > 0 {
		logger.L.Infow("winding: 已从本地库恢复落筒号基线", "count", len(pi.lastDoffNo))
	}
}

// persistDoffNo 把最新的落筒号写入 SystemSetting，供重启恢复用。
// 失败只打 warn，不影响落筒事件上报。
func (pi *PointInterpreter) persistDoffNo(winderID string, doffNo int64) {
	if pi.db == nil {
		return
	}
	key := doffNoKey(winderID)
	val := strconv.FormatInt(doffNo, 10)
	now := time.Now()
	setting := model.SystemSetting{
		Key:       key,
		Value:     val,
		Category:  "winding_doff_no",
		UpdatedAt: now,
	}
	if err := pi.db.Where("key = ?", key).
		Assign(model.SystemSetting{Value: val, UpdatedAt: now}).
		FirstOrCreate(&setting).Error; err != nil {
		logger.L.Warnw("winding: 持久化落筒号失败", "key", key, "error", err)
	}
}

// Configure 注入边端配置：线路编码与锭数。
func (pi *PointInterpreter) Configure(cfg *config.EdgeConfig) {
	if cfg == nil {
		return
	}
	if cfg.Edge.Line.Code != "" {
		pi.lineCode = cfg.Edge.Line.Code
	}
	if cfg.Edge.BobbinsPerDoff > 0 {
		pi.bobbinsPerDoff = cfg.Edge.BobbinsPerDoff
	} else {
		logger.L.Errorw("winding: bobbins_per_doff 未配置，落筒事件将被拒绝上报",
			"hint", "在 edge.yaml 的 edge 段配置 bobbins_per_doff（取自线路主数据）")
	}
	pi.bobbinLabelPrinter = cfg.Edge.BobbinLabelPrinter
}

// SetPrinter 注入打印机客户端（由 app 在构造后调用）。
func (pi *PointInterpreter) SetPrinter(prt *printer.Client) {
	pi.mu.Lock()
	pi.prt = prt
	pi.mu.Unlock()
}

// pointKey 是解析后的一条字段标识。
type pointKey struct {
	Group  string
	Winder int
	Side   string
	Field  string
}

func (k pointKey) winderID(deviceCode string) string {
	if k.Side == "" {
		return fmt.Sprintf("%s/W%d", deviceCode, k.Winder)
	}
	return fmt.Sprintf("%s/W%d/%s", deviceCode, k.Winder, k.Side)
}

// parsePointKey 解析点表键，支持两种形态：
//
//	group/W1/left/Field.Name  双胞胎（带侧别）
//	group/W1/Field.Name       单面锭位（无侧别）
func parsePointKey(key string) (pointKey, bool) {
	parts := strings.SplitN(key, "/", 4)
	var field string
	var side string
	switch len(parts) {
	case 4:
		side = parts[2]
		field = parts[3]
	case 3:
		// 单面锭位：group/W1/Field
		field = parts[2]
	default:
		return pointKey{}, false
	}
	w := strings.TrimPrefix(parts[1], "W")
	n, err := strconv.Atoi(w)
	if err != nil {
		return pointKey{}, false
	}
	return pointKey{
		Group:  parts[0],
		Winder: n,
		Side:   side,
		Field:  field,
	}, true
}

// HandlePoints 处理一次点表读数变化，返回需要上报的业务事件。
//
// data 里有两份数据：
//   - changed：本次变化的字段，用于**触发**判断
//   - values ：本次完整快照，用于**拼装**事件内容
//
// 必须用 values 而不是 changed 拼装：像批号 CodeNumber、机台名 MachineName
// 这类字段可能整班不变，永远不会出现在 changed 里。
func (pi *PointInterpreter) HandlePoints(deviceCode string, data map[string]any) []types.EdgeEvent {
	changed, ok := data["changed"].(map[string]any)
	if !ok || len(changed) == 0 {
		return nil
	}
	// 完整快照；缺失时退回 changed（兼容旧调用方）
	values, _ := data["values"].(map[string]any)
	if len(values) == 0 {
		values = changed
	}

	// 1. 把完整快照按 (winder, side) 归组，同时用 changed 记录本次哪些字段变了
	pi.mu.Lock()
	snapshots := make(map[string]map[string]any)
	triggered := make(map[string]bool)

	for rawKey, value := range values {
		k, ok := parsePointKey(rawKey)
		if !ok {
			continue
		}
		id := k.winderID(deviceCode)
		if snapshots[id] == nil {
			snapshots[id] = make(map[string]any)
		}
		snapshots[id][k.Field] = value
		// 累加到 pending，保证"字段先到、Doff_No 后到"也能拼出完整记录
		if pi.pending[id] == nil {
			pi.pending[id] = make(map[string]any)
		}
		pi.pending[id][k.Field] = value
	}

	for rawKey := range changed {
		k, ok := parsePointKey(rawKey)
		if !ok {
			continue
		}
		if isDoffField(k.Field) {
			triggered[k.winderID(deviceCode)] = true
		}
	}
	pi.mu.Unlock()

	// 2. 只检查"本次 Doff_No 确实变化"的 (winder, side)
	var events []types.EdgeEvent
	ids := make([]string, 0, len(triggered))
	for id := range triggered {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		field := findField(snapshots[id], "Doff_No", "DoffNo")
		if field == "" {
			continue
		}
		doffNo, ok := toInt64(snapshots[id][field])
		if !ok || doffNo <= 0 {
			continue
		}

		pi.mu.Lock()
		prev, seen := pi.lastDoffNo[id]
		pi.lastDoffNo[id] = doffNo
		snapshot := copyMap(pi.pending[id])
		delete(pi.pending, id)
		pi.mu.Unlock()

		// 每次更新都持久化：消除重启后第一次落筒丢失的窗口。
		//
		// 同步写而不是丢 goroutine：两个连续落筒号（5→6）如果并发写，
		// 提交顺序可能颠倒，库里留下 5，重启后就会把 6 当成新落筒再报一次。
		// 落筒号变化是分钟级事件，一次 SQLite 写入的开销可以忽略。
		pi.persistDoffNo(id, doffNo)

		// 首次读取只记录基线，不产生落筒（避免重启时补发历史）
		if !seen {
			continue
		}
		// 只在**值没变**时跳过，不要求递增。
		//
		// PLC 的落筒号会循环（从 N 回到 1）。原来写的是 doffNo <= prev，
		// 循环之后每一次落筒都会被当成"没有前进"而丢弃，直到计数器
		// 重新爬过 N —— 整整一个周期的落筒数据全部静默丢失，日志里
		// 连一条记录都没有。这是必须避开的失败形态。
		//
		// 改成 != ：只要落筒号变了就是一次新落筒。重复读取（值相同）
		// 仍然会被跳过，所以不会因为轮询而重复上报。
		if doffNo == prev {
			continue
		}

		// 落筒号跳变告警：边端停机期间如果发生了多次落筒，
		// 恢复后只能报最新那一次，中间的永远拿不回来了。
		// 告警让现场知道"这个时段可能有丢失"，能结合 PLC 侧记录对账。
		//
		// 循环（99→1）本身不算跳变，不告警：循环时 delta < 0，
		// 只有 delta > 1 且不跨循环时才是真正的跳变。
		// 循环上限未知（各机型不同），这里用 50 作为保守的"循环界"——
		// 正常一次推进不会超过 1，落筒周期是分钟级；超过 50 一定有问题。
		delta := doffNo - prev
		if delta > 1 {
			logger.L.Errorw("winding: 落筒号跳变——边端停机期间可能有落筒丢失",
				"device", deviceCode, "winder_id", id,
				"prev", prev, "now", doffNo,
				"missed_estimate", delta-1,
				"hint", "对照 PLC 侧 EndTimeOfPack 记录确认丢失范围")
		}

		k, _ := splitWinderID(id)
		ev := pi.buildDoffingEvent(deviceCode, k.Winder, k.Side, doffNo, snapshot)
		if ev != nil {
			events = append(events, *ev)
		}
	}
	return events
}

// isDoffField 判断字段是否是落筒号。
func isDoffField(field string) bool {
	return strings.HasSuffix(field, "Doff_No") || strings.HasSuffix(field, "DoffNo")
}

func splitWinderID(id string) (pointKey, bool) {
	// id 形如 "winder-L01/W3/left"（双胞胎）或 "winder-L01/W3"（单面锭位）
	parts := strings.Split(id, "/")
	if len(parts) != 2 && len(parts) != 3 {
		return pointKey{}, false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(parts[1], "W"))
	if err != nil {
		return pointKey{}, false
	}
	k := pointKey{Winder: n}
	if len(parts) == 3 {
		k.Side = parts[2]
	}
	return k, true
}

// buildDoffingEvent 根据同一次读取的字段拼出业务事件。
//
// payload 的键必须与中心 eventbus.Dispatcher 的约定一致，
// 否则中心解析不到 line_code / lot_code，会把整条落筒丢弃：
//
//	line_code   线路编码（来自 edge.line.code）
//	winder_name 机位名（设备/机位/侧别，对应旧库 winders.winder_name）
//	lot_code    批号（PLC 的 CodeNumber）
//	doff_no     落筒号
//	bobbin_count 本次落筒的锭数
//	end_time    落筒完成时间（RFC3339）
func (pi *PointInterpreter) buildDoffingEvent(deviceCode string, winder int, side string, doffNo int64, fields map[string]any) *types.EdgeEvent {
	// 锭数未知时宁可不上报，也不要写入一个猜出来的数字：
	// 错误的锭数会让中心的锭数与实物永久对不上，且事后无法区分。
	if pi.bobbinsPerDoff <= 0 {
		logger.L.Errorw("winding: skip doffing, bobbins_per_doff 未配置",
			"device", deviceCode, "winder", winder, "side", side, "doff_no", doffNo)
		return nil
	}

	now := time.Now()
	// 截到秒并固定时区：这是编码里的日期段来源，也是上报的 end_time 来源。
	// 两者必须完全同源，否则边端打出来的码和中心入库的码会差一秒。
	doffAt := now.In(bobbincode.CodeZone).Truncate(time.Second)

	lotCode := firstString(fields, "CodeNumber", "ProductionData")
	winderName := fmt.Sprintf("%s/W%d", deviceCode, winder)
	if side != "" {
		winderName = fmt.Sprintf("%s/W%d/%s", deviceCode, winder, side)
	}

	payload := map[string]any{
		// —— 中心消费的规范字段 ——
		"line_code":    pi.lineCode,
		"winder_name":  winderName,
		"lot_code":     lotCode,
		"doff_no":      doffNo,
		"bobbin_count": pi.bobbinsPerDoff,
		// 用 doffAt 而不是 now：它与丝锭编码里的日期段是同一个值，
		// 中心用这个 end_time 生成编码就能得到完全相同的字符串。
		"end_time": doffAt.Format(time.RFC3339),

		// —— 明细字段，便于排查与前端展示 ——
		"device":      deviceCode,
		"winder":      winder,
		"side":        side,
		"position":    firstString(fields, "POSITION_Name", "PositionNo"),
		"machine":     firstString(fields, "MachineName", "MachineNo"),
		"code_number": lotCode,
		"yarn_type":   firstString(fields, "Yarn_Type"),
		"plc_end_time": firstInt(fields, "EndTimeOfPack", "EndTime1"),
		"recorded_at": now.UTC().Format(time.RFC3339),
	}

	// 预缺陷/称重等扩展字段原样透传，便于中心侧扩展
	for k, v := range fields {
		if _, exists := payload[k]; !exists {
			payload[k] = v
		}
	}

	// 本地落一份记录（离线可查），失败不影响上报
	pi.persistLocal(deviceCode, winder, side, doffNo, lotCode, now)

	// 打印丝锭标签。
	//
	// 编码由 bobbincode.Build 确定性推导，与中心 ProcessDoffing 用的是
	// **同一个函数 + 同一个 doffAt**：边端断网时算出的码，
	// 和联网后中心生成的完全一致，不需要事先协调。
	//
	// 打印失败不影响事件上报：丝锭已经产出，数据比纸重要。
	pi.printBobbinLabels(lotCode, winder, side, doffNo, doffAt)

	logger.L.Infow("winding: doffing detected",
		"device", deviceCode, "winder", winder, "side", side,
		"doff_no", doffNo, "code", lotCode)

	return &types.EdgeEvent{
		EventType: protocol.EventDoffing,
		Timestamp: now,
		Payload:   payload,
	}
}

// printBobbinLabels 为本次落筒的每个锭打印一张标签。
//
// 逐锭发请求而不是批量：ZPL 打印机没有事务语义，
// 一张失败不该拖累其余。失败只记日志，不重试——
// 重试会把同一张标签打两遍，工人手里多一张会引起混淆；
// 需要补打时走 /edge/print 手工触发。
func (pi *PointInterpreter) printBobbinLabels(lotCode string, winder int, side string, doffNo int64, doffAt time.Time) {
	pi.mu.Lock()
	prt := pi.prt
	printerCode := pi.bobbinLabelPrinter
	count := pi.bobbinsPerDoff
	lineCode := pi.lineCode
	pi.mu.Unlock()

	// 四样缺一就不打印：没配打印机（现场常态）、没注入客户端、
	// 锭数未知（此时连编码都算不出来）。
	if prt == nil || printerCode == "" || count <= 0 {
		return
	}
	// 产线编码为空时编码会跨线重复。打印出去的标签是贴在实物上的，
	// 打错了要一张张撕，比不打的代价高得多——宁可跳过并告警。
	if lineCode == "" {
		logger.L.Errorw("winding: 未配置 edge.line.code，跳过丝锭标签打印",
			"hint", "产线编码是丝锭编码唯一性的一部分，缺失会导致跨线撞码")
		return
	}

	ok, failed := 0, 0
	for place := 1; place <= count; place++ {
		code, err := bobbincode.Build(lotCode, lineCode, winder, side, int(doffNo), place, doffAt)
		if err != nil {
			logger.L.Warnw("winding: 生成丝锭编码失败，跳过该锭标签",
				"winder", winder, "side", side, "doff_no", doffNo, "place", place, "error", err)
			failed++
			continue
		}

		data := map[string]any{
			"BobbinCode": code,
			"LotCode":    lotCode,
			"LineCode":   lineCode,
			"Winder":     winder,
			"Side":       sideLetterForLabel(side),
			"DoffNo":     doffNo,
			"Place":      place,
		}
		if err := prt.PrintTemplate(printerCode, bobbinLabelTemplate, data); err != nil {
			logger.L.Errorw("winding: 丝锭标签打印失败，数据已记录、可补打",
				"printer", printerCode, "bobbin", code, "error", err)
			failed++
			continue
		}
		ok++
	}

	if ok > 0 || failed > 0 {
		logger.L.Infow("winding: 丝锭标签打印完成",
			"printer", printerCode, "doff_no", doffNo, "ok", ok, "failed", failed)
	}
}

// sideLetterForLabel 把侧别转成标签上的短标记。
func sideLetterForLabel(side string) string {
	switch strings.ToLower(side) {
	case "left", "l", "a":
		return "L"
	case "right", "r", "b":
		return "R"
	default:
		return ""
	}
}

// persistLocal 在边端本地库写一条 doffing（尽力而为）。
//
// 注意：本地记录使用负数 ID 区段，避免与中心下发的主键冲突。
func (pi *PointInterpreter) persistLocal(deviceCode string, winder int, side string, doffNo int64, lotCode string, now time.Time) {
	if pi.db == nil {
		return
	}

	lineID := pi.resolveLineID()
	lotID := pi.resolveLotID(lotCode)

	winderName := fmt.Sprintf("%s/W%d", deviceCode, winder)
	if side != "" {
		winderName = fmt.Sprintf("%s/W%d/%s", deviceCode, winder, side)
	}

	err := pi.db.Transaction(func(tx *gorm.DB) error {
		d := model.Doffing{
			LineID:      lineID,
			WinderName:  winderName,
			DoffNo:      int(doffNo),
			LotID:       lotID,
			EndTime:     &now,
			BobbinCount: pi.bobbinsPerDoff,
			CreatedAt:   now,
		}
		return tx.Create(&d).Error
	})
	if err != nil {
		logger.L.Warnw("winding: local doffing persist failed",
			"device", deviceCode, "doff_no", doffNo, "error", err)
	}
}

func (pi *PointInterpreter) resolveLineID() int64 {
	var line model.ProductionLine
	if err := pi.db.Where("status = ?", "running").First(&line).Error; err == nil {
		return line.ID
	}
	return 0
}

func (pi *PointInterpreter) resolveLotID(code any) int64 {
	codeStr, _ := code.(string)
	if codeStr == "" {
		var lot model.Lot
		if err := pi.db.Where("status = ?", "active").First(&lot).Error; err == nil {
			return lot.ID
		}
		return 0
	}
	if lot, ok := pi.lotCache[codeStr]; ok && lot != nil {
		return lot.ID
	}
	var lot model.Lot
	if err := pi.db.Where("code = ?", strings.TrimSpace(codeStr)).First(&lot).Error; err == nil {
		pi.lotCache[codeStr] = &lot
		return lot.ID
	}
	return 0
}

// ---------------------------------------------------------------------------
// 辅助
// ---------------------------------------------------------------------------

func findField(m map[string]any, candidates ...string) string {
	for _, c := range candidates {
		for k := range m {
			if strings.HasSuffix(k, c) || strings.HasSuffix(k, "."+c) {
				return k
			}
		}
	}
	return ""
}

// firstString 按"后缀匹配"取字段值。
//
// 点表里的字段名带命名空间前缀（如 DatiBobine_DaWinder.POSITION_Name），
// 而业务语义只关心末段（POSITION_Name），因此这里必须用后缀匹配。
func firstString(m map[string]any, candidates ...string) string {
	for _, c := range candidates {
		// 先精确匹配
		if v, ok := m[c]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
		// 再后缀匹配
		for k, v := range m {
			if k == c || strings.HasSuffix(k, "."+c) {
				if s, ok := v.(string); ok && s != "" {
					return s
				}
			}
		}
	}
	return ""
}

// firstInt 同样按后缀匹配。
func firstInt(m map[string]any, candidates ...string) int64 {
	for _, c := range candidates {
		if v, ok := m[c]; ok {
			if n, ok := toInt64(v); ok {
				return n
			}
		}
		for k, v := range m {
			if k == c || strings.HasSuffix(k, "."+c) {
				if n, ok := toInt64(v); ok {
					return n
				}
			}
		}
	}
	return 0
}

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int8:
		return int64(n), true
	case uint8:
		return int64(n), true
	case int16:
		return int64(n), true
	case uint16:
		return int64(n), true
	case int32:
		return int64(n), true
	case uint32:
		return int64(n), true
	case int64:
		return n, true
	case uint64:
		return int64(n), true
	case float32:
		return int64(n), true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}
