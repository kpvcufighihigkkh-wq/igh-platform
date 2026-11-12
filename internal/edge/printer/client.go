package printer

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

type Printer struct {
	Code string `json:"code"`
	// Device / Head 用于按"设备 + 头号"定位打印机（打标机场景）
	Device    string    `json:"device,omitempty"`
	Head      int       `json:"head,omitempty"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Label     string    `json:"label,omitempty"`
	Online    bool      `json:"online"`
	LastCheck time.Time `json:"last_check,omitempty"`
}

type Client struct {
	printers  map[string]*Printer
	// byHead 按 (device, head) 索引，避免调用方拼接字符串编码
	byHead    map[string]*Printer
	templates map[string]*template.Template
	mu        sync.RWMutex
}

func NewClient() *Client {
	return &Client{
		printers:  make(map[string]*Printer),
		byHead:    make(map[string]*Printer),
		templates: make(map[string]*template.Template),
	}
}

func (c *Client) Register(code, host string, port int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.printers[code] = &Printer{Code: code, Host: host, Port: port}
}

// RegisterFor 注册一台"属于某设备第 head 号头"的打印机（打标机场景）。
//
// 现场按 IP 直配，不依赖任何命名约定：
//
//	printers:
//	  - { device: "sorting-L04", head: 1, host: "192.168.4.78", port: 3490 }
func (c *Client) RegisterFor(code, device string, head int, host string, port int, label string) {
	p := &Printer{
		Code: code, Device: device, Head: head,
		Host: host, Port: port, Label: label,
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.printers[code] = p
	c.byHead[headKeyOf(device, head)] = p
}

// ByHead 按 (设备, 头号) 找打印机。
func (c *Client) ByHead(device string, head int) (*Printer, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.byHead[headKeyOf(device, head)]
	return p, ok
}

func headKeyOf(device string, head int) string {
	return fmt.Sprintf("%s#%d", device, head)
}

func (c *Client) RegisterTemplate(name, zplTemplate string) error {
	t, err := template.New(name).Parse(zplTemplate)
	if err != nil {
		return fmt.Errorf("parse template %s: %w", name, err)
	}
	c.mu.Lock()
	c.templates[name] = t
	c.mu.Unlock()
	return nil
}

// LoadTemplates 从目录加载全部 *.zpl 模板，模板名取文件名（不含扩展名）。
//
// 之前 RegisterTemplate 只有测试在调，生产代码一次都没调过——
// 结果是 /edge/print 传 template_name 必然报 "template not found"，
// 工位页面的"补打标签"按钮点下去一直是失败的。
//
// 目录不存在不算错误：有的工位不打印，没必要强制要求有模板。
// 单个文件解析失败只记录并跳过，不能因为一个坏模板导致边端起不来。
func (c *Client) LoadTemplates(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("读取模板目录 %s 失败: %w", dir, err)
	}

	loaded := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".zpl") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			logger.L.Warnw("读取标签模板失败，跳过", "file", e.Name(), "error", err)
			continue
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		if err := c.RegisterTemplate(name, string(raw)); err != nil {
			logger.L.Warnw("解析标签模板失败，跳过", "file", e.Name(), "error", err)
			continue
		}
		loaded++
	}
	return loaded, nil
}

func (c *Client) Print(printerCode string, zplData []byte) error {
	c.mu.RLock()
	p, ok := c.printers[printerCode]
	c.mu.RUnlock()
	if !ok {
		return fmt.Errorf("printer %s not found", printerCode)
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(p.Host, strconv.Itoa(p.Port)), 3*time.Second)
	if err != nil {
		return fmt.Errorf("connect to printer %s: %w", printerCode, err)
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write(zplData); err != nil {
		return fmt.Errorf("write to printer %s: %w", printerCode, err)
	}

	logger.L.Infow("label printed", "printer", printerCode, "bytes", len(zplData))
	return nil
}

func (c *Client) PrintTemplate(printerCode, templateName string, data any) error {
	c.mu.RLock()
	t, ok := c.templates[templateName]
	c.mu.RUnlock()
	if !ok {
		return fmt.Errorf("template %s not found", templateName)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("render template %s: %w", templateName, err)
	}
	return c.Print(printerCode, buf.Bytes())
}

func (c *Client) CheckAll() {
	c.mu.RLock()
	codes := make([]string, 0, len(c.printers))
	for code := range c.printers {
		codes = append(codes, code)
	}
	c.mu.RUnlock()

	for _, code := range codes {
		c.checkOne(code)
	}
}

func (c *Client) checkOne(code string) {
	c.mu.RLock()
	p, ok := c.printers[code]
	c.mu.RUnlock()
	if !ok {
		return
	}

	addr := net.JoinHostPort(p.Host, strconv.Itoa(p.Port))
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)

	c.mu.Lock()
	p.LastCheck = time.Now()
	if err != nil {
		p.Online = false
	} else {
		p.Online = true
		conn.Close()
	}
	c.mu.Unlock()
}

func (c *Client) Status() []map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]map[string]any, 0, len(c.printers))
	for _, p := range c.printers {
		s := map[string]any{
			"code":   p.Code,
			"host":   p.Host,
			"port":   p.Port,
			"online": p.Online,
		}
		if !p.LastCheck.IsZero() {
			s["last_check"] = p.LastCheck.Format(time.RFC3339)
		}
		result = append(result, s)
	}
	return result
}

func (c *Client) List() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	codes := make([]string, 0, len(c.printers))
	for code := range c.printers {
		codes = append(codes, code)
	}
	return codes
}

// ByCode 按编码查打印机配置。
//
// 栈板标签是按"打包机绑定里的 label_printer 编码"去取的，
// 与打标机的 ByHead 是两种定位方式：一台打包机固定用一台标签机，
// 不存在"同一设备多个头"的概念。
func (c *Client) ByCode(code string) (*Printer, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.printers[code]
	return p, ok
}

func DefaultBobbinLabelTemplate() string {
	return strings.TrimSpace(`
^XA
^FO50,30^A0N,30,30^FD{{.LotCode}}^FS
^FO50,70^A0N,25,25^FDGrade: {{.GradeCode}}^FS
^FO50,100^A0N,25,25^FDWeight: {{.Weight}}kg^FS
^FO50,130^A0N,20,20^FD{{.Timestamp}}^FS
^FO50,160^BY2^BCN,80,Y,N,N^FD{{.BarCode}}^FS
^XZ
`)
}

func DefaultPalletLabelTemplate() string {
	return strings.TrimSpace(`
^XA
^FO50,30^A0N,35,35^FDPallet: {{.PalletCode}}^FS
^FO50,75^A0N,25,25^FDLot: {{.LotCode}}^FS
^FO50,105^A0N,25,25^FDGrade: {{.GradeCode}}^FS
^FO50,135^A0N,25,25^FDCount: {{.BobbinCount}} bobbins^FS
^FO50,165^A0N,25,25^FDWeight: {{.TotalWeight}}kg^FS
^FO50,195^A0N,20,20^FD{{.Timestamp}}^FS
^FO50,230^BY2^BCN,80,Y,N,N^FD{{.BarCode}}^FS
^XZ
`)
}
