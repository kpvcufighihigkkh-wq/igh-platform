package config

import "fmt"

// PointAddress 是"展开后"的一个具体读取地址。
//
// 点表里描述的是"一个机位内的布局"，运行时需要展开成：
//   - winder = 1..count  （第几个卷绕机位）
//   - side   = left/right（双胞胎机位的两侧）
//   - field  = UDT 字段名
//
// 绝对偏移 = winder_index*stride + side_offset + field_offset
type PointAddress struct {
	Group  string // 所属 poll_group
	DB     int
	Winder int    // 1-based；非重复布局为 0
	Side   string // "left" / "right"；非双胞胎为空
	Field  string
	Offset int // 相对 poll_group.Start 的字节偏移
	Type   string
	Length int
	Bit    int
}

// Key 返回稳定标识，用于变更检测与日志。
//
// 两种形态：
//   - 无锭位：group/Field
//   - 有锭位：group/W1/left/Field；单面锭位无侧别时为 group/W1/Field
func (p PointAddress) Key() string {
	if p.Winder == 0 {
		return fmt.Sprintf("%s/%s", p.Group, p.Field)
	}
	if p.Side == "" {
		return fmt.Sprintf("%s/W%d/%s", p.Group, p.Winder, p.Field)
	}
	return fmt.Sprintf("%s/W%d/%s/%s", p.Group, p.Winder, p.Side, p.Field)
}

// ExpandPoints 把一个设备的点表展开为具体地址列表。
//
// 校验已在 ValidatePLCDevice 中完成，这里只负责展开。
func ExpandPoints(d *PLCDevice) []PointAddress {
	var out []PointAddress
	for _, g := range d.PollGroups {
		out = append(out, ExpandGroup(g)...)
	}
	return out
}

// ExpandGroup 展开单个分组。
//
// 与 ExpandPoints 的区别：**不依赖设备上的 Twin 字段**，只看分组自身的
// repeat（count/stride/sides）。这样仿真器、读取器和测试可以共用同一套
// 展开逻辑，不会出现"真实设备展开了两侧、仿真只展开了字段"这种不一致。
//
// repeat 有三种形态：
//   - nil                ：不重复，整组只有一组地址
//   - sides == nil       ：单面锭位，count 个机位，每个 stride 一个
//   - sides != nil       ：双胞胎，count 个机位 × 左右两侧
func ExpandGroup(g PLCPollGroup) []PointAddress {
	var out []PointAddress
	if g.Repeat == nil {
		for _, p := range g.Points {
			out = append(out, PointAddress{
				Group:  g.Name,
				DB:     g.DB,
				Field:  p.Name,
				Offset: p.Offset,
				Type:   p.Type,
				Length: p.Length,
				Bit:    p.Bit,
			})
		}
		return out
	}

	// 单面锭位：Side 留空，键里不带侧别。
	if g.Repeat.Sides == nil {
		for i := 0; i < g.Repeat.Count; i++ {
			base := i * g.Repeat.Stride
			for _, p := range g.Points {
				out = append(out, PointAddress{
					Group:  g.Name,
					DB:     g.DB,
					Winder: i + 1,
					Field:  p.Name,
					Offset: base + p.Offset,
					Type:   p.Type,
					Length: p.Length,
					Bit:    p.Bit,
				})
			}
		}
		return out
	}

	leftOff := g.Repeat.Sides.Left.Offset
	rightOff := g.Repeat.Sides.Right.Offset
	for i := 0; i < g.Repeat.Count; i++ {
		base := i * g.Repeat.Stride
		for _, side := range []struct {
			name   string
			offset int
		}{{"left", leftOff}, {"right", rightOff}} {
			for _, p := range g.Points {
				out = append(out, PointAddress{
					Group:  g.Name,
					DB:     g.DB,
					Winder: i + 1,
					Side:   side.name,
					Field:  p.Name,
					Offset: base + side.offset + p.Offset,
					Type:   p.Type,
					Length: p.Length,
					Bit:    p.Bit,
				})
			}
		}
	}
	return out
}

// BlockLength 返回一个 poll_group 需要读取的总字节数。
func (g *PLCPollGroup) BlockLength() int {
	if g.Length > 0 {
		return g.Length
	}
	if g.Repeat != nil && g.Repeat.Count > 0 && g.Repeat.Stride > 0 {
		return g.Repeat.Count * g.Repeat.Stride
	}
	return 0
}
