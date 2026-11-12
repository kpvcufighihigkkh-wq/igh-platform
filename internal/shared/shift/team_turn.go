// Package shift 迁移自遗留 MySQL 函数 calc_team_turn。
//
// 遗留实现（DataBase/h028.sql）：
//
//	SET @localTimestamp = IN_timestamp + INTERVAL 8 HOUR;   -- PLC 给的是 UTC
//	SET @dayNumber = dayofmonth(@localTimestamp);
//	IF @dayNumber BETWEEN 1 AND 15 THEN
//	    rst = IF(time >= 08:00 AND time < 20:00, 1, 2);
//	    IF day=1  AND time >= 00:00 AND time < 08:00 THEN rst = 2; END IF;
//	    IF day=15 AND time >= 08:00 AND time < 16:00 THEN rst = 1; END IF;
//	    IF day=15 AND time >= 16:00 AND time <= 23:59:59 THEN rst = 2; END IF;
//	ELSEIF @dayNumber BETWEEN 16 AND 31 THEN
//	    rst = IF(time >= 08:00 AND time < 20:00, 2, 1);      -- 注意与上旬相反
//	    lastDay = dayofmonth(LAST_DAY(@localTimestamp));
//	    IF day=16 AND time >= 00:00 AND time < 08:00 THEN rst = 1; END IF;
//	    IF day=lastDay AND time >= 08:00 AND time < 16:00 THEN rst = 2; END IF;
//	    IF day=lastDay AND time >= 16:00 AND time <= 23:59:59 THEN rst = 1; END IF;
//	END IF;
//
// 业务含义：班次按"半月"轮换，上旬与下旬的 08:00–20:00 对应不同班次；
// 且在月份交界处（1 日、15 日、16 日、月末）有 4 条特殊覆盖规则。
//
// 这段逻辑有 10 个分支，边界极容易写错，因此本文件配套了覆盖全部分支的单元测试。
package shift

import "time"

// TeamTurn 是班次号，取值 1 或 2。遗留系统用 1/2 表示两个班组轮换。
type TeamTurn int

const (
	TeamTurnA TeamTurn = 1
	TeamTurnB TeamTurn = 2
)

// BeijingOffset 是北京时间相对 UTC 的偏移。
// 遗留系统硬编码为 `INTERVAL 8 HOUR`；保持不变以免改变既有语义。
const BeijingOffset = 8 * time.Hour

// CalcTeamTurn 从 PLC 时间戳计算班次。
//
// 参数 t 必须是 **UTC** 时间（遗留系统约定：PLC 提供 UTC，函数内部 +8 小时转北京时间）。
// 返回 1 或 2。
func CalcTeamTurn(t time.Time) TeamTurn {
	local := t.UTC().Add(BeijingOffset)

	day := local.Day()
	lastDay := lastDayOfMonth(local)
	hm := local.Hour()*100 + local.Minute() // 与 SQL 的 TIME 比较等价

	if day >= 1 && day <= 15 {
		// 上旬：08:00–20:00 → 班次 1，其余 → 班次 2
		rst := TeamTurnB
		if hm >= 800 && hm < 2000 {
			rst = TeamTurnA
		}
		// 1 日 00:00–08:00 → 2（与默认一致，保留以便对照遗留代码）
		if day == 1 && hm >= 0 && hm < 800 {
			rst = TeamTurnB
		}
		// 15 日 08:00–16:00 → 1
		if day == 15 && hm >= 800 && hm < 1600 {
			rst = TeamTurnA
		}
		// 15 日 16:00–24:00 → 2
		if day == 15 && hm >= 1600 && hm <= 2359 {
			rst = TeamTurnB
		}
		return rst
	}

	// 下旬（16 日–月末）：08:00–20:00 → 班次 2，其余 → 班次 1（与上旬相反）
	rst := TeamTurnA
	if hm >= 800 && hm < 2000 {
		rst = TeamTurnB
	}
	// 16 日 00:00–08:00 → 1
	if day == 16 && hm >= 0 && hm < 800 {
		rst = TeamTurnA
	}
	// 月末 08:00–16:00 → 2
	if day == lastDay && hm >= 800 && hm < 1600 {
		rst = TeamTurnB
	}
	// 月末 16:00–24:00 → 1
	if day == lastDay && hm >= 1600 && hm <= 2359 {
		rst = TeamTurnA
	}
	return rst
}

func lastDayOfMonth(t time.Time) int {
	// 当月 1 日 +1 月 -1 天，避免手动处理闰年
	firstOfNext := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()).AddDate(0, 1, 0)
	return firstOfNext.AddDate(0, 0, -1).Day()
}
