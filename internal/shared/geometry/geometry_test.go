package geometry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// golden 是 tools/geomgen/extract.js 从遗留系统直接生成的原始数据快照。
// 这个测试的作用：确保 Go 侧加载并解释后的结果，与遗留系统的原始数据**逐条一致**。
//
// 如果这个测试挂了，说明以下之一：
//   - YAML 被手工改动过（不允许）
//   - 加载器解释错了（真 bug）
//   - 遗留系统的几何表确实变了（需要重新生成并评审）
type goldenDoc struct {
	LoadingCapacity int `json:"loadingCapacity"`
	Patterns        []struct {
		Code string `json:"code"`
		Map  []struct {
			SpindlePlace int `json:"spindlePlace"`
			A            int `json:"a"`
			B            int `json:"b"`
		} `json:"map"`
	} `json:"patterns"`
	HeadMaps []struct {
		HeadCount    int `json:"headCount"`
		StepsPerHead int `json:"stepsPerHead"`
		Capacity     int `json:"capacity"`
		Entries      []struct {
			Head     int `json:"head"`
			Step     int `json:"step"`
			Position int `json:"position"`
		} `json:"entries"`
	} `json:"headMaps"`
}

func geometryDir() string {
	return filepath.Join("..", "..", "..", "configs", "geometry")
}

func loadGolden(t *testing.T) *goldenDoc {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(geometryDir(), "legacy_golden.json"))
	if err != nil {
		t.Fatalf("读取 legacy_golden.json 失败: %v", err)
	}
	var doc goldenDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("解析 legacy_golden.json 失败: %v", err)
	}
	return &doc
}

func TestLoadGeometry(t *testing.T) {
	ls, hm, err := Load(geometryDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ls.Capacity != 24 {
		t.Errorf("loading capacity = %d, want 24", ls.Capacity)
	}
	if len(ls.Patterns) != 5 {
		t.Errorf("pattern count = %d, want 5", len(ls.Patterns))
	}
	if len(hm.ByHeadCount) != 3 {
		t.Errorf("head map count = %d, want 3", len(hm.ByHeadCount))
	}
}

// TestGeometryMatchesLegacyGolden 是这一步的核心验证：
// Go 侧每一行、每一条映射都必须与遗留系统提取出来的原始数据完全相等。
func TestGeometryMatchesLegacyGolden(t *testing.T) {
	golden := loadGolden(t)
	ls, hm, err := Load(geometryDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// ---- 装载映射 ----
	if ls.Capacity != golden.LoadingCapacity {
		t.Errorf("capacity = %d, golden = %d", ls.Capacity, golden.LoadingCapacity)
	}
	if len(ls.Patterns) != len(golden.Patterns) {
		t.Fatalf("pattern count = %d, golden = %d", len(ls.Patterns), len(golden.Patterns))
	}
	for _, gp := range golden.Patterns {
		p, ok := ls.Patterns[gp.Code]
		if !ok {
			t.Errorf("missing pattern %q", gp.Code)
			continue
		}
		if len(p.Map) != len(gp.Map) {
			t.Errorf("pattern %q row count = %d, golden = %d",
				gp.Code, len(p.Map), len(gp.Map))
			continue
		}
		for i, gr := range gp.Map {
			got := p.Map[i]
			if got.SpindlePlace != gr.SpindlePlace || got.A != gr.A || got.B != gr.B {
				t.Errorf("pattern %q row %d = %+v, golden = {spindle:%d a:%d b:%d}",
					gp.Code, i, got, gr.SpindlePlace, gr.A, gr.B)
			}
		}
	}

	// ---- 打标头映射 ----
	if len(hm.ByHeadCount) != len(golden.HeadMaps) {
		t.Fatalf("head map count = %d, golden = %d",
			len(hm.ByHeadCount), len(golden.HeadMaps))
	}
	for _, gm := range golden.HeadMaps {
		m, ok := hm.ByHeadCount[gm.HeadCount]
		if !ok {
			t.Errorf("missing head map for %d heads", gm.HeadCount)
			continue
		}
		if m.StepsPerHead != gm.StepsPerHead {
			t.Errorf("%d heads: steps_per_head = %d, golden = %d",
				gm.HeadCount, m.StepsPerHead, gm.StepsPerHead)
		}
		if m.Capacity != gm.Capacity {
			t.Errorf("%d heads: capacity = %d, golden = %d",
				gm.HeadCount, m.Capacity, gm.Capacity)
		}
		if len(m.Entries) != len(gm.Entries) {
			t.Errorf("%d heads: entry count = %d, golden = %d",
				gm.HeadCount, len(m.Entries), len(gm.Entries))
			continue
		}
		for i, ge := range gm.Entries {
			got := m.Entries[i]
			if got.Head != ge.Head || got.Step != ge.Step || got.Position != ge.Position {
				t.Errorf("%d heads entry %d = %+v, golden = {head:%d step:%d position:%d}",
					gm.HeadCount, i, got, ge.Head, ge.Step, ge.Position)
			}
		}
	}

	// ---- 语义查询：与 golden 逐点比对 ----
	for _, gp := range golden.Patterns {
		for _, gr := range gp.Map {
			gotA, err := ls.Place(gp.Code, gr.SpindlePlace, SpindleA)
			if err != nil {
				t.Errorf("Place(%s, %d, A): %v", gp.Code, gr.SpindlePlace, err)
			} else if gotA != gr.A {
				t.Errorf("Place(%s, %d, A) = %d, want %d", gp.Code, gr.SpindlePlace, gotA, gr.A)
			}
			gotB, err := ls.Place(gp.Code, gr.SpindlePlace, SpindleB)
			if err != nil {
				t.Errorf("Place(%s, %d, B): %v", gp.Code, gr.SpindlePlace, err)
			} else if gotB != gr.B {
				t.Errorf("Place(%s, %d, B) = %d, want %d", gp.Code, gr.SpindlePlace, gotB, gr.B)
			}
		}
	}
	for _, gm := range golden.HeadMaps {
		for _, ge := range gm.Entries {
			got, err := hm.Position(gm.HeadCount, ge.Head, ge.Step)
			if err != nil {
				t.Errorf("Position(%d, %d, %d): %v", gm.HeadCount, ge.Head, ge.Step, err)
			} else if got != ge.Position {
				t.Errorf("Position(%d, %d, %d) = %d, want %d",
					gm.HeadCount, ge.Head, ge.Step, got, ge.Position)
			}
		}
	}
}

// TestPlaceRejectsUnknown 验证：查不到时返回错误，而不是像遗留系统那样静默返回 0。
func TestPlaceRejectsUnknown(t *testing.T) {
	ls, _, err := Load(geometryDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	cases := []struct {
		name    string
		pattern string
		place   int
		side    SpindleSide
	}{
		{"unknown pattern", "nope", 1, SpindleA},
		{"place out of range", "sx", 999, SpindleA},
		{"invalid side", "sx", 1, SpindleSide('Z')},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ls.Place(tc.pattern, tc.place, tc.side); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

// TestHeadMapRejectsOutOfRange 验证越界的头号/段号返回错误。
func TestHeadMapRejectsOutOfRange(t *testing.T) {
	_, hm, err := Load(geometryDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	cases := []struct {
		name      string
		headCount int
		head      int
		step      int
	}{
		{"unknown head count", 8, 1, 1},
		{"head too large", 16, 17, 1},
		{"head zero", 16, 0, 1},
		{"step too large", 16, 1, 3},
		{"step zero", 16, 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := hm.Position(tc.headCount, tc.head, tc.step); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

// TestLoadingPatternIsPermutation 显式验证每套装载模式都是 1..capacity 的排列。
func TestLoadingPatternIsPermutation(t *testing.T) {
	ls, _, err := Load(geometryDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, code := range ls.PatternCodes() {
		t.Run(code, func(t *testing.T) {
			if problems := ValidatePattern(ls.Patterns[code], ls.Capacity); len(problems) > 0 {
				t.Fatalf("invalid: %v", problems)
			}
		})
	}
}

// TestHeadMapsArePermutations 显式验证每套头映射都是 1..capacity 的排列。
func TestHeadMapsArePermutations(t *testing.T) {
	_, hm, err := Load(geometryDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, n := range hm.HeadCounts() {
		t.Run(string(rune('0'+n/10))+string(rune('0'+n%10))+"heads", func(t *testing.T) {
			if problems := ValidateHeadMap(hm.ByHeadCount[n]); len(problems) > 0 {
				t.Fatalf("invalid: %v", problems)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 包装栈板垛型
// ---------------------------------------------------------------------------

// TestBuildPalletLayout 覆盖常见的 6×4 垛型（6 层、每层 4 位、每位 2 锭 = 48 锭）。
func TestBuildPalletLayout(t *testing.T) {
	doc := &PalletLayoutsDoc{
		Layouts: []PalletLayoutDoc{{
			Code:               "std48",
			BobbinsPerPosition: 2,
			Layers: []PalletLayerDoc{
				{Layer: 1, Positions: 4},
				{Layer: 2, Positions: 4},
				{Layer: 3, Positions: 4},
				{Layer: 4, Positions: 4},
				{Layer: 5, Positions: 4},
				{Layer: 6, Positions: 4},
			},
		}},
	}
	layouts, err := BuildPalletLayouts(doc, "test")
	if err != nil {
		t.Fatalf("BuildPalletLayouts: %v", err)
	}
	l, err := layouts.Get("std48")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if l.Capacity != 48 {
		t.Errorf("Capacity = %d, want 48", l.Capacity)
	}
	if got := l.TotalBobbins(); got != 48 {
		t.Errorf("TotalBobbins = %d, want 48", got)
	}
}

// TestPalletLayout_Locate 验证"第 n 锭 → 层/位"的换算。
//
// 这是页面显示"当前装到第几层第几位"的唯一依据，
// 算错会让工人装错位置，属于会直接造成返工的错误。
func TestPalletLayout_Locate(t *testing.T) {
	// 每层 4 位、每位 2 锭：第 1 层是第 1-8 锭，第 2 层是第 9-16 锭
	doc := &PalletLayoutsDoc{Layouts: []PalletLayoutDoc{{
		Code: "layered", BobbinsPerPosition: 2,
		Layers: []PalletLayerDoc{
			{Layer: 1, Positions: 4},
			{Layer: 2, Positions: 4},
			{Layer: 3, Positions: 6}, // 层间位置数不同也要能算
		},
	}}}
	layouts, err := BuildPalletLayouts(doc, "test")
	if err != nil {
		t.Fatalf("BuildPalletLayouts: %v", err)
	}
	l, _ := layouts.Get("layered")

	cases := []struct {
		bobbin, wantLayer, wantPos int
	}{
		{1, 1, 1},
		{2, 1, 1},
		{3, 1, 2},
		{8, 1, 4},
		{9, 2, 1},
		{16, 2, 4},
		{17, 3, 1},
		{28, 3, 6},
	}
	for _, tc := range cases {
		layer, pos, err := l.Locate(tc.bobbin)
		if err != nil {
			t.Fatalf("Locate(%d): %v", tc.bobbin, err)
		}
		if layer != tc.wantLayer || pos != tc.wantPos {
			t.Errorf("Locate(%d) = 第%d层第%d位, want 第%d层第%d位",
				tc.bobbin, layer, pos, tc.wantLayer, tc.wantPos)
		}
	}

	// 越界必须报错，不能静默夹到边界：
	// PLC 给的值和配置的垛型对不上时，夹边会让工人以为一切正常。
	for _, bad := range []int{0, -1, 29} {
		if _, _, err := l.Locate(bad); err == nil {
			t.Errorf("Locate(%d) 应当报错", bad)
		}
	}
}

// TestPalletLayout_RejectsInvalid 覆盖各类配置错误。
func TestPalletLayout_RejectsInvalid(t *testing.T) {
	cases := []struct {
		name string
		doc  PalletLayoutDoc
	}{
		{"缺 code", PalletLayoutDoc{BobbinsPerPosition: 2,
			Layers: []PalletLayerDoc{{Layer: 1, Positions: 4}}}},
		{"每位置锭数为 0", PalletLayoutDoc{Code: "x", BobbinsPerPosition: 0,
			Layers: []PalletLayerDoc{{Layer: 1, Positions: 4}}}},
		{"没有层", PalletLayoutDoc{Code: "x", BobbinsPerPosition: 2}},
		{"层号不从 1 开始", PalletLayoutDoc{Code: "x", BobbinsPerPosition: 2,
			Layers: []PalletLayerDoc{{Layer: 2, Positions: 4}}}},
		{"层号跳号", PalletLayoutDoc{Code: "x", BobbinsPerPosition: 2,
			Layers: []PalletLayerDoc{{Layer: 1, Positions: 4}, {Layer: 3, Positions: 4}}}},
		{"某层位置数为 0", PalletLayoutDoc{Code: "x", BobbinsPerPosition: 2,
			Layers: []PalletLayerDoc{{Layer: 1, Positions: 0}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildPalletLayouts(&PalletLayoutsDoc{Layouts: []PalletLayoutDoc{tc.doc}}, "test")
			if err == nil {
				t.Fatal("应当报错")
			}
		})
	}

	t.Run("重复 code", func(t *testing.T) {
		one := PalletLayoutDoc{Code: "dup", BobbinsPerPosition: 2,
			Layers: []PalletLayerDoc{{Layer: 1, Positions: 4}}}
		_, err := BuildPalletLayouts(&PalletLayoutsDoc{
			Layouts: []PalletLayoutDoc{one, one},
		}, "test")
		if err == nil {
			t.Fatal("重复 code 应当报错")
		}
	})
}

// TestPalletLayout_UnsortedInput 验证输入顺序无关。
//
// 现场在页面上上下移动行是常事，要求"必须按层号顺序填"只会制造无谓错误。
func TestPalletLayout_UnsortedInput(t *testing.T) {
	doc := &PalletLayoutsDoc{Layouts: []PalletLayoutDoc{{
		Code: "unsorted", BobbinsPerPosition: 2,
		Layers: []PalletLayerDoc{
			{Layer: 3, Positions: 2},
			{Layer: 1, Positions: 4},
			{Layer: 2, Positions: 4},
		},
	}}}
	layouts, err := BuildPalletLayouts(doc, "test")
	if err != nil {
		t.Fatalf("BuildPalletLayouts: %v", err)
	}
	l, _ := layouts.Get("unsorted")
	if l.Layers[0].Layer != 1 || l.Layers[2].Layer != 3 {
		t.Fatalf("层未按层号排序: %+v", l.Layers)
	}
	if l.Capacity != 20 {
		t.Errorf("Capacity = %d, want 20（4+4+2 位 × 2 锭）", l.Capacity)
	}
	// 第 17 锭应落在第 3 层第 1 位
	layer, pos, err := l.Locate(17)
	if err != nil {
		t.Fatalf("Locate(17): %v", err)
	}
	if layer != 3 || pos != 1 {
		t.Errorf("Locate(17) = 第%d层第%d位, want 第3层第1位", layer, pos)
	}
}
