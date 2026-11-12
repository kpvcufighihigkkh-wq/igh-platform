package shift

import (
	"testing"
	"time"
)

// mk 构造一个"北京时间"的时刻，返回其 UTC 表示（模拟 PLC 传入的 UTC 时间戳）。
func mk(month time.Month, day, hour, minute int) time.Time {
	return mkYear(2026, month, day, hour, minute)
}

// mkYear 允许指定年份，用于覆盖闰年等场景。
func mkYear(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.FixedZone("CST", 8*3600)).UTC()
}

func TestCalcTeamTurn(t *testing.T) {
	cases := []struct {
		name string
		t    time.Time
		want TeamTurn
	}{
		// ---- 上旬（1–15 日）：08:00–20:00 → 1，其余 → 2 ----
		{"上旬 00:00 → 2", mk(time.March, 5, 0, 0), TeamTurnB},
		{"上旬 07:59 → 2", mk(time.March, 5, 7, 59), TeamTurnB},
		{"上旬 08:00 → 1", mk(time.March, 5, 8, 0), TeamTurnA},
		{"上旬 12:00 → 1", mk(time.March, 5, 12, 0), TeamTurnA},
		{"上旬 19:59 → 1", mk(time.March, 5, 19, 59), TeamTurnA},
		{"上旬 20:00 → 2", mk(time.March, 5, 20, 0), TeamTurnB},
		{"上旬 23:59 → 2", mk(time.March, 5, 23, 59), TeamTurnB},

		// ---- 上旬边界：1 日 00:00–08:00 → 2 ----
		{"1 日 00:00 → 2", mk(time.March, 1, 0, 0), TeamTurnB},
		{"1 日 07:59 → 2", mk(time.March, 1, 7, 59), TeamTurnB},
		{"1 日 08:00 → 1", mk(time.March, 1, 8, 0), TeamTurnA},

		// ---- 上旬边界：15 日 ----
		{"15 日 00:00 → 2", mk(time.March, 15, 0, 0), TeamTurnB},
		{"15 日 08:00 → 1", mk(time.March, 15, 8, 0), TeamTurnA},
		{"15 日 15:59 → 1", mk(time.March, 15, 15, 59), TeamTurnA},
		{"15 日 16:00 → 2", mk(time.March, 15, 16, 0), TeamTurnB},
		{"15 日 23:59 → 2", mk(time.March, 15, 23, 59), TeamTurnB},

		// ---- 下旬（16–月末）：08:00–20:00 → 2，其余 → 1（与上旬相反） ----
		{"下旬 00:00 → 1", mk(time.March, 20, 0, 0), TeamTurnA},
		{"下旬 07:59 → 1", mk(time.March, 20, 7, 59), TeamTurnA},
		{"下旬 08:00 → 2", mk(time.March, 20, 8, 0), TeamTurnB},
		{"下旬 12:00 → 2", mk(time.March, 20, 12, 0), TeamTurnB},
		{"下旬 19:59 → 2", mk(time.March, 20, 19, 59), TeamTurnB},
		{"下旬 20:00 → 1", mk(time.March, 20, 20, 0), TeamTurnA},
		{"下旬 23:59 → 1", mk(time.March, 20, 23, 59), TeamTurnA},

		// ---- 下旬边界：16 日 00:00–08:00 → 1 ----
		{"16 日 00:00 → 1", mk(time.March, 16, 0, 0), TeamTurnA},
		{"16 日 07:59 → 1", mk(time.March, 16, 7, 59), TeamTurnA},
		{"16 日 08:00 → 2", mk(time.March, 16, 8, 0), TeamTurnB},

		// ---- 月下旬边界：月末（3 月 31 日） ----
		{"月末 00:00 → 1", mk(time.March, 31, 0, 0), TeamTurnA},
		{"月末 08:00 → 2", mk(time.March, 31, 8, 0), TeamTurnB},
		{"月末 15:59 → 2", mk(time.March, 31, 15, 59), TeamTurnB},
		{"月末 16:00 → 1", mk(time.March, 31, 16, 0), TeamTurnA},
		{"月末 23:59 → 1", mk(time.March, 31, 23, 59), TeamTurnA},

		// ---- 不同月份长度：2 月（非闰年 28 天） ----
		{"2 月 28 日 08:00 → 2", mk(time.February, 28, 8, 0), TeamTurnB},
		{"2 月 28 日 16:00 → 1", mk(time.February, 28, 16, 0), TeamTurnA},
		// ---- 闰年 2 月 29 日 ----
		{"闰年 2 月 29 日 08:00 → 2", mkYear(2028, time.February, 29, 8, 0), TeamTurnB},
		{"闰年 2 月 29 日 16:00 → 1", mkYear(2028, time.February, 29, 16, 0), TeamTurnA},
		// ---- 30 天的月份 ----
		{"4 月 30 日 08:00 → 2", mk(time.April, 30, 8, 0), TeamTurnB},
		{"4 月 30 日 16:00 → 1", mk(time.April, 30, 16, 0), TeamTurnA},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CalcTeamTurn(tc.t); got != tc.want {
				t.Fatalf("CalcTeamTurn(%s) = %d, want %d",
					tc.t.UTC().Add(BeijingOffset).Format("2006-01-02 15:04"), got, tc.want)
			}
		})
	}
}

// TestCalcTeamTurn_AcceptsUTC 验证：传入 UTC 时间会先 +8 小时再判断。
// 北京时间 2026-03-05 12:00 = UTC 2026-03-05 04:00。
func TestCalcTeamTurn_AcceptsUTC(t *testing.T) {
	utc := time.Date(2026, time.March, 5, 4, 0, 0, 0, time.UTC)
	if got := CalcTeamTurn(utc); got != TeamTurnA {
		t.Fatalf("got %d, want %d (UTC 04:00 = 北京 12:00)", got, TeamTurnA)
	}

	// UTC 2026-03-05 23:30 = 北京 2026-03-06 07:30 → 上旬 07:30 → 班次 2
	utc2 := time.Date(2026, time.March, 5, 23, 30, 0, 0, time.UTC)
	if got := CalcTeamTurn(utc2); got != TeamTurnB {
		t.Fatalf("got %d, want %d (UTC 23:30 = 北京次日 07:30)", got, TeamTurnB)
	}
}

// TestCalcTeamTurn_AlwaysInRange 保证返回值只可能是 1 或 2。
func TestCalcTeamTurn_AlwaysInRange(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 60*24*31; i++ {
		tm := start.Add(time.Duration(i) * time.Minute)
		got := CalcTeamTurn(tm)
		if got != TeamTurnA && got != TeamTurnB {
			t.Fatalf("CalcTeamTurn(%s) = %d, out of range", tm, got)
		}
	}
}

// TestLastDayOfMonth 验证月末计算覆盖闰年与不同月份长度。
func TestLastDayOfMonth(t *testing.T) {
	cases := []struct {
		t    time.Time
		want int
	}{
		{time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC), 31},
		{time.Date(2026, time.February, 10, 0, 0, 0, 0, time.UTC), 28},
		{time.Date(2028, time.February, 10, 0, 0, 0, 0, time.UTC), 29}, // 闰年
		{time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC), 30},
	}
	for _, tc := range cases {
		if got := lastDayOfMonth(tc.t); got != tc.want {
			t.Fatalf("lastDayOfMonth(%s) = %d, want %d", tc.t.Format("2006-01"), got, tc.want)
		}
	}
}
