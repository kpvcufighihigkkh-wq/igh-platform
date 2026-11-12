// Package bobbincode 生成丝锭（丝饼）的唯一标识。
//
// 为什么需要它：外检工位靠扫码识别丝锭，而 bobbins 表原来只有自增主键。
// 自增 ID 有三个致命问题——没印在任何东西上（工人手上没有可扫的标识）、
// 跨环境重建后不稳定、且看不出是哪条线哪个锭位。
//
// ## 编码规则
//
//	{批号}-{产线}-{机位}{侧别}-{落筒号}-{锭位}
//
// 例：批号 250416A、线 L01、机位 W3、左侧、第 12 次落筒、第 3 个锭
//
//	250416A-L01-W3L-012-03
//
// ## 唯一性怎么保证
//
// 编码由**业务身份确定性推导**，不依赖任何计数器。
// 这一点是刻意的：边端断网重启、本地库重建、换机器都会让计数器归零，
// 靠计数器生成的编码迟早重复，而重复的丝锭码在追溯时是灾难
// （客诉查锭会查到另一只丝锭上）。
//
// 推导用的五元组 (lot_code, line_code, winder, side, doff_no, place)
// 与数据库已有的约束 uk_doffing(line_id, winder_name, doff_no, lot_id)
// 是同一套身份：一次落筒唯一，加上锭位序号后每个丝锭唯一。
// 另外 bobbins.code 上有唯一索引兜底——万一推导规则被破坏，
// 插入会直接失败而不是静默写入重复码。
//
// ## 为什么产线必须进编码
//
// 批号**不隐含产线**。Lot 模型没有 line_id，一个是同一批次可以并行在
// 多条产线上生产；另一个是 uk_doffing 本身就带 line_id，
// 说明 schema 允许"同批次 + 同机位 + 同落筒号"出现在不同产线。
//
// 漏掉产线会导致 O 线相撞：
//
//	L01: W3 左 第12次落筒 第1锭  →  250416A-W3L-012-01
//	L02: W3 左 第12次落筒 第1锭  →  250416A-W3L-012-01   ← 撞
//
// 撞上唯一索引后整条落筒的事务回滚（8 个锭全丢），事件被丢弃，
// 而实物照常出垛——数据与实物从这一刻起永久错位。
//
// ## 边界情况
//
// 同一产线同一批号下，PLC 的落筒号如果重置，会撞 uk_doffing——
// 那是既有约束的问题（落筒本身就无法区分），不在本包范围内，
// 但编码继承同样的语义：它不会凭空造出"看起来不同、实际是同一只丝锭"的第二套编码。
package bobbincode

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Format 描述编码的组成，供文档与校验共用。
const Format = "{lot_code}-{line_code}-{winder}{side}-{yyMMddHHmmss}-{doff_no:03d}-{place:02d}"

// TimeLayout 是编码里日期段的格式。固定 12 位数字，不带分隔符：
// 标签宽度有限，且纯数字肉眼比对时不容易看错。
const TimeLayout = "060102150405"

// CodeZone 是编码日期段使用的时区，固定为现场时区（UTC+8）。
//
// 刻意不读 time.Local / 配置：边端和中心必须算出同一个字符串，
// 只要有一台机器的时区设置不同，同一只丝锭就会得到两个编号。
// 中国现场没有夏令时，固定偏移是安全的。
var CodeZone = time.FixedZone("CST", 8*3600)

// MaxLength 是编码的最大长度，对应 bobbins.code 列宽。
//
// 批号最长 64（Lot.Code 是 varchar(64)），产线最长 16：
// 64 + 1 + 16 + 1 + 5(机位+侧别) + 1 + 12 + 1 + 3 + 1 + 2 = 107。
// 取 128 留余量。
const MaxLength = 128

// SideNone 表示该锭位没有左右侧别（单面锭位机型）。
const SideNone = "N"

// sideLetter 把配置里的侧别转成编码用的单字母。
//
// 点表里用的是 left / right，编码里要短：工人肉眼核对时
// "W3L" 比 "W3-left" 好认，标签也省宽度。
func sideLetter(side string) string {
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "left", "l", "a":
		return "L"
	case "right", "r", "b":
		return "R"
	case "":
		return SideNone
	default:
		// 未知侧别不猜。取首字母大写，保持可读性，
		// 同时让编码仍然唯一（不同侧别的首字母通常不同）。
		return strings.ToUpper(string(side)[:1])
	}
}

// Build 拼出丝锭编码。
//
// winder 是机位号（如 3 → "W3"）。lineCode 是产线编码（如 "L01"）。
// at 是落筒完成时刻（必须与上报给中心的 end_time 是同一个值）。
//
// ## 为什么必须带时间
//
// PLC 的落筒号会循环（从 N 回到 1）。只靠 (批号, 产线, 机位, 落筒号, 锭位)
// 无法唯一：循环后的第 12 次落筒与上一轮的第 12 次落筒完全同码。
//
// 落筒的物理身份其实是"哪条线的哪个机位在哪一刻落的这个筒"——
// 同一机位不可能在同一秒落两次筒（一个落筒周期是分钟级）。
// 因此把落筒完成时刻作为日期段进入编码，就得到了跨循环、跨批次、
// 跨产线都唯一的标识。
//
// 时间取 end_time 而不是"生成编码的时刻"：end_time 是边端产生并
// 随事件上报的，重传时不变，所以幂等性成立；边端断网时也能自己算。
//
// ## 为什么按固定时区格式化
//
// 边端和中心必须算出**同一个字符串**。如果各自按本地时区格式化，
// 一台机器时区配错就会得到相差 8 小时的码，同一只丝锭会有两个编号。
// 这里固定用 UTC+8（现场时区），不读运行环境的时区设置。
func Build(lotCode, lineCode string, winder int, side string, doffNo, place int, at time.Time) (string, error) {
	lot := strings.TrimSpace(lotCode)
	if lot == "" {
		return "", fmt.Errorf("丝锭编码缺少批号")
	}
	if winder <= 0 {
		return "", fmt.Errorf("丝锭编码缺少机位号")
	}
	if doffNo <= 0 {
		return "", fmt.Errorf("丝锭编码的落筒号必须从 1 开始，当前 %d", doffNo)
	}
	if place <= 0 {
		return "", fmt.Errorf("丝锭编码的锭位必须从 1 开始，当前 %d", place)
	}
	if at.IsZero() {
		return "", fmt.Errorf("丝锭编码缺少落筒时刻")
	}

	// 批号/产线里的分隔符会被编码自己的分隔符混淆
	// （"A-B-C" 无法判断边界），换成下划线而不是报错：
	// 两者都是外部数据，编码要能容错。
	clean := strings.NewReplacer("-", "_", " ", "_")
	lot = clean.Replace(lot)
	line := clean.Replace(strings.TrimSpace(lineCode))

	// 产线为空时省略这一段，避免出现 "LOT--W3L-..." 这种双连字符，
	// 让人误以为解析出错。
	var mid string
	if line != "" {
		mid = "-" + line
	}

	// 截到秒：纳秒级差异对唯一性没有帮助，却会让编码长得离谱。
	// 同一机位两次落筒之间至少相隔分钟级，秒级足够区分。
	stamp := at.In(CodeZone).Format(TimeLayout)

	// 落筒号保留在编码里：它是工人对照 PLC 屏幕的唯一凭据，
	// 去掉会让现场无法核对。但它不再承担唯一性——时间戳承担。
	code := fmt.Sprintf("%s%s-W%d%s-%s-%03d-%02d",
		lot, mid, winder, sideLetter(side), stamp, doffNo, place)
	if len(code) > MaxLength {
		return "", fmt.Errorf("丝锭编码过长（%d > %d）：%s", len(code), MaxLength, code)
	}
	return code, nil
}

// WinderIdentity 是从 winder_name 里解析出的机位身份。
type WinderIdentity struct {
	Winder int
	Side   string
}

// ParseWinderName 从 winder_name 解析机位号与侧别。
//
// 格式与边端上报一致（见 winding/points.go）：
//
//	"device/W3/left"  → {3, "left"}
//	"device/W3"       → {3, ""}
//
// 判断依据是**最后一段的形态**，不是段落数量：
// "plc-L01/W3" 只有两段但无侧别，"site/plc/W3/right" 四段有侧别。
// 按段数猜会把前者解析成"机位号=plc-L01"，直接报错。
func ParseWinderName(name string) (WinderIdentity, error) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) < 2 {
		return WinderIdentity{}, fmt.Errorf("机位名格式非法（需要 device/W<N>[/side]）: %q", name)
	}

	winderPart := parts[len(parts)-1]
	side := ""
	// 最后一段不是 W+数字，说明它是侧别，机位号在前一段
	if _, err := parseWinderNo(winderPart); err != nil {
		side = winderPart
		winderPart = parts[len(parts)-2]
	}

	n, err := parseWinderNo(winderPart)
	if err != nil {
		return WinderIdentity{}, fmt.Errorf("%w（机位名 %q）", err, name)
	}
	return WinderIdentity{Winder: n, Side: side}, nil
}

// parseWinderNo 解析 "W3" 这种机位号。
func parseWinderNo(s string) (int, error) {
	up := strings.ToUpper(strings.TrimSpace(s))
	if !strings.HasPrefix(up, "W") {
		return 0, fmt.Errorf("机位号缺少 W 前缀: %q", s)
	}
	n, err := strconv.Atoi(strings.TrimPrefix(up, "W"))
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("机位号不是正整数: %q", s)
	}
	return n, nil
}
