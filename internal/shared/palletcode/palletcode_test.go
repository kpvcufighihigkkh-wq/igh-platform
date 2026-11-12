package palletcode

import (
	"testing"
	"time"
)

// TestBuild_MatchesLegacyFormat 对齐旧库 GET_PALLET_CODE 的输出格式。
//
// 旧库函数体：
//
//	CONCAT(@palletPlantCode, DATE_FORMAT(db.product_date,'%y%m%d'),
//	       @palletizerCode, LPAD(db.daily_id,4,'0'))
//
// 现场的标签模板与条码规则按这个格式做的，改格式等于让旧标签无法识别。
func TestBuild_MatchesLegacyFormat(t *testing.T) {
	date := time.Date(2025, 4, 16, 0, 0, 0, 0, time.Local)

	cases := []struct {
		name       string
		plant      string
		palletizer int
		dailyID    int
		want       string
	}{
		{"常规", "12", 13, 1, "12250416130001"},
		{"两位打包机号不补零", "12", 11, 81, "12250416110081"},
		{"无厂编码", "", 11, 5, "250416110005"},
		{"序号补零到四位", "12", 13, 42, "12250416130042"},
		{"序号超过四位不截断", "12", 13, 12345, "122504161312345"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Build(tc.plant, date, tc.palletizer, tc.dailyID)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			if got != tc.want {
				t.Errorf("Build(%q, %s, %d, %d) = %q, want %q",
					tc.plant, date.Format("2006-01-02"), tc.palletizer, tc.dailyID, got, tc.want)
			}
		})
	}
}

// TestBuild_UsesProductDateNotToday 验证日期取自生产日期。
//
// 跨零点打包时"今天是几号"和"这垛产品是哪天生产的"不同，
// 标签上的日期必须跟产品走，否则追溯时对不上班次。
func TestBuild_UsesProductDateNotToday(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -3)
	got, err := Build("12", yesterday, 13, 7)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	wantPrefix := "12" + yesterday.Format("060102") + "13"
	if len(got) < len(wantPrefix) || got[:len(wantPrefix)] != wantPrefix {
		t.Errorf("Build = %q, 应以 %q 开头（日期段必须是生产日期）", got, wantPrefix)
	}
}

// TestBuild_RejectsIncompleteInput 缺关键字段必须报错，不能编出一个看似正常的码。
//
// 编错了会打印到实体标签上，发现时整垛已经入库，代价远高于当场报错。
func TestBuild_RejectsIncompleteInput(t *testing.T) {
	date := time.Date(2025, 4, 16, 0, 0, 0, 0, time.Local)

	cases := []struct {
		name       string
		productDate time.Time
		palletizer int
		dailyID    int
	}{
		{"缺生产日期", time.Time{}, 13, 1},
		{"缺打包机号", date, 0, 1},
		{"序号为 0", date, 13, 0},
		{"序号为负", date, 13, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := Build("12", tc.productDate, tc.palletizer, tc.dailyID); err == nil {
				t.Errorf("期望报错，实际得到 %q", got)
			}
		})
	}
}

// TestBuild_SameDaySequenceUnique 同一天不同序号必须得到不同编码。
func TestBuild_SameDaySequenceUnique(t *testing.T) {
	date := time.Date(2025, 4, 16, 0, 0, 0, 0, time.Local)
	seen := map[string]bool{}
	for i := 1; i <= 200; i++ {
		code, err := Build("12", date, 13, i)
		if err != nil {
			t.Fatalf("Build(%d): %v", i, err)
		}
		if seen[code] {
			t.Fatalf("序号 %d 生成的编码 %q 重复", i, code)
		}
		seen[code] = true
	}
}
