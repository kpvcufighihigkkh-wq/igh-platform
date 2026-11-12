package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// sortingGolden 是 tools/plcgen/sorting.js 从旧 UDT 提取的原始布局。
type sortingGolden struct {
	Source  string `json:"source"`
	Devices []struct {
		Code         string `json:"code"`
		Heads        int    `json:"heads"`
		StepsPerHead int    `json:"stepsPerHead"`
		DB           int    `json:"db"`
		Layout       struct {
			Start, IdModule, NumberModule, Step int
			Enabled, Alarm, Done, ModuleRead    int
			Size                                int
		} `json:"layout"`
		UdtLayout struct {
			Start, IdModule, NumberModule, Step int
			Enabled, Alarm, Done, ModuleRead    int
			Size                                int
		} `json:"udtLayout"`
	} `json:"devices"`
}

// TestSortingLaserMatchesGolden 是外检打标点表的契约测试。
//
// 打标握手一旦地址错位，现场会打出错误标签或干脆不打印，
// 而且很难从现象反推原因。所以这里逐设备、逐分组比对偏移。
func TestSortingLaserMatchesGolden(t *testing.T) {
	dir := instancesDir()

	raw, err := os.ReadFile(filepath.Join(dir, "sorting_laser_golden.json"))
	if err != nil {
		t.Fatalf("读取 golden 失败: %v", err)
	}
	var golden sortingGolden
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatalf("解析 golden 失败: %v", err)
	}
	if len(golden.Devices) == 0 {
		t.Fatal("golden 中没有设备")
	}

	doc, err := LoadPLCInstances(filepath.Join(dir, "sorting_laser.yaml"))
	if err != nil {
		t.Fatalf("加载点表失败: %v", err)
	}
	if len(doc.Devices) != len(golden.Devices) {
		t.Fatalf("设备数 = %d, golden = %d", len(doc.Devices), len(golden.Devices))
	}

	for i, gd := range golden.Devices {
		d := doc.Devices[i]
		if d.Code != gd.Code {
			t.Errorf("设备 %d code = %q, want %q", i, d.Code, gd.Code)
			continue
		}

		want := map[string]int{
			"laser_start":   gd.UdtLayout.Start,
			"laser_ctrl":    gd.UdtLayout.IdModule,
			"laser_enabled": gd.UdtLayout.Enabled,
			"laser_alarm":   gd.UdtLayout.Alarm,
			"laser_done":    gd.UdtLayout.Done,
			"laser_read":    gd.UdtLayout.ModuleRead,
		}

		got := make(map[string]int, len(d.PollGroups))
		for _, g := range d.PollGroups {
			got[g.Name] = g.Start
			if g.DB != gd.DB {
				t.Errorf("%s/%s db = %d, want %d", d.Code, g.Name, g.DB, gd.DB)
			}
		}

		for name, wantStart := range want {
			start, ok := got[name]
			if !ok {
				t.Errorf("%s 缺少分组 %s", d.Code, name)
				continue
			}
			if start != wantStart {
				t.Errorf("%s/%s start = %d, want %d", d.Code, name, start, wantStart)
			}
		}

		// 数组分组的元素数必须等于头数
		for _, g := range d.PollGroups {
			if g.Repeat == nil {
				continue
			}
			if g.Repeat.Count != gd.Heads {
				t.Errorf("%s/%s repeat.count = %d, want %d",
					d.Code, g.Name, g.Repeat.Count, gd.Heads)
			}
			if g.Repeat.Stride != 2 {
				t.Errorf("%s/%s stride = %d, want 2", d.Code, g.Name, g.Repeat.Stride)
			}
		}
	}
}
