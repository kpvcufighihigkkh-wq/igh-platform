package main

import (
	"testing"

	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/geometry"
	"github.com/igh-platform/igh-platform/internal/shared/siteconfig"
)

// TestMergeDocument_OnlyOverridesGivenSections 是导入工具最关键的约束：
//
// 保存接口是"整份文档替换"，如果只带 -loading 却提交整份文档，
// 会把边端已有点表整段清空，而导入命令本身还是成功的——
// 现场设备会凭空消失，且从命令输出完全看不出来。
func TestMergeDocument_OnlyOverridesGivenSections(t *testing.T) {
	base := &siteconfig.Document{
		Version: 7,
		PLC:     []config.PLCDevice{{Code: "keep-me", Host: "192.168.5.101", Port: 102}},
		Loading: &geometry.LoadingDoc{Capacity: 24},
		Heads:   &geometry.HeadMapsDoc{HeadMaps: []geometry.HeadMapDoc{{HeadCount: 6}}},
	}
	patch := &siteconfig.Document{
		PLC:     []config.PLCDevice{{Code: "new-plc", Host: "192.168.5.111", Port: 102}},
		Loading: &geometry.LoadingDoc{Capacity: 48},
		Heads:   &geometry.HeadMapsDoc{HeadMaps: []geometry.HeadMapDoc{{HeadCount: 12}}},
		Remark:  "只换装载表",
	}

	// 场景：只导入装载表
	opt := options{loadFile: "loading_sequences.yaml"}
	mergeDocument(base, patch, opt)

	if len(base.PLC) != 1 || base.PLC[0].Code != "keep-me" {
		t.Errorf("点表被覆盖了：%+v（只导装载表时点表必须原样保留）", base.PLC)
	}
	if base.Loading == nil || base.Loading.Capacity != 48 {
		t.Errorf("装载表未被覆盖：%+v", base.Loading)
	}
	if base.Heads == nil || base.Heads.HeadMaps[0].HeadCount != 6 {
		t.Errorf("锭位映射被覆盖了：%+v", base.Heads)
	}
	if base.Remark != "只换装载表" {
		t.Errorf("remark = %q", base.Remark)
	}
}

// TestMergeDocument_ReplacesAllGivenSections 验证三个部分都给时会全部替换。
func TestMergeDocument_ReplacesAllGivenSections(t *testing.T) {
	base := &siteconfig.Document{
		PLC:     []config.PLCDevice{{Code: "old"}},
		Loading: &geometry.LoadingDoc{Capacity: 24},
		Heads:   &geometry.HeadMapsDoc{HeadMaps: []geometry.HeadMapDoc{{HeadCount: 6}}},
	}
	patch := &siteconfig.Document{
		PLC:     []config.PLCDevice{{Code: "new"}},
		Loading: &geometry.LoadingDoc{Capacity: 48},
		Heads:   &geometry.HeadMapsDoc{HeadMaps: []geometry.HeadMapDoc{{HeadCount: 16}}},
	}

	mergeDocument(base, patch, options{
		plcFile: "a.yaml", loadFile: "b.yaml", headFile: "c.yaml",
	})

	if base.PLC[0].Code != "new" {
		t.Errorf("点表未替换：%+v", base.PLC)
	}
	if base.Loading.Capacity != 48 {
		t.Errorf("装载表未替换：%+v", base.Loading)
	}
	if base.Heads.HeadMaps[0].HeadCount != 16 {
		t.Errorf("锭位映射未替换：%+v", base.Heads)
	}
}

// TestMergeDocument_KeepsVersion 验证合并不改动版本号（由服务端递增）。
func TestMergeDocument_KeepsVersion(t *testing.T) {
	base := &siteconfig.Document{Version: 7}
	patch := &siteconfig.Document{Version: 99}

	mergeDocument(base, patch, options{loadFile: "x.yaml"})

	if base.Version != 7 {
		t.Errorf("version = %d, want 7（版本号只能由服务端递增）", base.Version)
	}
}

func TestChangedSections(t *testing.T) {
	cases := []struct {
		opt  options
		want string
	}{
		{options{loadFile: "a"}, "装载位置查表"},
		{options{plcFile: "a"}, "PLC 点表"},
		{options{plcFile: "a", headFile: "b"}, "PLC 点表、外检锭位映射"},
		{options{}, ""},
	}
	for _, tc := range cases {
		if got := changedSections(tc.opt); got != tc.want {
			t.Errorf("changedSections(%+v) = %q, want %q", tc.opt, got, tc.want)
		}
	}
}
