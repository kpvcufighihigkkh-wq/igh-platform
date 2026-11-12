// Package config 的 PLC 实例点表加载与校验。
//
// 点表由 tools/plcgen/extract.js 从遗留系统生成，本文件负责：
//  1. 加载 configs/devices/instances/*.yaml
//  2. 校验双胞胎布局的一致性（两侧必须收敛到同一 stride）
//  3. 校验每个字段落在 UDT 范围内
//
// 任何一条校验失败都返回错误——不允许"加载成功但地址是 NaN"这种情况。
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// PLCInstances 是一个点表文件的顶层结构。
type PLCInstances struct {
	Devices []PLCDevice `yaml:"devices"`
}

// LoadPLCInstances 加载并校验一个点表文件。
func LoadPLCInstances(path string) (*PLCInstances, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("plc instances: read %s: %w", path, err)
	}
	var doc PLCInstances
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("plc instances: parse %s: %w", path, err)
	}
	if len(doc.Devices) == 0 {
		return nil, fmt.Errorf("plc instances: %s: no devices", path)
	}
	for i := range doc.Devices {
		if err := ValidatePLCDevice(&doc.Devices[i]); err != nil {
			return nil, fmt.Errorf("plc instances: %s: device %d (%s): %w",
				path, i, doc.Devices[i].Code, err)
		}
	}
	return &doc, nil
}

// ValidatePLCDevice 校验单个设备的点表自洽性。
func ValidatePLCDevice(d *PLCDevice) error {
	if d.Code == "" {
		return fmt.Errorf("code is required")
	}
	if d.Host == "" {
		return fmt.Errorf("host is required")
	}
	if d.Port <= 0 {
		return fmt.Errorf("port must be > 0")
	}

	// ---- 双胞胎布局 ----
	if d.Twin != nil {
		if err := validateTwin(d.Twin); err != nil {
			return err
		}
	}

	// ---- 轮询组 ----
	if len(d.PollGroups) == 0 {
		// 允许只配置 read_blocks（仿真/简单场景），但 twin 与 poll_groups 必须成对
		if d.Twin != nil {
			return fmt.Errorf("twin configured but no poll_groups")
		}
		return nil
	}

	for gi := range d.PollGroups {
		g := &d.PollGroups[gi]
		if err := validatePollGroup(g, d.Twin); err != nil {
			return fmt.Errorf("poll_group %q: %w", g.Name, err)
		}
	}
	return nil
}

func validateTwin(t *PLCTwinWinder) error {
	if t.UdtSize <= 0 {
		return fmt.Errorf("twin.udt_size must be > 0")
	}
	if t.Count <= 0 {
		return fmt.Errorf("twin.count must be > 0")
	}
	if t.Stride <= 0 {
		return fmt.Errorf("twin.stride must be > 0")
	}

	evenEnd := t.OffsetEven + t.UdtSize + t.OffsetEndEven
	oddEnd := t.OffsetOdd + t.UdtSize + t.OffsetEndOdd
	if evenEnd != oddEnd {
		return fmt.Errorf(
			"twin sides do not converge: even end %d (%d+%d+%d) != odd end %d (%d+%d+%d)",
			evenEnd, t.OffsetEven, t.UdtSize, t.OffsetEndEven,
			oddEnd, t.OffsetOdd, t.UdtSize, t.OffsetEndOdd)
	}
	if t.Stride != evenEnd {
		return fmt.Errorf("twin.stride %d != even/odd end %d", t.Stride, evenEnd)
	}

	if t.Sides != nil {
		for name, off := range map[string]int{"left": t.Sides.Left.Offset, "right": t.Sides.Right.Offset} {
			if off < 0 {
				return fmt.Errorf("twin.sides.%s.offset must be >= 0", name)
			}
			if off+t.UdtSize > t.Stride {
				return fmt.Errorf(
					"twin.sides.%s range [%d, %d) exceeds stride %d",
					name, off, off+t.UdtSize, t.Stride)
			}
		}
	}
	return nil
}

func validatePollGroup(g *PLCPollGroup, twin *PLCTwinWinder) error {
	if g.DB <= 0 {
		return fmt.Errorf("db must be > 0")
	}
	if g.Length <= 0 {
		return fmt.Errorf("length must be > 0")
	}

	if g.Repeat != nil {
		if g.Repeat.Count <= 0 {
			return fmt.Errorf("repeat.count must be > 0")
		}
		if g.Repeat.Stride <= 0 {
			return fmt.Errorf("repeat.stride must be > 0")
		}
		if g.Length != g.Repeat.Count*g.Repeat.Stride {
			return fmt.Errorf(
				"length %d != repeat.count %d * repeat.stride %d",
				g.Length, g.Repeat.Count, g.Repeat.Stride)
		}
		if twin != nil {
			if g.Repeat.Stride != twin.Stride {
				return fmt.Errorf("repeat.stride %d != twin.stride %d",
					g.Repeat.Stride, twin.Stride)
			}
			if g.Repeat.Count != twin.Count {
				return fmt.Errorf("repeat.count %d != twin.count %d",
					g.Repeat.Count, twin.Count)
			}
		}
	}

	// 字段必须落在单个 UDT 内
	udtSize := 0
	if twin != nil {
		udtSize = twin.UdtSize
	}
	for _, p := range g.Points {
		if p.Name == "" {
			return fmt.Errorf("point with empty name")
		}
		if p.Offset < 0 {
			return fmt.Errorf("point %q: offset must be >= 0", p.Name)
		}
		w, err := pointWidth(p)
		if err != nil {
			return fmt.Errorf("point %q: %w", p.Name, err)
		}
		if udtSize > 0 && p.Offset+w > udtSize {
			return fmt.Errorf(
				"point %q range [%d, %d) exceeds udt size %d",
				p.Name, p.Offset, p.Offset+w, udtSize)
		}
	}
	return nil
}

// pointWidth 返回字段占用的字节数。
func pointWidth(p PLCPoint) (int, error) {
	switch p.Type {
	case "bool":
		return 1, nil
	case "byte", "uint8", "int8":
		return 1, nil
	case "int16", "uint16":
		return 2, nil
	case "int32", "uint32", "float32":
		return 4, nil
	case "float64":
		return 8, nil
	case "string":
		if p.Length <= 0 {
			return 0, fmt.Errorf("string length must be > 0")
		}
		return p.Length, nil
	default:
		return 0, fmt.Errorf("unsupported type %q", p.Type)
	}
}

// Sides 是双胞胎左右两侧的便捷访问。
func (t *PLCTwinWinder) SidesOrDefault() (left, right int) {
	if t.Sides != nil {
		return t.Sides.Left.Offset, t.Sides.Right.Offset
	}
	return t.OffsetEven, t.OffsetOdd
}

// AddressOf 计算第 index 个机位（0-based）、某一侧、某字段的绝对字节偏移。
//
//	address = base + index*stride + sideOffset + fieldOffset
func (t *PLCTwinWinder) AddressOf(index int, leftSide bool, fieldOffset int) int {
	left, right := t.SidesOrDefault()
	side := right
	if leftSide {
		side = left
	}
	return index*t.Stride + side + fieldOffset
}
