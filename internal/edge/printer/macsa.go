package printer

import (
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

// Macsa 是 Macsa 激光打标机的二进制协议驱动。
//
// 协议来源：遗留系统 O17006/src/MacsaPrinter.js（已逐个字节核对）。
// 与本项目其它打印机（Zebra ZPL 文本流）不同，Macsa 是二进制帧，
// 每一步都要精确控制字节内容，因此单独成文件而不是复用 Print()。
//
// 帧格式：
//
//	选标模板  02 16 2D 00 | MODE(4) | COPIES(4) | BATCH(4) | 模板名(8字节hex) | 03
//	写字段    02 04 41 01 | 字节数(2) | 00(写) | 字段号 | 文本hex | 03
type Macsa struct {
	conn *net.TCPConn
	host string
	port int
}

// MacsaVariable 是一条要写入模板的字段：模板里的字段号 + 文本值。
type MacsaVariable struct {
	// Field 是模板中的字段号（十进制字符串，如 "0" / "3" / "4" / "5"）
	Field string
	Value string
}

// NewMacsa 建立到打标机的连接。
func NewMacsa(host string, port int) (*Macsa, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("connect macsa %s: %w", addr, err)
	}
	tcp, ok := conn.(*net.TCPConn)
	if !ok {
		conn.Close()
		return nil, fmt.Errorf("connect macsa %s: unexpected conn type", addr)
	}
	tcp.SetKeepAlive(true)
	return &Macsa{conn: tcp, host: host, port: port}, nil
}

func (m *Macsa) Close() error {
	if m.conn == nil {
		return nil
	}
	return m.conn.Close()
}

// Print 打印一版标签：先选模板，再逐条写入字段。
//
// 必须按旧系统的方式**逐条**发送字段（不是一次发完）：
// 打标机固件按帧处理，一条帧里放多个字段会导致后发的字段覆盖先发的。
func (m *Macsa) Print(labelName string, vars []MacsaVariable) error {
	if m.conn == nil {
		return fmt.Errorf("macsa: not connected")
	}

	if err := m.sendLabel(labelName); err != nil {
		return err
	}
	for _, v := range vars {
		if err := m.sendVariable(v); err != nil {
			return err
		}
	}

	// 旧系统固定等待 1.5s 让打印完成；这里保持同样的节奏，
	// 因为协议层没有可等待的确认应答（源码中该段被注释掉了）。
	time.Sleep(1500 * time.Millisecond)

	logger.L.Infow("macsa label printed",
		"host", m.host, "port", m.port, "label", labelName, "fields", len(vars))
	return nil
}

// BuildLabelFrame 构造"选择模板"帧。
//
//	02 16 2D 00 | MODE(4B) | COPIES(4B) | BATCH(4B) | 名称(8B) | 03
//
// 模板名规则（来自旧系统）：去掉扩展名、转小写、最多 8 字节，右侧补 00。
// 单独成函数是为了能逐字节测试，不必连真机。
func BuildLabelFrame(labelName string) ([]byte, error) {
	name := strings.ToLower(labelName)
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[:i]
	}
	if len(name) > 8 {
		name = name[:8]
	}
	nameHex := hex.EncodeToString([]byte(name))
	for len(nameHex) < 16 {
		nameHex += "00"
	}

	frame := "02162D00" +
		"00000000" + // MODE_DWORD
		"00000000" + // COPIES_DWORD
		"00000000" + // BATCH_DWORD
		nameHex +
		"03"

	buf, err := hex.DecodeString(frame)
	if err != nil {
		return nil, fmt.Errorf("macsa: build label frame: %w", err)
	}
	return buf, nil
}

// BuildVariableFrame 构造"写入单个字段"帧。
//
//	02 04 41 01 | 数据字节数(2B) | 00 | 字段号 | 文本hex | 03
//
// 字节数 = 2 + 文本长度 + 字段数 - 1，与旧系统一致
// （2 = 字段号字节 + 尾部分隔，-1 因为最后一条没有分隔符）。
func BuildVariableFrame(v MacsaVariable) ([]byte, error) {
	fieldNo, err := parseFieldNo(v.Field)
	if err != nil {
		return nil, err
	}

	textHex := hex.EncodeToString([]byte(v.Value))
	byteCount := 2 + len(v.Value) + 1 - 1

	frame := "02044101" +
		fmt.Sprintf("%02X00", byteCount) +
		"00" + // OPTION_BYTE: 00 = write
		fmt.Sprintf("%02X", fieldNo) +
		textHex +
		"03"

	buf, err := hex.DecodeString(frame)
	if err != nil {
		return nil, fmt.Errorf("macsa: build variable frame: %w", err)
	}
	return buf, nil
}

// sendLabel 发送"选择模板"帧。
func (m *Macsa) sendLabel(labelName string) error {
	buf, err := BuildLabelFrame(labelName)
	if err != nil {
		return err
	}
	return m.write(buf)
}

// sendVariable 发送"写入单个字段"帧。
func (m *Macsa) sendVariable(v MacsaVariable) error {
	buf, err := BuildVariableFrame(v)
	if err != nil {
		return err
	}
	return m.write(buf)
}

func (m *Macsa) write(buf []byte) error {
	m.conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	if _, err := m.conn.Write(buf); err != nil {
		return fmt.Errorf("macsa: write: %w", err)
	}
	return nil
}

// parseFieldNo 把模板字段号（十进制字符串）转成字节。
func parseFieldNo(s string) (byte, error) {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0, fmt.Errorf("macsa: invalid field number %q", s)
	}
	if n < 0 || n > 255 {
		return 0, fmt.Errorf("macsa: field number %d out of range", n)
	}
	return byte(n), nil
}
