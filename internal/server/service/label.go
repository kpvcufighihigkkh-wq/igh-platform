package service

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

// 标签打印相关错误。分成具体类型，前端才能区分"模板写错了"和"打印机连不上"，
// 而不是统一弹一个"打印失败"。
var (
	ErrTemplateNotFound = fmt.Errorf("标签模板不存在")
	ErrPrinterNotFound  = fmt.Errorf("打印机不存在")
)

// placeholderRe 匹配模板里的占位符。
//
// 同时接受两种写法：
//   - {{.PalletCode}} —— Go text/template 风格的默认模板用的就是这种
//   - {{PalletCode}}  —— 早期实现的写法
//
// 旧的替换逻辑只认后者，而仓库里的 pallet.zpl 用的是前者，
// 结果是渲染出来的 ZPL 里原样保留着 {{.PalletCode}}，打出来就是废标签。
// 这里把两种都吃掉，避免模板与代码再各改一半。
var placeholderRe = regexp.MustCompile(`\{\{\s*\.?([A-Za-z0-9_]+)\s*\}\}`)

type LabelService struct {
	templateDir string
	printers    []PrinterConfig
	// timeout 是连接与写入的单次上限。打印是现场动作，
	// 卡住超过几秒就该报错让人处理，而不是让 HTTP 请求一直挂着。
	timeout time.Duration
	// dial 可替换，测试时注入内存监听器，避免依赖真实打印机
	dial func(network, addr string, timeout time.Duration) (net.Conn, error)
}

// PrinterConfig 是中心可用的打印机。
type PrinterConfig struct {
	Code string `mapstructure:"code" yaml:"code"`
	Name string `mapstructure:"name" yaml:"name"`
	Host string `mapstructure:"host" yaml:"host"`
	Port int    `mapstructure:"port" yaml:"port"`
}

func NewLabelService(configDir string, printers []PrinterConfig) *LabelService {
	return &LabelService{
		templateDir: filepath.Join(configDir, "labels", "templates"),
		printers:    printers,
		timeout:     5 * time.Second,
		dial:        net.DialTimeout,
	}
}

// SetDialer 替换连接函数，测试用。
func (s *LabelService) SetDialer(fn func(string, string, time.Duration) (net.Conn, error)) {
	s.dial = fn
}

// Printers 返回已配置的打印机列表，供前端下拉选择。
func (s *LabelService) Printers() []PrinterConfig { return s.printers }

// resolvePrinter 按编码或直接地址定位打印机。
//
// 两种用法都支持：
//   - printer_code 走配置，改地址不用改前端
//   - printer_host 直接给 IP，用于临时接一台调试机
func (s *LabelService) resolvePrinter(code, host string, port int) (string, error) {
	if code != "" {
		for _, p := range s.printers {
			if p.Code == code {
				return net.JoinHostPort(p.Host, fmt.Sprint(p.portOr9600())), nil
			}
		}
		return "", fmt.Errorf("%w: %s", ErrPrinterNotFound, code)
	}
	if strings.TrimSpace(host) == "" {
		return "", fmt.Errorf("必须指定 printer_code 或 printer_host")
	}
	if port <= 0 {
		port = 9100
	}
	return net.JoinHostPort(host, fmt.Sprint(port)), nil
}

func (p PrinterConfig) portOr9600() int {
	// Zebra 走 9100；这里不叫 9600，函数名是笔误保护——
	// 现场如果配了别的端口，必须显式写进配置。
	if p.Port <= 0 {
		return 9100
	}
	return p.Port
}

// safeTemplatePath 解析模板路径并阻止目录穿越。
//
// template 直接来自 HTTP 请求体。如果不做限制，传 ../../../configs/project.yaml
// 就能读取任意文件内容并打到打印机上（或者借报错回显文件是否存在）。
func (s *LabelService) safeTemplatePath(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("模板名不能为空")
	}
	// 只允许单层文件名，不允许任何路径分隔符
	if name != filepath.Base(name) || strings.Contains(name, "..") {
		return "", fmt.Errorf("模板名不合法: %q", name)
	}
	path := filepath.Join(s.templateDir, name)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("%w: %s", ErrTemplateNotFound, name)
	}
	return path, nil
}

// Render 渲染模板，不做打印。
//
// 单独暴露出来是为了能测：渲染对不对（占位符有没有替换干净）
// 和打印能不能连通是两件事，混在一起测不出问题在哪。
func (s *LabelService) Render(templateName string, variables map[string]string) ([]byte, error) {
	path, err := s.safeTemplatePath(templateName)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取模板 %s 失败: %w", templateName, err)
	}

	missing := make([]string, 0)
	zpl := placeholderRe.ReplaceAllStringFunc(string(raw), func(m string) string {
		key := placeholderRe.FindStringSubmatch(m)[1]
		v, ok := variables[key]
		if !ok {
			missing = append(missing, key)
			return m
		}
		return v
	})

	// 缺字段直接报错，不要打出一张 "$sxxx" 的废标签：
	// 现场一卷标签纸几百张，打废一张就得人工撕掉，还容易混进成品。
	if len(missing) > 0 {
		return nil, fmt.Errorf("模板 %s 缺少变量: %s", templateName, strings.Join(missing, ", "))
	}

	out := []byte(zpl)
	if !strings.HasPrefix(strings.TrimSpace(zpl), "^XA") {
		return nil, fmt.Errorf("模板 %s 渲染结果不是 ZPL（应以 ^XA 开头）", templateName)
	}
	if !strings.HasSuffix(strings.TrimSpace(zpl), "^XZ") {
		return nil, fmt.Errorf("模板 %s 渲染结果不是完整 ZPL（应以 ^XZ 结尾）", templateName)
	}
	return out, nil
}

// Print 渲染并发送到打印机。
func (s *LabelService) Print(templateName string, variables map[string]string, printerCode, printerHost string, printerPort int) error {
	zpl, err := s.Render(templateName, variables)
	if err != nil {
		return err
	}
	addr, err := s.resolvePrinter(printerCode, printerHost, printerPort)
	if err != nil {
		return err
	}
	if err := s.Send(addr, zpl); err != nil {
		return err
	}
	logger.L.Infow("标签已发送到打印机",
		"template", templateName, "printer", addr, "bytes", len(zpl))
	return nil
}

// Send 把渲染好的内容写到打印机。
//
// 每次新建连接：Zebra 收到 ^XZ 即完成一张；长连接在打印机休眠或重启后
// 会变成半开连接，下一张静默丢失——比报错更难查。
func (s *LabelService) Send(addr string, zpl []byte) error {
	conn, err := s.dial("tcp", addr, s.timeout)
	if err != nil {
		return fmt.Errorf("连接打印机 %s 失败: %w", addr, err)
	}
	defer conn.Close()

	if err := conn.SetWriteDeadline(time.Now().Add(s.timeout)); err != nil {
		return fmt.Errorf("设置写超时失败: %w", err)
	}
	n, err := conn.Write(zpl)
	if err != nil {
		return fmt.Errorf("写入打印机 %s 失败（已写 %d/%d 字节）: %w", addr, n, len(zpl), err)
	}
	if n != len(zpl) {
		// 短写意味着标签不完整，必须报错——静默当成成功会让现场收到半张标签
		return fmt.Errorf("写入打印机 %s 不完整：%d/%d 字节", addr, n, len(zpl))
	}
	return nil
}

func (s *LabelService) ListTemplates() ([]string, error) {
	entries, err := os.ReadDir(s.templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
