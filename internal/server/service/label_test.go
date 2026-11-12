package service

import (
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakePrinter 起一个本地 TCP 监听，冒充 Zebra 打印机。
//
// 用真实连接而不是 mock 掉 dial 的返回：打印这条链路最容易出问题的地方
// 恰恰是"字节有没有完整写出去"，mock 掉连 conn 就连这个问题都测不到了。
type fakePrinter struct {
	addr string
	mu   sync.Mutex
	got  []byte
	ln   net.Listener
}

func newFakePrinter(t *testing.T) *fakePrinter {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	fp := &fakePrinter{addr: ln.Addr().String(), ln: ln}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			data, _ := io.ReadAll(conn)
			conn.Close()
			fp.mu.Lock()
			fp.got = append(fp.got, data...)
			fp.mu.Unlock()
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return fp
}

func (fp *fakePrinter) received() []byte {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	return append([]byte(nil), fp.got...)
}

// waitFor 等打印机收到内容（TCP 写完后对端不一定立刻读完）。
func (fp *fakePrinter) waitFor(t *testing.T, min int) []byte {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if d := fp.received(); len(d) >= min {
			return d
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fp.received()
}

func newLabelServiceForTest(t *testing.T, templateText string) *LabelService {
	t.Helper()
	dir := t.TempDir()
	tplDir := filepath.Join(dir, "labels", "templates")
	if err := os.MkdirAll(tplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if templateText != "" {
		if err := os.WriteFile(filepath.Join(tplDir, "test.zpl"), []byte(templateText), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return NewLabelService(dir, []PrinterConfig{
		{Code: "line1", Name: "一号线标签机", Host: "127.0.0.1", Port: 9100},
	})
}

const testTemplate = `^XA
^FO40,40^FDPALLET {{.PalletCode}}^FS
^FO40,100^FDLot: {{LotCode}}^FS
^XZ`

// 核心用例：真的把 ZPL 写到 TCP 对端。
//
// 改造前这个函数只把内容打进日志就 return nil，接口返回 200、
// 实际什么都没打。这条测试就是防它退化回去。
func TestPrint_ReallySendsToPrinter(t *testing.T) {
	fp := newFakePrinter(t)
	svc := newLabelServiceForTest(t, testTemplate)

	err := svc.Print("test.zpl", map[string]string{
		"PalletCode": "P0001",
		"LotCode":    "HX-2026-001",
	}, "", "127.0.0.1", fp.port(t))
	if err != nil {
		t.Fatalf("打印失败: %v", err)
	}

	got := string(fp.waitFor(t, 10))
	if !strings.Contains(got, "PALLET P0001") {
		t.Errorf("打印机收到的内容里没有栈板编码:\n%s", got)
	}
	if !strings.Contains(got, "Lot: HX-2026-001") {
		t.Errorf("打印机收到的内容里没有批号:\n%s", got)
	}
	if strings.Contains(got, "{{") {
		t.Errorf("占位符没有被替换:\n%s", got)
	}
	if !strings.HasPrefix(strings.TrimSpace(got), "^XA") ||
		!strings.HasSuffix(strings.TrimSpace(got), "^XZ") {
		t.Errorf("发出去的不是完整 ZPL:\n%s", got)
	}
}

// 模板里用 {{.Xxx}}（Go 模板风格），变量按 Xxx 传，也必须能替换。
//
// 这是改造前真实存在的 bug：模板写 {{.PalletCode}}，代码只替换
// {{PalletCode}}，结果打出来的是带 {{.PalletCode}} 字样的废标签。
func TestRender_HandlesDotSyntax(t *testing.T) {
	svc := newLabelServiceForTest(t, `^XA^FD{{.PalletCode}}|{{Plain}}^XZ`)

	out, err := svc.Render("test.zpl", map[string]string{"PalletCode": "P1", "Plain": "P2"})
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}
	if string(out) != "^XA^FDP1|P2^XZ" {
		t.Fatalf("渲染结果 = %q", string(out))
	}
}

// 缺变量必须报错，不能打出带占位符的废标签。
func TestRender_MissingVariableFails(t *testing.T) {
	svc := newLabelServiceForTest(t, `^XA^FD{{.PalletCode}}/{{.LotCode}}^XZ`)

	_, err := svc.Render("test.zpl", map[string]string{"PalletCode": "P1"})
	if err == nil {
		t.Fatal("缺变量时应返回错误")
	}
	if !strings.Contains(err.Error(), "LotCode") {
		t.Errorf("错误信息应指出缺哪个变量，实际: %v", err)
	}
}

// 模板名来自 HTTP 请求体，必须挡住路径穿越。
func TestRender_RejectsPathTraversal(t *testing.T) {
	svc := newLabelServiceForTest(t, `^XA^XZ`)

	for _, name := range []string{
		"../../../configs/project.yaml",
		"..\\..\\project.yaml",
		"../test.zpl",
		"/etc/passwd",
	} {
		if _, err := svc.Render(name, nil); err == nil {
			t.Errorf("模板名 %q 应被拒绝", name)
		}
	}
}

func TestRender_TemplateNotFound(t *testing.T) {
	svc := newLabelServiceForTest(t, `^XA^XZ`)
	_, err := svc.Render("nope.zpl", nil)
	if !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("应返回 ErrTemplateNotFound，实际 %v", err)
	}
}

// 渲染结果不是 ZPL 时拒绝发送：打到实体标签上就是废标签，还浪费耗材。
func TestRender_RejectsNonZPL(t *testing.T) {
	svc := newLabelServiceForTest(t, `not a zpl`)
	if _, err := svc.Render("test.zpl", nil); err == nil {
		t.Fatal("非 ZPL 内容应被拒绝")
	}
}

// 按编码找打印机，而不是要求前端写死 IP。
func TestPrint_ResolvesPrinterByCode(t *testing.T) {
	fp := newFakePrinter(t)
	svc := newLabelServiceForTest(t, `^XA^FDX^XZ`)
	// 把配置里的打印机指向 fakePrinter
	svc.printers = []PrinterConfig{
		{Code: "line1", Name: "一号线", Host: "127.0.0.1", Port: fp.port(t)},
	}

	if err := svc.Print("test.zpl", nil, "line1", "", 0); err != nil {
		t.Fatalf("按编码打印失败: %v", err)
	}
	if got := string(fp.waitFor(t, 5)); !strings.Contains(got, "^XA") {
		t.Fatalf("打印机没收到内容: %q", got)
	}
}

func TestPrint_UnknownPrinterCode(t *testing.T) {
	svc := newLabelServiceForTest(t, `^XA^XZ`)
	err := svc.Print("test.zpl", nil, "no-such", "", 0)
	if !errors.Is(err, ErrPrinterNotFound) {
		t.Fatalf("应返回 ErrPrinterNotFound，实际 %v", err)
	}
}

// 打印机连不上时必须报错，不能假装成功。
func TestPrint_PrinterUnreachable(t *testing.T) {
	svc := newLabelServiceForTest(t, `^XA^XZ`)
	svc.timeout = 300 * time.Millisecond

	// 这个端口先监听再关掉，拿到一个确定没人监听的地址
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()

	host, port, _ := net.SplitHostPort(addr)
	var p int
	for _, c := range port {
		p = p*10 + int(c-'0')
	}

	err := svc.Print("test.zpl", nil, "", host, p)
	if err == nil {
		t.Fatal("打印机不可达时必须返回错误，否则现场会以为已经打出来了")
	}
	if !strings.Contains(err.Error(), "连接打印机") {
		t.Errorf("错误信息应说明是连接失败，实际: %v", err)
	}
}

func TestListTemplates(t *testing.T) {
	svc := newLabelServiceForTest(t, `^XA^XZ`)
	names, err := svc.ListTemplates()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "test.zpl" {
		t.Fatalf("模板列表 = %v", names)
	}
}

// port 解析 fakePrinter 的随机端口。
func (fp *fakePrinter) port(t *testing.T) int {
	t.Helper()
	_, portStr, err := net.SplitHostPort(fp.addr)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, c := range portStr {
		n = n*10 + int(c-'0')
	}
	return n
}
