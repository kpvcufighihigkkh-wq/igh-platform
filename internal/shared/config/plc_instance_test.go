package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// plcGolden 是 tools/plcgen/extract.js 从 h028.sql + UDT 生成的原始快照。
// 这个测试保证：Go 侧加载并解释后的点表，与生成器产出的数据逐字段一致。
//
// 如果这里挂了，说明：
//   - YAML 被手工改过（不允许）
//   - 加载/校验逻辑有 bug
//   - 生成器重新生成后数据确实变了（需要评审）
type plcGoldenDoc struct {
	Source  string `json:"source"`
	Devices []struct {
		ID       int    `json:"id"`
		Code     string `json:"code"`
		Host     string `json:"host"`
		WinderDB int    `json:"winderDb"`
		UdtSize  int    `json:"udtSize"`
		Winder   struct {
			Schema        string `json:"schema"`
			Count         int    `json:"count"`
			UdtSize       int    `json:"udtSize"`
			Stride        int    `json:"stride"`
			OffsetEven    int    `json:"offsetEven"`
			OffsetEndEven int    `json:"offsetEndEven"`
			OffsetOdd     int    `json:"offsetOdd"`
			OffsetEndOdd  int    `json:"offsetEndOdd"`
			Sides *struct {
				Left  int `json:"left"`
				Right int `json:"right"`
			} `json:"sides"`
		} `json:"winder"`
		Fields []struct {
			Name   string `json:"name"`
			Offset int    `json:"offset"`
			Length int    `json:"length"`
		} `json:"fields"`
	} `json:"devices"`
}

func instancesDir() string {
	return filepath.Join("..", "..", "..", "configs", "devices", "instances")
}

func loadPLCGolden(t *testing.T) *plcGoldenDoc {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(instancesDir(), "plc_golden.json"))
	if err != nil {
		t.Fatalf("读取 plc_golden.json 失败: %v", err)
	}
	var doc plcGoldenDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("解析 plc_golden.json 失败: %v", err)
	}
	return &doc
}

func TestLoadPLCInstances(t *testing.T) {
	doc, err := LoadPLCInstances(filepath.Join(instancesDir(), "spinning_sides.yaml"))
	if err != nil {
		t.Fatalf("LoadPLCInstances: %v", err)
	}
	if len(doc.Devices) != 8 {
		t.Fatalf("device count = %d, want 8", len(doc.Devices))
	}
}

// TestPLCInstancesMatchGolden 是核心契约测试：逐设备、逐字段比对。
func TestPLCInstancesMatchGolden(t *testing.T) {
	golden := loadPLCGolden(t)
	doc, err := LoadPLCInstances(filepath.Join(instancesDir(), "spinning_sides.yaml"))
	if err != nil {
		t.Fatalf("LoadPLCInstances: %v", err)
	}

	if len(doc.Devices) != len(golden.Devices) {
		t.Fatalf("device count = %d, golden = %d", len(doc.Devices), len(golden.Devices))
	}

	for i, gd := range golden.Devices {
		d := doc.Devices[i]
		if d.Code != gd.Code {
			t.Errorf("device %d code = %q, want %q", i, d.Code, gd.Code)
		}
		if d.Host != gd.Host {
			t.Errorf("%s host = %q, want %q", d.Code, d.Host, gd.Host)
		}
		if d.Twin == nil {
			t.Errorf("%s: twin is nil", d.Code)
			continue
		}
		tw := d.Twin
		if tw.UdtSize != gd.UdtSize {
			t.Errorf("%s udt_size = %d, want %d", d.Code, tw.UdtSize, gd.UdtSize)
		}
		if tw.Count != gd.Winder.Count {
			t.Errorf("%s count = %d, want %d", d.Code, tw.Count, gd.Winder.Count)
		}
		if tw.Stride != gd.Winder.Stride {
			t.Errorf("%s stride = %d, want %d", d.Code, tw.Stride, gd.Winder.Stride)
		}
		if tw.OffsetEven != gd.Winder.OffsetEven {
			t.Errorf("%s offset_even = %d, want %d", d.Code, tw.OffsetEven, gd.Winder.OffsetEven)
		}
		if tw.OffsetOdd != gd.Winder.OffsetOdd {
			t.Errorf("%s offset_odd = %d, want %d", d.Code, tw.OffsetOdd, gd.Winder.OffsetOdd)
		}
		// 单面锭位没有左右两侧：golden 里 sides 为 null，
		// 配置里也必须是 nil，否则同一段字节会被展开两遍、重复上报。
		if gd.Winder.Sides == nil {
			if tw.Sides != nil {
				t.Errorf("%s: single-sided line must not declare sides, got %+v", d.Code, tw.Sides)
			}
			if pg := d.PollGroups[0]; pg.Repeat != nil && pg.Repeat.Sides != nil {
				t.Errorf("%s: single-sided repeat must not declare sides", d.Code)
			}
		} else {
			left, right := tw.SidesOrDefault()
			if left != gd.Winder.Sides.Left {
				t.Errorf("%s left side = %d, want %d", d.Code, left, gd.Winder.Sides.Left)
			}
			if right != gd.Winder.Sides.Right {
				t.Errorf("%s right side = %d, want %d", d.Code, right, gd.Winder.Sides.Right)
			}
		}

		// 轮询组：DB 号与长度
		if len(d.PollGroups) != 1 {
			t.Errorf("%s: want exactly 1 poll group, got %d", d.Code, len(d.PollGroups))
			continue
		}
		pg := d.PollGroups[0]
		if pg.DB != gd.WinderDB {
			t.Errorf("%s db = %d, want %d", d.Code, pg.DB, gd.WinderDB)
		}
		if pg.Length != gd.Winder.Count*gd.Winder.Stride {
			t.Errorf("%s length = %d, want %d", d.Code, pg.Length, gd.Winder.Count*gd.Winder.Stride)
		}

		// 字段逐条比对
		if len(pg.Points) != len(gd.Fields) {
			t.Errorf("%s point count = %d, want %d", d.Code, len(pg.Points), len(gd.Fields))
			continue
		}
		for j, gf := range gd.Fields {
			p := pg.Points[j]
			if p.Name != gf.Name {
				t.Errorf("%s point %d name = %q, want %q", d.Code, j, p.Name, gf.Name)
			}
			if p.Offset != gf.Offset {
				t.Errorf("%s point %q offset = %d, want %d", d.Code, p.Name, p.Offset, gf.Offset)
			}
			w, err := pointWidth(p)
			if err != nil {
				t.Errorf("%s point %q: %v", d.Code, p.Name, err)
				continue
			}
			if w != gf.Length {
				t.Errorf("%s point %q width = %d, want %d", d.Code, p.Name, w, gf.Length)
			}
		}
	}
}

// TestTwinStrideInvariant 显式验证"两侧收敛"这条核心不变量。
func TestTwinStrideInvariant(t *testing.T) {
	doc, err := LoadPLCInstances(filepath.Join(instancesDir(), "spinning_sides.yaml"))
	if err != nil {
		t.Fatalf("LoadPLCInstances: %v", err)
	}
	for _, d := range doc.Devices {
		if d.Twin == nil {
			continue
		}
		t.Run(d.Code, func(t *testing.T) {
			tw := d.Twin
			evenEnd := tw.OffsetEven + tw.UdtSize + tw.OffsetEndEven
			oddEnd := tw.OffsetOdd + tw.UdtSize + tw.OffsetEndOdd
			if evenEnd != oddEnd {
				t.Fatalf("even end %d != odd end %d", evenEnd, oddEnd)
			}
			if tw.Stride != evenEnd {
				t.Fatalf("stride %d != %d", tw.Stride, evenEnd)
			}
		})
	}
}

// TestAddressOf 验证绝对地址计算：base + index*stride + side + fieldOffset。
func TestAddressOf(t *testing.T) {
	tw := &PLCTwinWinder{
		UdtSize: 92, Stride: 198, Count: 24,
		OffsetEven: 96, OffsetEndEven: 10,
		OffsetOdd: 4, OffsetEndOdd: 102,
		Sides: &PLCUdtSide{
			Left:  PLCSideOffset{Offset: 96},
			Right: PLCSideOffset{Offset: 4},
		},
	}
	cases := []struct {
		name  string
		index int
		left  bool
		field int
		want  int
	}{
		{"winder0 left POSITION_Name", 0, true, 0, 96},
		{"winder0 right POSITION_Name", 0, false, 0, 4},
		{"winder0 left CodeNumber", 0, true, 79, 175},
		{"winder1 left POSITION_Name", 1, true, 0, 294},
		{"winder1 right POSITION_Name", 1, false, 0, 202},
		{"winder23 right CodeNumber", 23, false, 79, 23*198 + 4 + 79},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tw.AddressOf(tc.index, tc.left, tc.field)
			if got != tc.want {
				t.Fatalf("AddressOf(%d,%v,%d) = %d, want %d",
					tc.index, tc.left, tc.field, got, tc.want)
			}
		})
	}
}

// TestRejectsBrokenTwin 验证：两侧不收敛时加载失败，而不是静默通过。
func TestRejectsBrokenTwin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.yaml")
	content := `
devices:
  - code: "broken"
    host: "127.0.0.1"
    port: 102
    twin:
      udt_size: 92
      stride: 250
      count: 24
      offset_even: 96
      offset_end_even: 10
      offset_odd: 4
      offset_end_odd: 102
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPLCInstances(path); err == nil {
		t.Fatal("expected error for non-converging twin offsets, got nil")
	}
}

// TestRejectsFieldOutsideUdt 验证：字段超出 UDT 大小时加载失败。
func TestRejectsFieldOutsideUdt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad_point.yaml")
	content := `
devices:
  - code: "bad"
    host: "127.0.0.1"
    port: 102
    twin:
      udt_size: 92
      stride: 198
      count: 1
      offset_even: 96
      offset_end_even: 10
      offset_odd: 4
      offset_end_odd: 102
    poll_groups:
      - name: "g"
        db: 555
        length: 198
        repeat: { count: 1, stride: 198 }
        points:
          - { name: "too_far", offset: 90, type: "int32" }
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPLCInstances(path); err == nil {
		t.Fatal("expected error for point outside UDT, got nil")
	}
}
