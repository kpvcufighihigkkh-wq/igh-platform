package printer

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"text/template"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

// Zebra 是 Zebra 标签打印机的驱动（ZPL 文本流 over TCP 9100）。
//
// 与 Macsa 打标机的区别：
//   - Macsa 是二进制帧协议，每个字段单独一帧，用于丝饼打标
//   - Zebra 直接吃 ZPL 文本，一条 TCP 连接写完整段 ZPL 即打印，
//     用于栈板标签（旧系统 palletizers.settings 里 firstPrinter/lastPrinter 就是这种）
//
// 两者帧格式完全不同，因此不复用 Print() 的模板机制：
// Zebra 需要按"标签内容 → ZPL"渲染，渲染失败必须报错而不是打出一张空标签。
type Zebra struct {
	host    string
	port    int
	timeout time.Duration
	// lastZPL 保存最近一次发送的内容，便于现场排查"到底发了什么"
	lastZPL []byte
}

// PalletLabel 是栈板标签的字段集合。
//
// 字段与旧系统栈板标签一致：编码、批号、等级、锭数、重量、日期。
// Timestamp / BarCode 是默认模板已有的两行：打印时间与条码内容。
// 条码默认用栈板编码——现场扫码枪读的就是它，和编码保持同源。
type PalletLabel struct {
	PalletCode  string
	LotCode     string
	GradeCode   string
	BobbinCount int
	Layers      int
	TotalWeight float64
	HasWeight   bool
	ProductDate string
	Palletizer  int
	// Timestamp 打印时刻（模板里显示"什么时候打的"）
	Timestamp string
	// BarCode 条码内容；留空时用 PalletCode
	BarCode string
}

// NewZebra 创建驱动。timeout 为 0 时用 5 秒。
func NewZebra(host string, port int, timeout time.Duration) *Zebra {
	if port == 0 {
		port = 9100
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Zebra{host: host, port: port, timeout: timeout}
}

func (z *Zebra) String() string {
	return fmt.Sprintf("%s:%d", z.host, z.port)
}

// LastZPL 返回最近一次发送的 ZPL，测试与现场排查用。
func (z *Zebra) LastZPL() []byte {
	return append([]byte(nil), z.lastZPL...)
}

// Render 按模板渲染 ZPL。
//
// templateText 为 Go text/template 语法（{{.PalletCode}} 等）。
// 渲染后必须仍以 ^XA 开头、含 ^XZ 结尾，否则拒绝发送——
// 一张格式错误的 ZPL 打到实体标签上就是废标签，还会浪费耗材。
func (z *Zebra) Render(templateText string, l PalletLabel) ([]byte, error) {
	if strings.TrimSpace(templateText) == "" {
		templateText = DefaultPalletLabelTemplate()
	}
	// 条码与打印时间的缺省值在这里补，而不是要求每个调用点都填：
	// 漏填会让模板渲染失败或打出空条码，现场扫码枪直接读不出来。
	if strings.TrimSpace(l.BarCode) == "" {
		l.BarCode = l.PalletCode
	}
	if strings.TrimSpace(l.Timestamp) == "" {
		l.Timestamp = time.Now().Format("2006/01/02 15:04")
	}

	t, err := template.New("pallet").Option("missingkey=error").Parse(templateText)
	if err != nil {
		return nil, fmt.Errorf("zebra: 解析标签模板失败: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, l); err != nil {
		return nil, fmt.Errorf("zebra: 渲染标签模板失败: %w", err)
	}

	zpl := bytes.TrimSpace(buf.Bytes())
	if !bytes.HasPrefix(zpl, []byte("^XA")) {
		return nil, fmt.Errorf("zebra: 渲染结果不以 ^XA 开头，拒绝发送")
	}
	if !bytes.HasSuffix(zpl, []byte("^XZ")) {
		return nil, fmt.Errorf("zebra: 渲染结果不以 ^XZ 结尾，拒绝发送")
	}
	return zpl, nil
}

// Print 渲染并发送一张栈板标签。
func (z *Zebra) Print(templateText string, l PalletLabel) error {
	zpl, err := z.Render(templateText, l)
	if err != nil {
		return err
	}
	return z.Send(zpl)
}

// Send 把一段 ZPL 写到打印机。
//
// 每次新建连接：Zebra 收到 ^XZ 即完成一张，长连接反而会在
// 打印机休眠/重启后变成半开连接，下一张静默丢失。
func (z *Zebra) Send(zpl []byte) error {
	addr := z.String()
	conn, err := net.DialTimeout("tcp", addr, z.timeout)
	if err != nil {
		return fmt.Errorf("zebra: 连接 %s 失败: %w", addr, err)
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(z.timeout))
	if _, err := conn.Write(zpl); err != nil {
		return fmt.Errorf("zebra: 写入 %s 失败: %w", addr, err)
	}

	z.lastZPL = append([]byte(nil), zpl...)
	logger.L.Infow("zebra pallet label sent",
		"printer", addr, "pallet", l0(zpl), "bytes", len(zpl))
	return nil
}

// l0 取第一行 ^FD 内容用于日志，避免把整张 ZPL 打进日志。
func l0(zpl []byte) string {
	s := string(zpl)
	if i := strings.Index(s, "^FD"); i >= 0 {
		rest := s[i+3:]
		if j := strings.Index(rest, "^FS"); j >= 0 {
			return rest[:j]
		}
	}
	return ""
}
