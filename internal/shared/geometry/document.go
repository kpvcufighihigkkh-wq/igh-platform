package geometry

import "fmt"

// 本文件把"几何数据"定义成可序列化文档，并给出与存储格式无关的构造入口。
//
// 背景：装载映射和打标映射都是现场配置，不能写死在仓库里。
//   - 开发期：tools/geomgen 从遗留系统生成 YAML，作为初始数据导入
//   - 运行期：中心数据库保存 JSON 文档，由现场在 Web UI 编辑后下发
//
// 因此校验和构造必须只依赖结构体，不依赖 YAML 文件；
// Load* 系列函数只是"读文件 + 调用 Build*"的薄壳。

// LoadingDoc 是装载映射的完整文档。
//
// 语义：卷绕机锭位(spindle_place) + 落筒侧(A/B) → 容器位置。
type LoadingDoc struct {
	Capacity int                 `yaml:"capacity" json:"capacity"`
	Patterns []LoadingPatternDoc `yaml:"patterns" json:"patterns"`
}

// LoadingPatternDoc 是一套装载模式（遗留系统的 loading_side）。
type LoadingPatternDoc struct {
	Code string          `yaml:"code" json:"code"`
	Map  []LoadingRowDoc `yaml:"map" json:"map"`
}

// LoadingRowDoc 是装载映射的一行。
type LoadingRowDoc struct {
	SpindlePlace int `yaml:"spindle_place" json:"spindle_place"`
	A            int `yaml:"a" json:"a"`
	B            int `yaml:"b" json:"b"`
}

// HeadMapsDoc 是打标头映射的完整文档。
//
// 语义：打标头(head) + 段序(step) → 模块内锭位。
type HeadMapsDoc struct {
	HeadMaps []HeadMapDoc `yaml:"head_maps" json:"head_maps"`
}

// HeadMapDoc 是一套打标头布局（遗留系统的 udt{N}Macsa.js）。
type HeadMapDoc struct {
	HeadCount    int            `yaml:"head_count" json:"head_count"`
	StepsPerHead int            `yaml:"steps_per_head" json:"steps_per_head"`
	Capacity     int            `yaml:"capacity" json:"capacity"`
	Entries      []HeadEntryDoc `yaml:"entries" json:"entries"`
}

// HeadEntryDoc 是一条打标映射。
type HeadEntryDoc struct {
	Head     int `yaml:"head" json:"head"`
	Step     int `yaml:"step" json:"step"`
	Position int `yaml:"position" json:"position"`
}

// PalletLayoutDoc 是包装栈板的物理布局。
//
// 语义：一垛由若干层组成，每层有若干个"位置"，每个位置放若干锭。
//
// 为什么必须可配置而不是写死：
//   - 不同包装机/不同产品的垛型不同（4 层 6 列、6 层 4 列、底层 6 上层 4……）
//   - PLC 点表里只有 Layers 一个总数，给不出"每层几个位置"
//   - 现场改垛型不应该改代码重新发版
//
// 与 LoadingDoc / HeadMapsDoc 同一套模式：JSON 文档、中心编辑下发、边端校验后适用。
type PalletLayoutDoc struct {
	// Code 是布局编号，用于多套垛型共存（如按产品/机台区分）
	Code string `yaml:"code" json:"code"`
	// Layers 按"从下往上"排列。每层位置数可以不同：
	// 现场确实存在底层放满、上层收窄的垛型。
	Layers []PalletLayerDoc `yaml:"layers" json:"layers"`
	// BobbinsPerPosition 是每个位置放几锭，现场常见为 2。
	BobbinsPerPosition int `yaml:"bobbins_per_position" json:"bobbins_per_position"`
}

// PalletLayerDoc 是栈板的一层。
type PalletLayerDoc struct {
	// Layer 是层号，1-based，从底层开始。
	Layer int `yaml:"layer" json:"layer"`
	// Positions 是这一层的位置数（列数）。
	Positions int `yaml:"positions" json:"positions"`
}

// PalletLayoutsDoc 是多套垛型的集合。
//
// 现场通常只有一种垛型，但换产品/换包装机时会变，
// 所以按 code 存放多套，由配置或点表指定当前用哪一套。
type PalletLayoutsDoc struct {
	// Layouts 是全部垛型。
	Layouts []PalletLayoutDoc `yaml:"layouts" json:"layouts"`
	// Default 是默认垛型编号；为空时取 Layouts 的第一项。
	Default string `yaml:"default" json:"default,omitempty"`
}

// BuildLoadingSequences 由文档构造装载映射，并做完整校验。
//
// source 仅用于错误信息（文件名或"中心下发配置 v3"之类）。
// 校验规则与遗留系统一致：A、B 两列合并后必须是 1..capacity 的完整排列。
func BuildLoadingSequences(doc *LoadingDoc, source string) (*LoadingSequences, error) {
	if doc == nil {
		return nil, fmt.Errorf("geometry: %s: empty loading document", source)
	}
	if doc.Capacity <= 0 {
		return nil, fmt.Errorf("geometry: %s: invalid capacity %d", source, doc.Capacity)
	}

	ls := &LoadingSequences{
		Capacity: doc.Capacity,
		Patterns: make(map[string]*LoadingPattern, len(doc.Patterns)),
	}

	for _, p := range doc.Patterns {
		if p.Code == "" {
			return nil, fmt.Errorf("geometry: %s: empty pattern code", source)
		}
		if _, dup := ls.Patterns[p.Code]; dup {
			return nil, fmt.Errorf("geometry: %s: duplicate pattern %q", source, p.Code)
		}
		pattern := &LoadingPattern{
			Code:  p.Code,
			Map:   make([]LoadingRow, 0, len(p.Map)),
			index: make(map[int]int, len(p.Map)),
		}
		for _, r := range p.Map {
			if _, dup := pattern.index[r.SpindlePlace]; dup {
				return nil, fmt.Errorf("geometry: %s pattern %q: duplicate spindle_place %d",
					source, p.Code, r.SpindlePlace)
			}
			pattern.index[r.SpindlePlace] = len(pattern.Map)
			pattern.Map = append(pattern.Map, LoadingRow{
				SpindlePlace: r.SpindlePlace,
				A:            r.A,
				B:            r.B,
			})
		}
		if problems := ValidatePattern(pattern, ls.Capacity); len(problems) > 0 {
			return nil, fmt.Errorf("geometry: %s pattern %q invalid: %v", source, p.Code, problems)
		}
		ls.Patterns[p.Code] = pattern
	}

	if len(ls.Patterns) == 0 {
		return nil, fmt.Errorf("geometry: %s: no patterns", source)
	}
	return ls, nil
}

// BuildHeadMaps 由文档构造打标映射，并做完整校验。
func BuildHeadMaps(doc *HeadMapsDoc, source string) (*HeadMaps, error) {
	if doc == nil {
		return nil, fmt.Errorf("geometry: %s: empty head map document", source)
	}

	hm := &HeadMaps{ByHeadCount: make(map[int]*HeadMap, len(doc.HeadMaps))}
	for _, m := range doc.HeadMaps {
		if _, dup := hm.ByHeadCount[m.HeadCount]; dup {
			return nil, fmt.Errorf("geometry: %s: duplicate head_count %d", source, m.HeadCount)
		}
		headMap := &HeadMap{
			HeadCount:    m.HeadCount,
			StepsPerHead: m.StepsPerHead,
			Capacity:     m.Capacity,
			Entries:      make([]HeadEntry, 0, len(m.Entries)),
			lookup:       make(map[headKey]int, len(m.Entries)),
		}
		for _, e := range m.Entries {
			key := headKey{head: e.Head, step: e.Step}
			if _, dup := headMap.lookup[key]; dup {
				return nil, fmt.Errorf("geometry: %s: head %d step %d duplicated",
					source, e.Head, e.Step)
			}
			headMap.lookup[key] = e.Position
			headMap.Entries = append(headMap.Entries, HeadEntry{
				Head: e.Head, Step: e.Step, Position: e.Position,
			})
		}
		if problems := ValidateHeadMap(headMap); len(problems) > 0 {
			return nil, fmt.Errorf("geometry: %s head map %d invalid: %v",
				source, m.HeadCount, problems)
		}
		hm.ByHeadCount[m.HeadCount] = headMap
	}

	if len(hm.ByHeadCount) == 0 {
		return nil, fmt.Errorf("geometry: %s: no head maps", source)
	}
	return hm, nil
}

// LoadingDocOf 把已构造的装载映射导出为文档，供前端回显与另存。
func (l *LoadingSequences) LoadingDocOf() *LoadingDoc {
	if l == nil {
		return nil
	}
	doc := &LoadingDoc{Capacity: l.Capacity}
	for _, code := range l.PatternCodes() {
		p := l.Patterns[code]
		pd := LoadingPatternDoc{Code: p.Code, Map: make([]LoadingRowDoc, 0, len(p.Map))}
		for _, r := range p.Map {
			pd.Map = append(pd.Map, LoadingRowDoc{
				SpindlePlace: r.SpindlePlace, A: r.A, B: r.B,
			})
		}
		doc.Patterns = append(doc.Patterns, pd)
	}
	return doc
}

// HeadMapsDocOf 把已构造的打标映射导出为文档。
func (h *HeadMaps) HeadMapsDocOf() *HeadMapsDoc {
	if h == nil {
		return nil
	}
	doc := &HeadMapsDoc{}
	for _, n := range h.HeadCounts() {
		m := h.ByHeadCount[n]
		md := HeadMapDoc{
			HeadCount:    m.HeadCount,
			StepsPerHead: m.StepsPerHead,
			Capacity:     m.Capacity,
			Entries:      make([]HeadEntryDoc, 0, len(m.Entries)),
		}
		for _, e := range m.Entries {
			md.Entries = append(md.Entries, HeadEntryDoc{
				Head: e.Head, Step: e.Step, Position: e.Position,
			})
		}
		doc.HeadMaps = append(doc.HeadMaps, md)
	}
	return doc
}
