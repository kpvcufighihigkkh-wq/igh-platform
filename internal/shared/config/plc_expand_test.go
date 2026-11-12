package config

import (
	"path/filepath"
	"testing"
)

// TestExpandPoints_Twin 验证双胞胎布局展开：机位 × 两侧 × 字段。
func TestExpandPoints_Twin(t *testing.T) {
	doc, err := LoadPLCInstances(filepath.Join(instancesDir(), "spinning_sides.yaml"))
	if err != nil {
		t.Fatalf("LoadPLCInstances: %v", err)
	}

	// winder-L01: count=24, stride=198, 6 个字段, 两侧
	d := doc.Devices[0]
	pts := ExpandPoints(&d)

	wantCount := 24 * 2 * 6
	if len(pts) != wantCount {
		t.Fatalf("expanded points = %d, want %d", len(pts), wantCount)
	}

	// 校验几个关键地址
	find := func(winder int, side, field string) PointAddress {
		for _, p := range pts {
			if p.Winder == winder && p.Side == side && p.Field == field {
				return p
			}
		}
		t.Fatalf("missing %d/%s/%s", winder, side, field)
		return PointAddress{}
	}

	// winder1 left POSITION_Name: 0*198 + 96 + 0 = 96
	if got := find(1, "left", "DatiBobine_DaWinder.POSITION_Name").Offset; got != 96 {
		t.Errorf("winder1 left POSITION_Name offset = %d, want 96", got)
	}
	// winder1 right POSITION_Name: 0*198 + 4 + 0 = 4
	if got := find(1, "right", "DatiBobine_DaWinder.POSITION_Name").Offset; got != 4 {
		t.Errorf("winder1 right POSITION_Name offset = %d, want 4", got)
	}
	// winder1 left CodeNumber: 96 + 79 = 175
	if got := find(1, "left", "DatiBobine_DaWinder.CodeNumber").Offset; got != 175 {
		t.Errorf("winder1 left CodeNumber offset = %d, want 175", got)
	}
	// winder2 left POSITION_Name: 198 + 96 = 294
	if got := find(2, "left", "DatiBobine_DaWinder.POSITION_Name").Offset; got != 294 {
		t.Errorf("winder2 left POSITION_Name offset = %d, want 294", got)
	}
	// winder24 right CodeNumber: 23*198 + 4 + 79 = 4637
	if got := find(24, "right", "DatiBobine_DaWinder.CodeNumber").Offset; got != 23*198+4+79 {
		t.Errorf("winder24 right CodeNumber offset = %d, want %d", got, 23*198+4+79)
	}

	// 所有地址必须落在读取块长度内
	blockLen := d.PollGroups[0].BlockLength()
	for _, p := range pts {
		if p.Offset < 0 || p.Offset+fieldWidth(p) > blockLen {
			t.Fatalf("%s exceeds block length %d", p.Key(), blockLen)
		}
	}
}

// TestExpandPoints_KeyStable 验证 Key 唯一且稳定（用于变更检测）。
func TestExpandPoints_KeyStable(t *testing.T) {
	doc, err := LoadPLCInstances(filepath.Join(instancesDir(), "spinning_sides.yaml"))
	if err != nil {
		t.Fatalf("LoadPLCInstances: %v", err)
	}
	d := doc.Devices[0]

	seen := make(map[string]bool)
	for _, p := range ExpandPoints(&d) {
		k := p.Key()
		if seen[k] {
			t.Fatalf("duplicate key: %s", k)
		}
		seen[k] = true
	}
	if len(seen) != 24*2*6 {
		t.Fatalf("unique keys = %d, want %d", len(seen), 24*2*6)
	}
}

// TestExpandPoints_NonTwin 验证非双胞胎布局（只有字段，无重复）。
func TestExpandPoints_NonTwin(t *testing.T) {
	d := PLCDevice{
		Code: "plain",
		Host: "127.0.0.1",
		Port: 102,
		PollGroups: []PLCPollGroup{{
			Name:   "g",
			DB:     100,
			Length: 20,
			Points: []PLCPoint{
				{Name: "a", Offset: 0, Type: "int16"},
				{Name: "b", Offset: 4, Type: "int32"},
			},
		}},
	}
	pts := ExpandPoints(&d)
	if len(pts) != 2 {
		t.Fatalf("points = %d, want 2", len(pts))
	}
	if pts[0].Winder != 0 || pts[0].Side != "" {
		t.Fatalf("non-twin point should have no winder/side: %+v", pts[0])
	}
}

func fieldWidth(p PointAddress) int {
	switch p.Type {
	case "bool", "byte", "uint8", "int8":
		return 1
	case "int16", "uint16":
		return 2
	case "int32", "uint32", "float32":
		return 4
	case "float64":
		return 8
	case "string":
		return p.Length
	default:
		return 0
	}
}
