package printer

import (
	"encoding/hex"
	"testing"
)

// TestBuildLabelFrame_MatchesLegacy 逐字节比对旧系统 MacsaPrinter.js 的组帧。
//
// 旧实现：
//
//	const hexLabelName = stringToHexString(labelName.replace(/\.[^/.]+$/, '')).substring(0,16).padEnd(16,'00');
//	const command = `02162D00${MODE}${COPIES}${BATCH}${hexLabelName}03`;
func TestBuildLabelFrame_MatchesLegacy(t *testing.T) {
	cases := []struct {
		name  string
		label string
		// 期望的 8 字节名称段（hex）
		wantNameHex string
	}{
		{"去掉扩展名", "mae3.lm1", "6d61653300000000"},        // "mae3"
		{"无扩展名", "sorting", "736f7274696e6700"},          // "sorting" 7 字节 + 补 1 个 00
		{"超长截断到 8 字节", "abcdefghij.lm1", "6162636465666768"}, // "abcdefgh"
		{"转小写", "MAE3.LM1", "6d61653300000000"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			frame, err := BuildLabelFrame(tc.label)
			if err != nil {
				t.Fatalf("BuildLabelFrame: %v", err)
			}

			// 帧头 4 字节 + 3 个 4 字节 DWORD + 8 字节名称 + 结束符
			if len(frame) != 4+12+8+1 {
				t.Fatalf("frame length = %d, want %d", len(frame), 4+12+8+1)
			}
			if got := hex.EncodeToString(frame[:4]); got != "02162d00" {
				t.Errorf("header = %s, want 02162d00", got)
			}
			if got := hex.EncodeToString(frame[4:16]); got != "000000000000000000000000" {
				t.Errorf("mode/copies/batch = %s, want all zeros", got)
			}
			if got := hex.EncodeToString(frame[16:24]); got != tc.wantNameHex {
				t.Errorf("name = %s, want %s", got, tc.wantNameHex)
			}
			if frame[24] != 0x03 {
				t.Errorf("footer = %02x, want 03", frame[24])
			}
		})
	}
}

// TestBuildVariableFrame_MatchesLegacy 逐字节比对字段写入帧。
//
// 旧实现：
//
//	const DATA_BYTECOUNT_WORD = (2 + len + 1 - 1).toString(16).padStart(2,'0') + '00'
//	const command = `02044101${DATA_BYTECOUNT_WORD}00${FIELD}${hex}03`;
func TestBuildVariableFrame_MatchesLegacy(t *testing.T) {
	cases := []struct {
		name       string
		v          MacsaVariable
		wantCount  byte
		wantSuffix string // 字段号 + 文本 hex
	}{
		// 旧公式：2 + 文本长度 + 字段数 - 1；单字段时即 2 + 文本长度
		{"字段0 短文本", MacsaVariable{Field: "0", Value: "AB"}, 4, "004142"},
		{"字段3 日期时间", MacsaVariable{Field: "3", Value: "2019/07/31"}, 12, "03323031392f30372f3331"},
		{"字段5 纯数字", MacsaVariable{Field: "5", Value: "12345"}, 7, "053132333435"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			frame, err := BuildVariableFrame(tc.v)
			if err != nil {
				t.Fatalf("BuildVariableFrame: %v", err)
			}

			// 02 04 41 01 | count(2) | option(1) | field+text | 03
			if got := hex.EncodeToString(frame[:4]); got != "02044101" {
				t.Errorf("header = %s, want 02044101", got)
			}
			// 字节数：低字节在前的高位写法（旧系统是 padStart(2)+'00'）
			if frame[4] != tc.wantCount {
				t.Errorf("byte count = %d, want %d", frame[4], tc.wantCount)
			}
			if frame[5] != 0x00 {
				t.Errorf("count high byte = %02x, want 00", frame[5])
			}
			if frame[6] != 0x00 {
				t.Errorf("option = %02x, want 00 (write)", frame[6])
			}
			body := hex.EncodeToString(frame[7 : len(frame)-1])
			if body != tc.wantSuffix {
				t.Errorf("field+text = %s, want %s", body, tc.wantSuffix)
			}
			if frame[len(frame)-1] != 0x03 {
				t.Errorf("footer = %02x, want 03", frame[len(frame)-1])
			}
		})
	}
}

// TestBuildVariableFrame_InvalidField 验证非法字段号被拒，而不是写出垃圾帧。
func TestBuildVariableFrame_InvalidField(t *testing.T) {
	for _, f := range []string{"", "abc", "256", "-1"} {
		if _, err := BuildVariableFrame(MacsaVariable{Field: f, Value: "x"}); err == nil {
			t.Errorf("field %q: expected error, got nil", f)
		}
	}
}
