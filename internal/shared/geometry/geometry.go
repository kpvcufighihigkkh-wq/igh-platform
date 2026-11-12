// Package geometry 承载"物理几何表"——这些数据描述的是机械设备的物理布局，
// 无法从任何公式推导，必须从现场/遗留系统原样承接。
//
// 两类几何：
//  1. LoadingSequence：卷绕机锭位 + 落筒侧(A/B) → 容器位置（料车/吊架）
//     对应遗留系统的 loading_bobbins_sequence 表 + calc_place() 函数。
//  2. HeadMap：打标头 + 段序 → 模块内锭位
//     对应遗留系统的 getBobbinPosition(index, step)。
//
// 本包只做"查表"，不含任何业务逻辑。所有数据由 tools/geomgen 从遗留系统提取，
// 并通过 TestGeometryMatchesLegacyGolden 与老系统的原始数据逐条比对。
package geometry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// 领域类型
// ---------------------------------------------------------------------------

// SpindleSide 是落筒侧。遗留系统用 'A' / 'B' 表示同一卷绕机的两次落筒。
type SpindleSide byte

const (
	SpindleA SpindleSide = 'A'
	SpindleB SpindleSide = 'B'
)

// ErrNotFound 表示几何表中不存在对应的映射项。
// 注意：遗留系统的 calc_place 在查不到时返回 0 且不报错，导致静默的位置错误。
// 新系统必须显式返回错误，由调用方决定如何处理。
var ErrNotFound = errors.New("geometry: mapping not found")

// ---------------------------------------------------------------------------
// LoadingSequence
// ---------------------------------------------------------------------------

// LoadingRow 是装载映射的一行：某个锭位在 A/B 两次落筒时分别落到容器的哪个位置。
type LoadingRow struct {
	SpindlePlace int // 卷绕机上的锭位，1-based
	A            int // A 侧落筒 → 容器位置
	B            int // B 侧落筒 → 容器位置
}

// LoadingPattern 是一套完整的装载模式（遗留系统的 tunnel_side）。
type LoadingPattern struct {
	// Code 是遗留系统 loading_side 的取值：sx / dx / sxw / dxw / fdy
	Code string
	// Map 按 SpindlePlace 升序排列
	Map []LoadingRow
	// index 用于 O(1) 查找：spindlePlace → 行下标
	index map[int]int
}

// LoadingSequences 是全部装载模式 + 容量。
type LoadingSequences struct {
	// Capacity 是容器容量（遗留系统为 24）
	Capacity int
	// Patterns 按 code 索引
	Patterns map[string]*LoadingPattern
}

// Place 按遗留 calc_place 的语义查表：
//
//	calc_place(IN_old_place, IN_spindle, IN_tunnel_side, IN_container_side)
//
// 与遗留实现的差异：查不到时返回 ErrNotFound，而不是静默返回 0。
func (l *LoadingSequences) Place(patternCode string, spindlePlace int, side SpindleSide) (int, error) {
	p, ok := l.Patterns[patternCode]
	if !ok {
		return 0, fmt.Errorf("geometry: unknown loading pattern %q: %w", patternCode, ErrNotFound)
	}
	idx, ok := p.index[spindlePlace]
	if !ok {
		return 0, fmt.Errorf("geometry: spindle place %d not found in pattern %q: %w",
			spindlePlace, patternCode, ErrNotFound)
	}
	row := p.Map[idx]
	switch side {
	case SpindleA:
		return row.A, nil
	case SpindleB:
		return row.B, nil
	default:
		return 0, fmt.Errorf("geometry: invalid spindle side %q", string(rune(side)))
	}
}

// PatternCodes 返回全部模式代码（有序，便于稳定输出/测试）。
func (l *LoadingSequences) PatternCodes() []string {
	// 固定顺序，避免 map 遍历随机
	known := []string{"dx", "dxw", "fdy", "sx", "sxw"}
	var out []string
	for _, c := range known {
		if _, ok := l.Patterns[c]; ok {
			out = append(out, c)
		}
	}
	// 追加未知代码（保持可扩展）
	for c := range l.Patterns {
		found := false
		for _, k := range known {
			if k == c {
				found = true
				break
			}
		}
		if !found {
			out = append(out, c)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// HeadMap
// ---------------------------------------------------------------------------

// HeadEntry 是一条打标映射：第 head 号打标头在第 step 段应打印哪个锭位。
type HeadEntry struct {
	Head     int
	Step     int
	Position int
}

// HeadMap 是一套打标头布局（遗留系统的 udt{N}Macsa.js）。
type HeadMap struct {
	// HeadCount 是打标头数量（6 / 12 / 16）
	HeadCount int
	// StepsPerHead 是每头在一个模块内参与的打标段数（4 / 2 / 2）
	StepsPerHead int
	// Capacity 是模块容量，= HeadCount * StepsPerHead（24 / 24 / 32）
	Capacity int
	// Entries 保留原始顺序，便于与遗留数据逐条比对
	Entries []HeadEntry
	// lookup[(head, step)] → position
	lookup map[headKey]int
}

type headKey struct {
	head int
	step int
}

// HeadMaps 按头数索引的多套布局。
type HeadMaps struct {
	ByHeadCount map[int]*HeadMap
}

// Position 返回第 head 号头在第 step 段应打印的锭位。
//
// 与遗留实现的差异：越界时返回 ErrNotFound，而不是返回 undefined / 空标签。
// 调用方如需"约占位标签"，应显式处理 ErrNotFound。
func (h *HeadMaps) Position(headCount, head, step int) (int, error) {
	m, ok := h.ByHeadCount[headCount]
	if !ok {
		return 0, fmt.Errorf("geometry: no head map for %d heads: %w", headCount, ErrNotFound)
	}
	if head < 1 || head > m.HeadCount {
		return 0, fmt.Errorf("geometry: head %d out of range 1..%d: %w", head, m.HeadCount, ErrNotFound)
	}
	if step < 1 || step > m.StepsPerHead {
		return 0, fmt.Errorf("geometry: step %d out of range 1..%d: %w", step, m.StepsPerHead, ErrNotFound)
	}
	pos, ok := m.lookup[headKey{head: head, step: step}]
	if !ok {
		return 0, fmt.Errorf("geometry: no mapping for head %d step %d: %w", head, step, ErrNotFound)
	}
	return pos, nil
}

// HeadCounts 返回全部头数（升序）。
func (h *HeadMaps) HeadCounts() []int {
	out := make([]int, 0, len(h.ByHeadCount))
	for n := range h.ByHeadCount {
		out = append(out, n)
	}
	// 简单插入排序，数量极小
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// YAML 结构（与 tools/geomgen 的输出一一对应）
// ---------------------------------------------------------------------------

// 结构定义见 document.go：LoadingDoc / LoadingPatternDoc / LoadingRowDoc
// 与 HeadMapsDoc / HeadMapDoc / HeadEntryDoc。

// ---------------------------------------------------------------------------
// 加载与校验
// ---------------------------------------------------------------------------

// Load 从目录加载两份 YAML 并执行完整校验。
func Load(dir string) (*LoadingSequences, *HeadMaps, error) {
	ls, err := LoadLoadingSequences(filepath.Join(dir, "loading_sequences.yaml"))
	if err != nil {
		return nil, nil, err
	}
	hm, err := LoadHeadMaps(filepath.Join(dir, "head_maps.yaml"))
	if err != nil {
		return nil, nil, err
	}
	return ls, hm, nil
}

// LoadLoadingSequences 加载并校验装载映射。
//
// 只负责读文件，构造与校验统一走 BuildLoadingSequences，
// 保证"文件导入"和"中心下发"两条路径的校验规则完全一致。
func LoadLoadingSequences(path string) (*LoadingSequences, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("geometry: read %s: %w", path, err)
	}
	var doc LoadingDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("geometry: parse %s: %w", path, err)
	}
	return BuildLoadingSequences(&doc, path)
}

// LoadHeadMaps 加载并校验打标头映射。
func LoadHeadMaps(path string) (*HeadMaps, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("geometry: read %s: %w", path, err)
	}
	var doc HeadMapsDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("geometry: parse %s: %w", path, err)
	}
	return BuildHeadMaps(&doc, path)
}

// ValidatePattern 校验一套装载映射：A、B 两列合并后必须是 1..capacity 的完整排列。
func ValidatePattern(p *LoadingPattern, capacity int) []string {
	var problems []string
	seen := make(map[int]int, capacity) // position → spindlePlace
	for _, row := range p.Map {
		if row.SpindlePlace < 1 {
			problems = append(problems, fmt.Sprintf("invalid spindle_place %d", row.SpindlePlace))
		}
		for _, v := range []struct {
			side string
			val  int
		}{{"A", row.A}, {"B", row.B}} {
			if v.val < 1 || v.val > capacity {
				problems = append(problems, fmt.Sprintf(
					"spindle %d side %s = %d out of range 1..%d",
					row.SpindlePlace, v.side, v.val, capacity))
				continue
			}
			if prev, dup := seen[v.val]; dup {
				problems = append(problems, fmt.Sprintf(
					"position %d duplicated (spindle %d and %d)", v.val, prev, row.SpindlePlace))
				continue
			}
			seen[v.val] = row.SpindlePlace
		}
	}
	for v := 1; v <= capacity; v++ {
		if _, ok := seen[v]; !ok {
			problems = append(problems, fmt.Sprintf("position %d missing", v))
		}
	}
	return problems
}

// ValidateHeadMap 校验一套打标映射：position 必须是 1..capacity 的完整排列，
// 且条目数必须等于 headCount × stepsPerHead。
func ValidateHeadMap(m *HeadMap) []string {
	var problems []string
	if m.HeadCount <= 0 {
		problems = append(problems, fmt.Sprintf("invalid head_count %d", m.HeadCount))
	}
	if m.StepsPerHead <= 0 {
		problems = append(problems, fmt.Sprintf("invalid steps_per_head %d", m.StepsPerHead))
	}
	expected := m.HeadCount * m.StepsPerHead
	if expected > 0 && m.Capacity != expected {
		problems = append(problems, fmt.Sprintf(
			"capacity %d != head_count*steps_per_head %d", m.Capacity, expected))
	}
	if expected > 0 && len(m.Entries) != expected {
		problems = append(problems, fmt.Sprintf(
			"entries %d != head_count*steps_per_head %d", len(m.Entries), expected))
	}
	seen := make(map[int]int, len(m.Entries))
	for _, e := range m.Entries {
		if e.Head < 1 || e.Head > m.HeadCount {
			problems = append(problems, fmt.Sprintf("head %d out of range", e.Head))
		}
		if e.Step < 1 || e.Step > m.StepsPerHead {
			problems = append(problems, fmt.Sprintf("step %d out of range", e.Step))
		}
		if e.Position < 1 || e.Position > m.Capacity {
			problems = append(problems, fmt.Sprintf("position %d out of range", e.Position))
			continue
		}
		if _, dup := seen[e.Position]; dup {
			problems = append(problems, fmt.Sprintf("position %d duplicated", e.Position))
			continue
		}
		seen[e.Position] = e.Head*1000 + e.Step
	}
	for v := 1; v <= m.Capacity; v++ {
		if _, ok := seen[v]; !ok {
			problems = append(problems, fmt.Sprintf("position %d missing", v))
		}
	}
	return problems
}

// ---------------------------------------------------------------------------
// PalletLayout：包装栈板垛型
// ---------------------------------------------------------------------------

// PalletLayer 是栈板的一层。
type PalletLayer struct {
	// Layer 是层号，1-based，从底层开始
	Layer int
	// Positions 是这一层的位置数
	Positions int
	// Start 是这一层第一个位置在整个垛里的序号（含），用于把"第 N 锭"换算成层/位
	Start int
}

// PalletLayout 是一套完整的垛型。
//
// 它回答的是"这一垛长什么样"，不是"已经装了多少"——
// 装载进度来自 PLC（Layers / BobbinCount），这里只提供几何换算。
type PalletLayout struct {
	Code string
	// Layers 按层号升序，每个元素的位置数可以不同
	Layers []PalletLayer
	// BobbinsPerPosition 是每个位置放几锭
	BobbinsPerPosition int
	// TotalPositions 是整垛的位置总数
	TotalPositions int
	// Capacity 是整垛的锭数 = TotalPositions * BobbinsPerPosition
	Capacity int
}

// TotalBobbins 返回整垛锭数。
func (p *PalletLayout) TotalBobbins() int { return p.Capacity }

// Locate 把"整垛第 n 锭"（1-based）换算成层号与层内位置号。
//
// 为什么需要它：PLC 只给出已装锭数或层数，页面上要显示"当前装到第 3 层第 5 位"
// 必须做这个换算。算不出来的原因只有越界——那说明 PLC 给的值与配置的垛型对不上，
// 属于配置错误或点表映射错误，必须报错而不是静默夹到边界。
func (p *PalletLayout) Locate(bobbinIndex int) (layer, position int, err error) {
	if bobbinIndex < 1 || bobbinIndex > p.Capacity {
		return 0, 0, fmt.Errorf(
			"geometry: bobbin index %d out of range 1..%d for layout %q",
			bobbinIndex, p.Capacity, p.Code)
	}
	// 先把锭序号折算成位置序号，再在层表里定位
	posIndex := (bobbinIndex-1)/p.BobbinsPerPosition + 1
	for i := len(p.Layers) - 1; i >= 0; i-- {
		l := p.Layers[i]
		if posIndex >= l.Start {
			return l.Layer, posIndex - l.Start + 1, nil
		}
	}
	return 0, 0, fmt.Errorf("geometry: layout %q has no layer covering position %d", p.Code, posIndex)
}

// ValidatePalletLayout 校验一套垛型，返回全部问题。
func ValidatePalletLayout(p *PalletLayout) []string {
	var problems []string
	if p == nil {
		return []string{"垛型为空"}
	}
	if p.Code == "" {
		problems = append(problems, "垛型缺少 code")
	}
	if p.BobbinsPerPosition <= 0 {
		problems = append(problems, fmt.Sprintf(
			"每个位置的锭数必须 > 0，当前 %d", p.BobbinsPerPosition))
	}
	if len(p.Layers) == 0 {
		problems = append(problems, "垛型没有任何层")
	}
	// 层号必须从 1 开始连续递增：允许跳号会让"第 N 层"这类显示直接错位，
	// 而且现场手填时最容易犯的就是漏写一层。
	for i, l := range p.Layers {
		if l.Layer != i+1 {
			problems = append(problems, fmt.Sprintf(
				"层号必须从 1 开始连续，第 %d 项的层号是 %d", i+1, l.Layer))
		}
		if l.Positions <= 0 {
			problems = append(problems, fmt.Sprintf(
				"第 %d 层的位置数必须 > 0，当前 %d", l.Layer, l.Positions))
		}
	}
	if p.Capacity > 0 && p.TotalPositions*p.BobbinsPerPosition != p.Capacity {
		problems = append(problems, fmt.Sprintf(
			"容量 %d 与 位置数×每位置锭数 %d 不一致",
			p.Capacity, p.TotalPositions*p.BobbinsPerPosition))
	}
	return problems
}

// BuildPalletLayout 由文档构造垛型，并做完整校验。
//
// 与 BuildLoadingSequences / BuildHeadMaps 同一套约定：
// source 只用于错误信息，构造与校验不依赖任何存储格式。
func BuildPalletLayout(doc *PalletLayoutDoc, source string) (*PalletLayout, error) {
	if doc == nil {
		return nil, fmt.Errorf("geometry: %s: empty pallet layout document", source)
	}

	layout := &PalletLayout{
		Code:               doc.Code,
		Layers:             make([]PalletLayer, 0, len(doc.Layers)),
		BobbinsPerPosition: doc.BobbinsPerPosition,
	}

	// 按层号排序后构造，同时算出每层的起始位置序号。
	// 输入顺序不要求有序——现场编辑时上下移动行很常见，
	// 要求"必须按顺序填"只会制造无谓的错误。
	sorted := make([]PalletLayerDoc, len(doc.Layers))
	copy(sorted, doc.Layers)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Layer < sorted[j].Layer })

	next := 1
	for _, l := range sorted {
		layout.Layers = append(layout.Layers, PalletLayer{
			Layer: l.Layer, Positions: l.Positions, Start: next,
		})
		next += l.Positions
	}
	layout.TotalPositions = next - 1
	layout.Capacity = layout.TotalPositions * layout.BobbinsPerPosition

	if problems := ValidatePalletLayout(layout); len(problems) > 0 {
		return nil, fmt.Errorf("geometry: %s %q invalid: %v", source, doc.Code, problems)
	}
	return layout, nil
}

// PalletLayouts 是按 code 索引的多套垛型。
type PalletLayouts struct {
	byCode map[string]*PalletLayout
}

// Get 按 code 取垛型。
func (p *PalletLayouts) Get(code string) (*PalletLayout, error) {
	l, ok := p.byCode[code]
	if !ok {
		return nil, fmt.Errorf("geometry: unknown pallet layout %q: %w", code, ErrNotFound)
	}
	return l, nil
}

// Codes 返回全部垛型编号（有序，便于稳定输出/测试）。
func (p *PalletLayouts) Codes() []string {
	out := make([]string, 0, len(p.byCode))
	for c := range p.byCode {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// BuildPalletLayouts 由文档构造多套垛型。
func BuildPalletLayouts(doc *PalletLayoutsDoc, source string) (*PalletLayouts, error) {
	if doc == nil || len(doc.Layouts) == 0 {
		return nil, fmt.Errorf("geometry: %s: no pallet layouts", source)
	}
	out := &PalletLayouts{byCode: make(map[string]*PalletLayout, len(doc.Layouts))}
	for i := range doc.Layouts {
		l, err := BuildPalletLayout(&doc.Layouts[i], source)
		if err != nil {
			return nil, err
		}
		if _, dup := out.byCode[l.Code]; dup {
			return nil, fmt.Errorf("geometry: %s: duplicate pallet layout %q", source, l.Code)
		}
		out.byCode[l.Code] = l
	}
	return out, nil
}
