package plc

import (
	"math"
	"testing"
)

func TestDecodeValue_Integers(t *testing.T) {
	// 大端：0x12 0x34 = 4660
	raw := []byte{0x12, 0x34, 0xFF, 0xFE, 0x80, 0x00, 0x00, 0x01}

	cases := []struct {
		name   string
		typ    string
		offset int
		want   any
	}{
		{"uint8", "uint8", 0, byte(0x12)},
		{"int8", "int8", 0, int8(0x12)},
		{"int8 negative", "int8", 1, int8(0x34)},
		{"uint16", "uint16", 0, uint16(0x1234)},
		{"int16", "int16", 0, int16(0x1234)},
		{"uint16 high", "uint16", 2, uint16(0xFFFE)},
		{"int16 negative", "int16", 2, int16(-2)},
		{"uint32", "uint32", 0, uint32(0x1234FFFE)},
		{"int32 negative", "int32", 4, int32(-2147483647)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DecodeValue(raw[tc.offset:], tc.typ, 0, 0)
			if err != nil {
				t.Fatalf("DecodeValue: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v (%T), want %v (%T)", got, got, tc.want, tc.want)
			}
		})
	}
}

func TestDecodeValue_Float(t *testing.T) {
	buf := make([]byte, 4)
	bits := math.Float32bits(123.5)
	buf[0] = byte(bits >> 24)
	buf[1] = byte(bits >> 16)
	buf[2] = byte(bits >> 8)
	buf[3] = byte(bits)

	got, err := DecodeValue(buf, "float32", 0, 0)
	if err != nil {
		t.Fatalf("DecodeValue: %v", err)
	}
	if got != float32(123.5) {
		t.Fatalf("got %v, want 123.5", got)
	}
}

func TestDecodeValue_Bool(t *testing.T) {
	raw := []byte{0b0000_0101}
	for _, tc := range []struct {
		bit  int
		want bool
	}{{0, true}, {1, false}, {2, true}, {3, false}} {
		got, err := DecodeValue(raw, "bool", 0, tc.bit)
		if err != nil {
			t.Fatalf("bit %d: %v", tc.bit, err)
		}
		if got != tc.want {
			t.Fatalf("bit %d = %v, want %v", tc.bit, got, tc.want)
		}
	}
}

func TestDecodeValue_String(t *testing.T) {
	cases := []struct {
		name string
		raw  []byte
		len  int
		want string
	}{
		{"nul terminated", []byte("ABC\x00XYZW"), 8, "ABC"},
		{"nul at end", []byte("ABCDEFG\x00"), 8, "ABCDEFG"},
		{"space padded", []byte("ABC     "), 8, "ABC"},
		{"trim both", []byte("  ABC   "), 8, "ABC"},
		{"all nul", []byte{0, 0, 0, 0}, 4, ""},
		{"chinese utf8", []byte("你好      "), 10, "你好"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DecodeValue(tc.raw, "string", tc.len, 0)
			if err != nil {
				t.Fatalf("DecodeValue: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDecodeValue_Errors(t *testing.T) {
	cases := []struct {
		name string
		raw  []byte
		typ  string
		len  int
		bit  int
	}{
		{"short int32", []byte{1, 2}, "int32", 0, 0},
		{"short string", []byte("ABC"), "string", 8, 0},
		{"zero string length", []byte("ABC"), "string", 0, 0},
		{"bad bit", []byte{1}, "bool", 0, 9},
		{"unknown type", []byte{1}, "weird", 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := DecodeValue(tc.raw, tc.typ, tc.len, tc.bit); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

// TestDecodeValue_RealTMTLayout 用真实 UDT 布局做端到端解码验证。
func TestDecodeValue_RealTMTLayout(t *testing.T) {
	// 构造一个 92 字节的 TMT UDT 数据块
	buf := make([]byte, 92)
	copy(buf[0:], []byte("POS-A   ")) // POSITION_Name @0 len 8
	buf[8], buf[9] = 0x00, 0x2A       // Doff_No @8 WORD = 42
	// EndTimeOfPack @15 DWORD
	buf[15], buf[16], buf[17], buf[18] = 0x00, 0x0F, 0x42, 0x40
	copy(buf[34:], []byte("120/48      ")) // Yarn_Type @34 len 12
	copy(buf[71:], []byte("WDC-43L "))     // MachineName @71 len 8
	copy(buf[79:], []byte("G2091200482S")) // CodeNumber @79 len 12

	checks := []struct {
		field  string
		offset int
		typ    string
		length int
		want   any
	}{
		{"POSITION_Name", 0, "string", 8, "POS-A"},
		{"Doff_No", 8, "uint16", 0, uint16(42)},
		{"EndTimeOfPack", 15, "uint32", 0, uint32(1000000)},
		{"Yarn_Type", 34, "string", 12, "120/48"},
		{"MachineName", 71, "string", 8, "WDC-43L"},
		{"CodeNumber", 79, "string", 12, "G2091200482S"},
	}
	for _, c := range checks {
		got, err := DecodeValue(buf[c.offset:], c.typ, c.length, 0)
		if err != nil {
			t.Fatalf("%s: %v", c.field, err)
		}
		if got != c.want {
			t.Fatalf("%s = %v (%T), want %v (%T)", c.field, got, got, c.want, c.want)
		}
	}
}
