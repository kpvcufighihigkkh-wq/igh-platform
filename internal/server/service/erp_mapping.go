package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

// ColumnRule 描述"ERP 表的一列从新系统的哪个字段取值"。
//
// 三种取值方式，按优先级：
//
//	Const   固定值（如空字符串、固定工厂代码）
//	From    点路径，支持嵌套与下标：lot.code / doffings[0]
//	Fallback 主路径取到空值时回退（如 end_time 为空就用 created_at）
type ColumnRule struct {
	Const    *string `mapstructure:"const"    yaml:"const"`
	From     string  `mapstructure:"from"     yaml:"from"`
	Fallback string  `mapstructure:"fallback" yaml:"fallback"`
}

// TableMapping 描述一张新系统表到 ERP 表的对应关系。
type TableMapping struct {
	// Target 是 ERP 侧的表名
	Target string `mapstructure:"target" yaml:"target"`
	// Key 是 ERP 表的唯一键列，用于生成 ON DUPLICATE KEY UPDATE
	Key string `mapstructure:"key" yaml:"key"`
	// Columns 是 ERP 列名 → 取值规则
	Columns map[string]ColumnRule `mapstructure:"columns" yaml:"columns"`
}

// MappingDoc 是 mapping.yaml 的整体结构。
type MappingDoc struct {
	Tables map[string]TableMapping `mapstructure:"tables" yaml:"tables"`
}

// LoadMappings 从 <configDir>/erp/mapping.yaml 读取字段映射。
//
// 为什么映射要配置化：ERP 侧的字段名与语义由对方系统决定，
// 我们无从假设（fangsi 的 doffings 有 order_code / paper_tube，
// 新系统根本没有同名字段）。写死在代码里，对方改一次表结构就要重新发版。
//
// 读不到文件时返回空映射并告警：此时 push_only 会明确失败（而不是假装推送成功）。
func LoadMappings(configDir string) map[string]TableMapping {
	path := filepath.Join(configDir, "erp", "mapping.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		logger.L.Warnw("erp: 未读到 mapping.yaml，push_only 将无法推送",
			"path", path, "error", err)
		return nil
	}

	var doc MappingDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		logger.L.Errorw("erp: mapping.yaml 解析失败", "path", path, "error", err)
		return nil
	}

	out := make(map[string]TableMapping, len(doc.Tables))
	for name, m := range doc.Tables {
		if m.Target == "" {
			logger.L.Warnw("erp: 映射缺少 target，已跳过", "table", name)
			continue
		}
		if len(m.Columns) == 0 {
			logger.L.Warnw("erp: 映射没有列定义，已跳过", "table", name)
			continue
		}
		out[name] = m
	}
	logger.L.Infow("erp mappings loaded", "tables", len(out))
	return out
}

// SetMappings 注入字段映射（由 main 在启动时载入）。
func (s *ERPService) SetMappings(m map[string]TableMapping) { s.mappings = m }

// buildView 把一条业务记录展开成"可被映射规则寻址"的视图。
//
// 视图是嵌套 map，外键关联的字段（批号、线路名）在这里一次查好，
// 避免映射规则里出现"自己去查数据库"这种隐式行为。
func (s *ERPService) buildView(item model.ERPSyncQueue) (map[string]any, error) {
	switch item.Target {
	case "doffings":
		return s.doffingView(item.RecordID)
	case "trolleys":
		return s.trolleyView(item.RecordID)
	case "pallets":
		return s.palletView(item.RecordID)
	default:
		return nil, fmt.Errorf("erp: 表 %q 没有视图构造器", item.Target)
	}
}

func (s *ERPService) doffingView(id int64) (map[string]any, error) {
	var d model.Doffing
	if err := s.db.First(&d, id).Error; err != nil {
		return nil, fmt.Errorf("erp: 读取落筒 %d 失败: %w", id, err)
	}
	view := structToMap(d)
	view["id"] = d.ID
	view["end_time"] = timeOrNil(d.EndTime)
	view["created_at"] = d.CreatedAt
	view["lot"] = s.lotView(d.LotID)
	view["line"] = s.lineView(d.LineID)
	return view, nil
}

func (s *ERPService) trolleyView(id int64) (map[string]any, error) {
	var t model.Trolley
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, fmt.Errorf("erp: 读取料车 %d 失败: %w", id, err)
	}
	view := structToMap(t)
	view["id"] = t.ID
	view["loading_time"] = t.LoadingTime
	view["created_at"] = t.CreatedAt

	if t.LotID != nil {
		view["lot"] = s.lotView(*t.LotID)
	} else {
		view["lot"] = map[string]any{}
	}
	if t.LineID != nil {
		view["line"] = s.lineView(*t.LineID)
	} else {
		view["line"] = map[string]any{}
	}

	// DoffingIDs 是 JSON 数组字符串。
	//
	// 同时提供两种形态：
	//   doffings           → []any，元素是落筒 id（对应 ERP 的 doffing_1_id 这类纯 id 列）
	//   doffing_records    → []any，元素是明细 map（winder_name / line_name 等）
	//
	// 不合并成一种：ERP 的 trolleys 表既存 id 又存机位名，
	// 若把 doffings[0] 变成 map，doffing_1_id 就取不到 id 了。
	var ids []int64
	if t.DoffingIDs != "" {
		if err := json.Unmarshal([]byte(t.DoffingIDs), &ids); err != nil {
			logger.L.Warnw("erp: 料车的 doffing_ids 解析失败", "trolley_id", id, "error", err)
		}
	}

	idList := make([]any, 0, len(ids))
	recList := make([]any, 0, len(ids))
	for _, did := range ids {
		idList = append(idList, did)

		rec := map[string]any{"id": did}
		var d model.Doffing
		if err := s.db.First(&d, did).Error; err != nil {
			logger.L.Warnw("erp: 料车关联的落筒不存在",
				"trolley_id", id, "doffing_id", did, "error", err)
		} else {
			rec["winder_name"] = d.WinderName
			rec["doff_no"] = d.DoffNo
			rec["line_name"] = s.lineName(d.LineID)
			rec["end_time"] = timeOrNil(d.EndTime)
		}
		recList = append(recList, rec)
	}
	view["doffings"] = idList
	view["doffing_records"] = recList
	return view, nil
}

// lineName 只取线路名，供嵌套场景使用（取不到返回空串而不是空对象）。
func (s *ERPService) lineName(id int64) string {
	if id <= 0 {
		return ""
	}
	var line model.ProductionLine
	if err := s.db.First(&line, id).Error; err != nil {
		return ""
	}
	return line.Name
}

func (s *ERPService) palletView(id int64) (map[string]any, error) {
	var p model.Pallet
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, fmt.Errorf("erp: 读取栈板 %d 失败: %w", id, err)
	}
	view := structToMap(p)
	view["id"] = p.ID
	view["created_at"] = p.CreatedAt
	view["product_date"] = timeOrNil(p.ProductDate)
	view["lot"] = s.lotView(p.LotID)
	return view, nil
}

func (s *ERPService) lotView(id int64) map[string]any {
	if id <= 0 {
		return map[string]any{}
	}
	var lot model.Lot
	if err := s.db.First(&lot, id).Error; err != nil {
		logger.L.Warnw("erp: 批次不存在，关联字段留空", "lot_id", id, "error", err)
		return map[string]any{}
	}
	m := structToMap(lot)
	m["code"] = lot.Code
	m["order_code"] = lot.OrderCode
	m["specification"] = lot.Specification
	m["paper_tube_color"] = lot.PaperTubeColor
	return m
}

func (s *ERPService) lineView(id int64) map[string]any {
	if id <= 0 {
		return map[string]any{}
	}
	var line model.ProductionLine
	if err := s.db.First(&line, id).Error; err != nil {
		logger.L.Warnw("erp: 线路不存在，关联字段留空", "line_id", id, "error", err)
		return map[string]any{}
	}
	return map[string]any{"code": line.Code, "name": line.Name}
}

// structToMap 把结构体转成 map，键用 json tag（与对外字段名一致）。
//
// 用 JSON 往返而不是反射逐字段：结构体字段本来就有 json tag，
// 这样映射规则里写的字段名和接口返回的字段名是同一套，不容易错。
func structToMap(v any) map[string]any {
	raw, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return map[string]any{}
	}
	return m
}

func timeOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

// applyMapping 按映射规则从视图取值，返回列名与值。
//
// 列名排序后返回：保证同一份配置每次生成的 SQL 一致，便于日志比对与缓存。
func applyMapping(m TableMapping, view map[string]any) ([]string, []any, error) {
	cols := make([]string, 0, len(m.Columns))
	for c := range m.Columns {
		cols = append(cols, c)
	}
	sortStrings(cols)

	vals := make([]any, 0, len(cols))
	for _, c := range cols {
		rule := m.Columns[c]
		v, err := resolveRule(rule, view)
		if err != nil {
			return nil, nil, fmt.Errorf("erp: 列 %s 取值失败: %w", c, err)
		}
		vals = append(vals, normalizeForMySQL(v))
	}
	return cols, vals, nil
}

func resolveRule(rule ColumnRule, view map[string]any) (any, error) {
	if rule.Const != nil {
		return *rule.Const, nil
	}
	if rule.From == "" {
		return nil, fmt.Errorf("规则既没有 const 也没有 from")
	}
	v := lookupPath(view, rule.From)
	if isEmptyValue(v) && rule.Fallback != "" {
		v = lookupPath(view, rule.Fallback)
	}
	return v, nil
}

// lookupPath 按 "a.b" / "a[0].b" 形式取值。
//
// 只支持这两种形态：映射配置是给现场运维手写的，
// 语法越简单越不容易写错，出错时的报错也更好懂。
func lookupPath(view map[string]any, path string) any {
	cur := any(view)
	for _, seg := range strings.Split(path, ".") {
		name, indexes := splitIndexes(seg)
		cur = lookupKey(cur, name)
		for _, idx := range indexes {
			cur = lookupIndex(cur, idx)
		}
	}
	return cur
}

// splitIndexes 把 "doffings[1][2]" 拆成 ("doffings", [1,2])。
func splitIndexes(seg string) (string, []int) {
	open := strings.Index(seg, "[")
	if open < 0 {
		return seg, nil
	}
	name := seg[:open]
	var idxs []int
	rest := seg[open:]
	for {
		l := strings.Index(rest, "[")
		if l < 0 {
			break
		}
		r := strings.Index(rest, "]")
		if r < l {
			break
		}
		if n, err := strconv.Atoi(rest[l+1 : r]); err == nil {
			idxs = append(idxs, n)
		}
		rest = rest[r+1:]
	}
	return name, idxs
}

func lookupKey(cur any, key string) any {
	switch m := cur.(type) {
	case map[string]any:
		return m[key]
	default:
		return nil
	}
}

func lookupIndex(cur any, idx int) any {
	list, ok := cur.([]any)
	if !ok || idx < 0 || idx >= len(list) {
		return nil
	}
	return list[idx]
}

func isEmptyValue(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	default:
		return false
	}
}

// normalizeForMySQL 把 Go 值转成 MySQL 驱动能接受的形态。
//
// 时间统一格式化成 'YYYY-MM-DD HH:MM:SS'：ERP 侧是 timestamp 列，
// 直接传 time.Time 在部分驱动/时区设置下会偏移。
func normalizeForMySQL(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case time.Time:
		return x.Format("2006-01-02 15:04:05")
	case *time.Time:
		if x == nil {
			return nil
		}
		return x.Format("2006-01-02 15:04:05")
	case float64:
		// JSON 往返会把整数变成 float64，落库前还原成整数避免 "1234.000000"
		if x == float64(int64(x)) {
			return int64(x)
		}
		return x
	default:
		return v
	}
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
