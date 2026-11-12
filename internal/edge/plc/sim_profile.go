package plc

import (
	"fmt"
	"math"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/config"
)

// SimProfile 是仿真器使用的点表布局。
//
// 仿真器不再生成"假 DB1/DB2"，而是按真实点表的字段与偏移生成数据，
// 这样边端在仿真模式下走的是与真机完全相同的解码路径。
type SimProfile struct {
	// Groups 按 DB 号索引
	Groups map[int][]config.PLCPollGroup
}

// NewSimProfile 从设备配置构造仿真布局。
func NewSimProfile(dev config.PLCDevice) *SimProfile {
	p := &SimProfile{Groups: make(map[int][]config.PLCPollGroup)}
	for _, g := range dev.PollGroups {
		p.Groups[g.DB] = append(p.Groups[g.DB], g)
	}
	return p
}

// Empty 表示没有可用的点表布局。
func (p *SimProfile) Empty() bool {
	return p == nil || len(p.Groups) == 0
}

// fill 按点表把生成值写入 buf。
//
// buf 对应 (db, start, size)；字段的绝对偏移先减去 start 再写入。
func (s *simulationProtocol) fillFromProfile(buf []byte, dbNumber, start int, elapsed float64) bool {
	if s.profile == nil || s.profile.Empty() {
		return false
	}
	groups, ok := s.profile.Groups[dbNumber]
	if !ok {
		return false
	}

	wrote := false
	for _, g := range groups {
		for _, p := range config.ExpandGroup(g) {
			abs := g.Start + p.Offset
			rel := abs - start
			w := simWidth(p)
			if w <= 0 || rel < 0 || rel+w > len(buf) {
				continue
			}
			value := simValueFor(p, elapsed, s.baseTime())
			// 只写字段自身的宽度，绝不能传 buf[rel:]，
			// 否则字符串分支会把后续所有字段覆盖成空格。
			if writeSimValue(buf[rel:rel+w], p.Type, value) {
				wrote = true
			}
		}
	}
	return wrote
}

// baseTime 返回仿真数据的基准时刻。
//
// 用协议自己的 startTime，而不是 time.Now()：
//
//  1. **可复现**：同一 elapsed 必须产出同一份数据。原来 EndTimeOfPack 直接
//     取 time.Now()，两次"同一时刻"的填充只要跨过秒边界就会不同——
//     调试时看到数据自己变，会误以为是业务逻辑在动。
//  2. **与 elapsed 一致**：elapsed 本来就是相对 startTime 的偏移，
//     时间类字段也要用同一个基准，否则"10 秒后"的语义是混的。
//
// startTime 为零（未 Connect，测试直接 fillFromProfile 的情况）时退回一个
// 固定的基准，保证结果仍然只由入参决定。
func (s *simulationProtocol) baseTime() time.Time {
	if s.startTime.IsZero() {
		return simEpoch
	}
	return s.startTime
}

// simEpoch 是未连接时使用的固定基准，仅用于让仿真数据可复现。
var simEpoch = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

// simWidth 返回字段宽度（与 config 侧保持一致）。
func simWidth(p config.PointAddress) int {
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

// simValueFor 按字段语义生成一个合理值。
//
// 规则：
//   - Doff_No / DoffNo: 每个机位按 25 秒周期递增，相邻机位错开
//   - EndTimeOfPack: 基准时刻 + elapsed，再回退到本机位上次完成的整秒
//   - POSITION_Name: "L<line>-<winder><side>" 形式，便于人工核对
//   - MachineName: 固定的机台名
//   - CodeNumber: 稳定的批号样式
//   - Yarn_Type: 从固定集合取
//
// base 是时间类字段的基准时刻，由调用方传入（见 baseTime）。
// 不能在这里取 time.Now()：那会让"同一 elapsed"产出不同数据，
// 既不可复现，也会让时间类字段与 elapsed 脱节。
func simValueFor(p config.PointAddress, elapsed float64, base time.Time) any {
	winder := p.Winder
	if winder == 0 {
		winder = 1
	}
	side := p.Side
	if side == "" {
		side = "left"
	}
	// 相邻机位错开 2.5 秒，避免所有机位同时落筒
	phase := elapsed + float64(winder)*2.5
	// 每 25 秒一次落筒
	doffNo := int(phase/25.0) + 1

	name := p.Field
	switch {
	case contains(name, "POSITION_Name"):
		return fmt.Sprintf("%s-%02d%s", sideCode(side), winder, abbrev(side))
	case contains(name, "Doff_No"), contains(name, "DoffNo"):
		return uint16(doffNo)
	case contains(name, "EndTimeOfPack"):
		// 基准时刻 + elapsed，再回退到本机位上次完成的整秒。
		// 用 base 而不是 time.Now()：同一 elapsed 必须得到同一结果，
		// 否则仿真数据会自己变，调试时无法区分"业务在动"和"时间在走"。
		return uint32(base.Add(time.Duration(elapsed*float64(time.Second))).Unix() -
			int64(25.0-elapsedPhase(phase, 25.0)))
	case contains(name, "Yarn_Type"), contains(name, "ProductionData"):
		return yarnTypes[(winder-1)%len(yarnTypes)]
	case contains(name, "MachineName"), contains(name, "MachineNo"):
		return fmt.Sprintf("WDC-%02dL", winder)
	case contains(name, "CodeNumber"):
		return fmt.Sprintf("G2091%06d", 200000+winder*37)
	case contains(name, "PositionID"), contains(name, "PositionNo"):
		return uint16(winder)
	case contains(name, "CalculateNetWeight"):
		return float32(3.2 + float64(winder%5)*0.05)
	case contains(name, "Quality"):
		return uint16(0xFFFF)
	default:
		// 未识别字段：按类型给一个稳定占位值
		switch p.Type {
		case "string":
			return "SIM"
		case "float32", "float64":
			return float32(1.0)
		default:
			return uint16(0)
		}
	}
}

func elapsedPhase(t, period float64) float64 {
	return t - float64(int(t/period))*period
}

func contains(s, sub string) bool {
	return len(sub) <= len(s) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func sideCode(side string) string {
	if side == "right" {
		return "R"
	}
	return "L"
}

func abbrev(side string) string {
	if side == "right" {
		return "R"
	}
	return "L"
}

var yarnTypes = []string{"120/48", "150/72", "75/24", "300/96"}

// writeSimValue 把生成值按 S7 大端序写入缓冲区。
func writeSimValue(buf []byte, typ string, value any) bool {
	switch typ {
	case "bool":
		b, ok := value.(bool)
		if !ok {
			return false
		}
		if b {
			buf[0] = 1
		} else {
			buf[0] = 0
		}
		return true
	case "byte", "uint8":
		v, ok := toUint64(value)
		if !ok {
			return false
		}
		buf[0] = byte(v)
		return true
	case "int8":
		v, ok := toInt64(value)
		if !ok {
			return false
		}
		buf[0] = byte(v)
		return true
	case "uint16", "int16":
		v, ok := toInt64(value)
		if !ok {
			return false
		}
		buf[0] = byte(uint16(v) >> 8)
		buf[1] = byte(uint16(v))
		return true
	case "uint32", "int32":
		v, ok := toInt64(value)
		if !ok {
			return false
		}
		u := uint32(v)
		buf[0] = byte(u >> 24)
		buf[1] = byte(u >> 16)
		buf[2] = byte(u >> 8)
		buf[3] = byte(u)
		return true
	case "float32":
		f, ok := toFloat64(value)
		if !ok {
			return false
		}
		u := math.Float32bits(float32(f))
		buf[0] = byte(u >> 24)
		buf[1] = byte(u >> 16)
		buf[2] = byte(u >> 8)
		buf[3] = byte(u)
		return true
	case "string":
		s, ok := value.(string)
		if !ok {
			return false
		}
		for i := 0; i < len(buf); i++ {
			buf[i] = 0x20 // 空格填充
		}
		for i := 0; i < len(s) && i < len(buf); i++ {
			buf[i] = s[i]
		}
		// 剩余保持空格，模拟 S7 C 字符串
		return true
	default:
		return false
	}
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int8:
		return int64(n), true
	case uint8:
		return int64(n), true
	case int16:
		return int64(n), true
	case uint16:
		return int64(n), true
	case int32:
		return int64(n), true
	case uint32:
		return int64(n), true
	case int64:
		return n, true
	default:
		return 0, false
	}
}

func toUint64(v any) (uint64, bool) {
	n, ok := toInt64(v)
	if !ok || n < 0 {
		return 0, false
	}
	return uint64(n), true
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float32:
		return float64(n), true
	case float64:
		return n, true
	case int:
		return float64(n), true
	default:
		return 0, false
	}
}
