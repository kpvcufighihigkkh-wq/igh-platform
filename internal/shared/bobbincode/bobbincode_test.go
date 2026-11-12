package bobbincode

import (
	"strings"
	"testing"
	"time"
)

// testAt 是全部用例共用的落筒时刻：2025-04-16 09:30:15 (+08)。
// 固定值让编码断言稳定可读。
var testAt = time.Date(2025, 4, 16, 9, 30, 15, 0, CodeZone)

// TestBuild_Format 固定编码格式。
//
// 这个格式一旦改了，已经贴在现场的丝锭就扫不出来了——
// 所以它不是"实现细节"，需要显式固定住。
func TestBuild_Format(t *testing.T) {
	cases := []struct {
		name   string
		lot    string
		line   string
		winder int
		side   string
		doff   int
		place  int
		want   string
	}{
		{"双侧-左侧", "250416A", "L01", 3, "left", 12, 3, "250416A-L01-W3L-250416093015-012-03"},
		{"双侧-右侧", "250416A", "L01", 3, "right", 12, 3, "250416A-L01-W3R-250416093015-012-03"},
		{"单面机", "250416A", "L01", 5, "", 7, 1, "250416A-L01-W5N-250416093015-007-01"},
		{"落筒号三位足够", "X", "L02", 1, "", 1, 1, "X-L02-W1N-250416093015-001-01"},
		{"落筒号超三位不截断", "X", "L02", 1, "", 1234, 5, "X-L02-W1N-250416093015-1234-05"},
		{"锭位超两位不截断", "X", "L02", 1, "", 1, 128, "X-L02-W1N-250416093015-001-128"},
		// 产线为空时省略该段，避免出现 "LOT--W3L-..." 的双连字符
		{"无产线编码", "X", "", 1, "", 1, 1, "X-W1N-250416093015-001-01"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Build(tc.lot, tc.line, tc.winder, tc.side, tc.doff, tc.place, testAt)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Build = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestBuild_Uniqueness 是本包最重要的性质：
// 不同的 (批号, 机位, 侧别, 落筒号, 锭位) 必须得到不同的编码。
//
// 靠计数器生成编码之所以不可取，就是因为这个性质会在重启后失效。
// 这里穷举一段范围验证推导规则本身没有碰撞。
func TestBuild_Uniqueness(t *testing.T) {
	seen := make(map[string]string)
	add := func(lot, line string, winder int, side string, doff, place int) {
		code, err := Build(lot, line, winder, side, doff, place, testAt)
		if err != nil {
			t.Fatalf("Build(%s,%s,%d,%s,%d,%d): %v", lot, line, winder, side, doff, place, err)
		}
		key := code
		ident := strings.Join([]string{lot, line, side, itoa(winder), itoa(doff), itoa(place)}, "|")
		if prev, dup := seen[key]; dup {
			t.Fatalf("编码碰撞: %q 同时来自 %s 和 %s", code, prev, ident)
		}
		seen[key] = ident
	}

	lots := []string{"250416A", "250416B", "250417A"}
	lines := []string{"L01", "L02"}
	sides := []string{"left", "right", ""}
	for _, lot := range lots {
		for _, line := range lines {
			for winder := 1; winder <= 4; winder++ {
				for _, side := range sides {
					for doff := 1; doff <= 20; doff++ {
						for place := 1; place <= 8; place++ {
							add(lot, line, winder, side, doff, place)
						}
					}
				}
			}
		}
	}
	if len(seen) != len(lots)*len(lines)*4*len(sides)*20*8 {
		t.Fatalf("编码数量 %d 与输入组合数不符", len(seen))
	}
}

func itoa(n int) string {
	return strings.TrimSpace(strings.Repeat(" ", 0) + intToStr(n))
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// TestBuild_RejectsInvalid 覆盖非法输入。
func TestBuild_RejectsInvalid(t *testing.T) {
	cases := []struct {
		name           string
		lot            string
		winder         int
		doff, place    int
	}{
		{"空批号", "  ", 1, 1, 1},
		{"机位号为 0", "X", 0, 1, 1},
		{"机位号为负", "X", -1, 1, 1},
		{"落筒号为 0", "X", 1, 0, 1},
		{"锭位为 0", "X", 1, 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Build(tc.lot, "L01", tc.winder, "", tc.doff, tc.place, testAt); err == nil {
				t.Fatal("应当报错")
			}
		})
	}
}

// TestBuild_LotWithSeparator 验证批号里的分隔符被替换。
//
// 批号是外部数据，可能含 "-" 或空格。不处理的话
// "A-B" + 机位 W1 + ... 会得到 "A-B-W1N-001-01"，
// 人眼和解析都分不清哪一段是批号。
func TestBuild_LotWithSeparator(t *testing.T) {
	got, err := Build("A-B C", "L01", 1, "", 1, 1, testAt)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got != "A_B_C-L01-W1N-250416093015-001-01" {
		t.Fatalf("Build = %q", got)
	}
	// 5 个分隔符：批号↔产线、产线↔机位、机位↔日期段、日期段↔落筒号、落筒号↔锭位
	if strings.Count(got, "-") != 5 {
		t.Fatalf("编码里应恰好有 5 个分隔符: %q", got)
	}
}

// TestParseWinderName 覆盖边端上报的机位名格式。
func TestParseWinderName(t *testing.T) {
	cases := []struct {
		in         string
		winder     int
		side       string
		wantErr    bool
	}{
		{"plc-L01/W3/left", 3, "left", false},
		{"plc-L01/W3", 3, "", false},
		{"w3", 0, "", true},
		{"nogear", 0, "", true},
		{"plc-L01/X3/left", 0, "", true},
		{"plc-L01/W0", 0, "", true},
		// 设备名本身带斜杠：只认最后两段
		{"site/plc-L01/W7/right", 7, "right", false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseWinderName(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("期望报错，得到 %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseWinderName: %v", err)
			}
			if got.Winder != tc.winder || got.Side != tc.side {
				t.Fatalf("got %+v, want {%d %q}", got, tc.winder, tc.side)
			}
		})
	}
}

// TestRoundTrip_WinderName 验证解析出的身份再编码后仍唯一。
func TestRoundTrip_WinderName(t *testing.T) {
	names := []string{"plc-L01/W1/left", "plc-L01/W1/right", "plc-L01/W2/left", "plc-L01/W2"}
	seen := make(map[string]string, len(names))
	for _, n := range names {
		id, err := ParseWinderName(n)
		if err != nil {
			t.Fatalf("ParseWinderName(%q): %v", n, err)
		}
		code, err := Build("LOT1", "L01", id.Winder, id.Side, 1, 1, testAt)
		if err != nil {
			t.Fatalf("Build(%q): %v", n, err)
		}
		if prev, dup := seen[code]; dup {
			t.Fatalf("%q 与 %q 得到同一编码 %q", prev, n, code)
		}
		seen[code] = n
	}
}

// TestBuild_UniqueAcrossLines 覆盖"同一批次并行在多条产线生产"的场景。
//
// 这是实际会发生的：Lot 模型没有 line_id，几条线可以同时做同一个批号，
// 而各线的机位号都从 1 开始编。编码里少了产线就会撞码，
// 撞上唯一索引后整条落筒的事务回滚——实物照常出垛，数据却丢了。
func TestBuild_UniqueAcrossLines(t *testing.T) {
	const lot = "250416A"
	// 两条线、完全相同的机位/侧别/落筒号/锭位
	a, err := Build(lot, "L01", 3, "left", 12, 1, testAt)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	b, err := Build(lot, "L02", 3, "left", 12, 1, testAt)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if a == b {
		t.Fatalf("跨产线撞码: %q == %q", a, b)
	}
	if a != "250416A-L01-W3L-250416093015-012-01" || b != "250416A-L02-W3L-250416093015-012-01" {
		t.Fatalf("编码格式不符: %q / %q", a, b)
	}

	// 再确认机位号 >= 10 时也不会歧义：
	// 若产线编号以数字结尾、机位号也可能多位，少了分隔符会分不清边界。
	c, _ := Build(lot, "L1", 23, "left", 12, 1, testAt)
	d, _ := Build(lot, "L12", 3, "left", 12, 1, testAt)
	if c == d {
		t.Fatalf("产线/机位边界歧义: %q == %q", c, d)
	}
}

// TestBuild_LineWithSeparator 验证产线编码里的分隔符也被清理。
func TestBuild_LineWithSeparator(t *testing.T) {
	got, err := Build("LOT", "L-01 A", 1, "", 1, 1, testAt)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got != "LOT-L_01_A-W1N-250416093015-001-01" {
		t.Fatalf("Build = %q", got)
	}
}
