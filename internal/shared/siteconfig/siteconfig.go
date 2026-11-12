// Package siteconfig 定义"现场配置"——那些必须由现场人员按实际机械布局填写、
// 不能写死在代码或仓库里的数据。
//
// 包含三类：
//  1. PLC 点表   ：DB 号、块偏移、机位步长、字段字节布局
//  2. 装载位置查表：卷绕机锭位 + 落筒侧 → 容器位置
//  3. 外检锭位映射：打标头 + 段序 → 模块内锭位
//
// 设计约束：
//   - 同一份文档同时可在中心编辑、随心跳下发、在边端落盘缓存，
//     因此只依赖 JSON，不绑定任何存储格式
//   - 校验与构造完全分离于加载：中心保存前校验一次，边端应用前再校验一次
//   - 版本单调递增；边端只在人工确认后才把新版本切换为生效版本
package siteconfig

import (
	"fmt"

	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/geometry"
)

// Document 是一份完整的现场配置。
//
// 三个部分相互独立，可以只更新其中之一：
// 未填写的部分表示"该边端不使用这类配置"，不是"清空"。
type Document struct {
	// Version 是中心侧的单调递增版本号，用于边端判断是否需要更新
	Version int `json:"version"`
	// UpdatedAt 是最后一次保存时间（RFC3339），仅用于展示与排查
	UpdatedAt string `json:"updated_at,omitempty"`
	// UpdatedBy 是最后修改人，便于追溯是谁改了现场配置
	UpdatedBy string `json:"updated_by,omitempty"`
	// Remark 是本次修改说明，回滚时能看懂每一版为什么改
	Remark string `json:"remark,omitempty"`

	// PLC 是要下发给边端的 PLC 设备点表（按 code 覆盖同名设备）
	PLC []config.PLCDevice `json:"plc,omitempty"`
	// Loading 是装载位置查表
	Loading *geometry.LoadingDoc `json:"loading,omitempty"`
	// Heads 是外检锭位映射
	Heads *geometry.HeadMapsDoc `json:"heads,omitempty"`
	// Pallets 是包装栈板垛型。每层位置数可不同（底层满排、上层收窄），
	// 因为不同产品/包装机的垛型不一样，而 PLC 只给得出总层数。
	Pallets *geometry.PalletLayoutsDoc `json:"pallets,omitempty"`
}

// Validate 校验整份文档，返回全部问题而不是遇到第一个就返回。
//
// 为什么收集全部问题：现场人员在页面上一次就能看到所有错误，
// 比"改一条、保存、再报下一条"高效得多。
func Validate(doc *Document) []string {
	if doc == nil {
		return []string{"配置文档为空"}
	}
	var problems []string

	seen := make(map[string]int, len(doc.PLC))
	for i := range doc.PLC {
		d := &doc.PLC[i]
		if prev, dup := seen[d.Code]; dup {
			problems = append(problems, fmt.Sprintf(
				"PLC 设备 %q 重复出现（第 %d 条与第 %d 条）", d.Code, prev+1, i+1))
			continue
		}
		seen[d.Code] = i
		if err := config.ValidatePLCDevice(d); err != nil {
			problems = append(problems, fmt.Sprintf("PLC 设备 %q：%v", d.Code, err))
		}
	}

	if doc.Loading != nil {
		if _, err := geometry.BuildLoadingSequences(doc.Loading, "装载位置查表"); err != nil {
			problems = append(problems, err.Error())
		}
	}

	if doc.Heads != nil {
		if _, err := geometry.BuildHeadMaps(doc.Heads, "外检锭位映射"); err != nil {
			problems = append(problems, err.Error())
		}
	}

	if doc.Pallets != nil {
		if _, err := geometry.BuildPalletLayouts(doc.Pallets, "包装栈板垛型"); err != nil {
			problems = append(problems, err.Error())
		}
	}

	return problems
}

// BuildPLCDevices 校验并返回 PLC 设备列表。
//
// 返回拷贝，避免调用方修改内部状态后影响已下发的配置。
func BuildPLCDevices(doc *Document) ([]config.PLCDevice, error) {
	if doc == nil || len(doc.PLC) == 0 {
		return nil, nil
	}
	out := make([]config.PLCDevice, len(doc.PLC))
	copy(out, doc.PLC)
	for i := range out {
		if err := config.ValidatePLCDevice(&out[i]); err != nil {
			return nil, fmt.Errorf("PLC 设备 %q：%w", out[i].Code, err)
		}
	}
	return out, nil
}
