package printer

import (
	"net"
	"strings"
	"testing"
	"time"
)

func sampleLabel() PalletLabel {
	return PalletLabel{
		PalletCode:  "12250416130001",
		LotCode:     "HX-2026-001",
		GradeCode:   "AA",
		BobbinCount: 48,
		Layers:      6,
		TotalWeight: 1200.5,
		HasWeight:   true,
		ProductDate: "2025-04-16",
		Palletizer:  13,
	}
}

// TestZebra_RenderDefaultTemplate 验证默认模板能渲染出合法 ZPL。
func TestZebra_RenderDefaultTemplate(t *testing.T) {
	z := NewZebra("127.0.0.1", 9100, time.Second)

	zpl, err := z.Render("", sampleLabel())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	s := string(zpl)
	if !strings.HasPrefix(s, "^XA") {
		t.Errorf("ZPL 应以 ^XA 开头：%q", s[:min(20, len(s))])
	}
	if !strings.HasSuffix(s, "^XZ") {
		t.Errorf("ZPL 应以 ^XZ 结尾")
	}
	if !strings.Contains(s, "12250416130001") {
		t.Errorf("ZPL 里没有栈板编码")
	}
	if !strings.Contains(s, "HX-2026-001") {
		t.Errorf("ZPL 里没有批号")
	}
}

// TestZebra_RenderRejectsMalformedResult 验证渲染结果不合法时拒绝发送。
//
// 一张格式错误的 ZPL 打到实体标签上就是废标签，还浪费耗材。
func TestZebra_RenderRejectsMalformedResult(t *testing.T) {
	z := NewZebra("127.0.0.1", 9100, time.Second)

	cases := []struct {
		name string
		tmpl string
	}{
		{"缺 ^XA", "{{.PalletCode}}^XZ"},
		{"缺 ^XZ", "^XA{{.PalletCode}}"},
		{"完全不是 ZPL", "hello {{.PalletCode}}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := z.Render(tc.tmpl, sampleLabel()); err == nil {
				t.Error("期望报错，实际通过")
			}
		})
	}
}

// TestZebra_RenderRejectsBadTemplate 验证模板语法错误被拒。
func TestZebra_RenderRejectsBadTemplate(t *testing.T) {
	z := NewZebra("127.0.0.1", 9100, time.Second)
	if _, err := z.Render("^XA{{.Invalid^XZ", sampleLabel()); err == nil {
		t.Error("期望报错，实际通过")
	}
}

// TestZebra_SendWritesToSocket 用假打印服务器验证真的写到了 TCP 端口。
func TestZebra_SendWritesToSocket(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	got := make(chan []byte, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 4096)
		n, _ := conn.Read(buf)
		got <- buf[:n]
	}()

	addr := ln.Addr().(*net.TCPAddr)
	z := NewZebra("127.0.0.1", addr.Port, 2*time.Second)

	if err := z.Print("", sampleLabel()); err != nil {
		t.Fatalf("Print: %v", err)
	}

	select {
	case data := <-got:
		if !strings.HasPrefix(string(data), "^XA") {
			t.Errorf("打印机收到的内容不以 ^XA 开头：%q", string(data))
		}
		if !strings.Contains(string(data), "12250416130001") {
			t.Errorf("打印机没收到栈板编码")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("打印机没有收到数据")
	}

	// LastZPL 应保存最后一次发送内容，便于现场排查
	if !strings.Contains(string(z.LastZPL()), "12250416130001") {
		t.Error("LastZPL 未记录发送内容")
	}
}

// TestZebra_SendFailsWhenPrinterDown 验证打印机不在线时报错而不是静默成功。
//
// 静默成功会让操作工以为标签已出，实际一垛没有标签。
func TestZebra_SendFailsWhenPrinterDown(t *testing.T) {
	z := NewZebra("127.0.0.1", 1, 500*time.Millisecond)
	if err := z.Print("", sampleLabel()); err == nil {
		t.Fatal("期望连接失败，实际成功")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
