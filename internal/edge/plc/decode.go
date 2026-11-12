package plc

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// DecodeValue 按点表类型把一个字段从原始字节解码为 Go 值。
//
// 采用西门子 S7 默认的大端字节序。
// 字符串按 S7 的 C 类型处理：以 0x00 或空格结尾，两侧空白被裁剪。
func DecodeValue(raw []byte, typ string, length int, bit int) (any, error) {
	switch typ {
	case "bool":
		if bit < 0 || bit > 7 {
			return nil, fmt.Errorf("invalid bit %d", bit)
		}
		if len(raw) < 1 {
			return false, nil
		}
		return raw[0]&(1<<uint(bit)) != 0, nil
	case "byte", "uint8":
		if len(raw) < 1 {
			return nil, errShortRead(1, len(raw))
		}
		return raw[0], nil
	case "int8":
		if len(raw) < 1 {
			return nil, errShortRead(1, len(raw))
		}
		return int8(raw[0]), nil
	case "int16":
		if len(raw) < 2 {
			return nil, errShortRead(2, len(raw))
		}
		return int16(binary.BigEndian.Uint16(raw)), nil
	case "uint16":
		if len(raw) < 2 {
			return nil, errShortRead(2, len(raw))
		}
		return binary.BigEndian.Uint16(raw), nil
	case "int32":
		if len(raw) < 4 {
			return nil, errShortRead(4, len(raw))
		}
		return int32(binary.BigEndian.Uint32(raw)), nil
	case "uint32":
		if len(raw) < 4 {
			return nil, errShortRead(4, len(raw))
		}
		return binary.BigEndian.Uint32(raw), nil
	case "float32":
		if len(raw) < 4 {
			return nil, errShortRead(4, len(raw))
		}
		return math.Float32frombits(binary.BigEndian.Uint32(raw)), nil
	case "float64":
		if len(raw) < 8 {
			return nil, errShortRead(8, len(raw))
		}
		return math.Float64frombits(binary.BigEndian.Uint64(raw)), nil
	case "string":
		if length <= 0 {
			return nil, fmt.Errorf("string length must be > 0")
		}
		if len(raw) < length {
			return nil, errShortRead(length, len(raw))
		}
		return decodeS7String(raw[:length]), nil
	default:
		return nil, fmt.Errorf("unsupported type %q", typ)
	}
}

// decodeS7String 按 S7 C 字符串语义解码：截断到第一个 0x00，裁剪首尾空白。
func decodeS7String(b []byte) string {
	if i := indexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return strings.TrimSpace(string(b))
}

func indexByte(b []byte, c byte) int {
	for i := range b {
		if b[i] == c {
			return i
		}
	}
	return -1
}

func errShortRead(want, got int) error {
	return fmt.Errorf("short read: need %d bytes, got %d", want, got)
}
