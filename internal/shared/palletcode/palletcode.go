// Package palletcode 生成栈板编码。
//
// 规则移植自旧库 h028.sql 的 GET_PALLET_CODE / GET_DTY_PALLET_CODE 函数：
//
//	CONCAT(厂编码, DATE_FORMAT(product_date,'%y%m%d'), 打包机号, LPAD(daily_id,4,'0'))
//
// 例：厂编码 "12" + 2025-04-16 + 打包机 13 + 当天第 1 垛
//     → "1225041613" + "0001" = "12250416130001"
//
// 注意旧函数里打包机号是整数直接 CONCAT，不补零（11/12/13 两位）；
// 这里保持一致，否则标签条码格式会与现场既有设备不符。
//
// 放在 shared 而不是服务端：边端断网时仍要能出垛打标，
// 编码必须能在边端本地算出来，服务端只负责幂等落库。
package palletcode

import (
	"fmt"
	"strings"
	"time"
)

// DailyIDWidth 是当天序号宽度，对应旧库 LPAD(daily_id,4,'0')。
const DailyIDWidth = 4

// Build 拼出栈板编码。
//
// plantCode 为厂编码（旧库 settings.palletPlantCode），可为空——
// 现场没配厂编码时编码仍然可用，只是少了前缀。
// dailyID 从 1 开始；超过宽度时按实际位数扩展，不截断（宁可编码变长，
// 也不能让同一天两垛拿到同一个编码）。
func Build(plantCode string, productDate time.Time, palletizerCode, dailyID int) (string, error) {
	if productDate.IsZero() {
		return "", fmt.Errorf("栈板编码缺少生产日期")
	}
	if palletizerCode <= 0 {
		return "", fmt.Errorf("栈板编码缺少打包机号")
	}
	if dailyID <= 0 {
		return "", fmt.Errorf("栈板编码的当天序号必须从 1 开始，当前 %d", dailyID)
	}

	seq := fmt.Sprintf("%0*d", DailyIDWidth, dailyID)
	return strings.TrimSpace(plantCode) +
		productDate.Format("060102") +
		fmt.Sprintf("%d", palletizerCode) +
		seq, nil
}
