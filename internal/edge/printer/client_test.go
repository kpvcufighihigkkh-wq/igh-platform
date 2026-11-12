package printer

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

func TestMain(m *testing.M) {
	logger.Init("debug")
	os.Exit(m.Run())
}

func TestRegister_And_List(t *testing.T) {
	c := NewClient()
	c.Register("p1", "192.168.1.50", 9100)
	c.Register("p2", "192.168.1.51", 9100)

	codes := c.List()
	if len(codes) != 2 {
		t.Fatalf("expected 2 printers, got %d", len(codes))
	}
}

func TestRegisterTemplate_InvalidSyntax(t *testing.T) {
	c := NewClient()
	err := c.RegisterTemplate("bad", "{{.Invalid")
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
}

func TestRegisterTemplate_Valid(t *testing.T) {
	c := NewClient()
	err := c.RegisterTemplate("bobbin", DefaultBobbinLabelTemplate())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrint_UnknownPrinter(t *testing.T) {
	c := NewClient()
	err := c.Print("unknown", []byte("^XA^XZ"))
	if err == nil {
		t.Fatal("expected error for unknown printer")
	}
}

func TestPrintTemplate_UnknownTemplate(t *testing.T) {
	c := NewClient()
	c.Register("p1", "127.0.0.1", 9100)
	err := c.PrintTemplate("p1", "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown template")
	}
}

func TestPrint_TCP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	received := make(chan []byte, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 4096)
		n, _ := conn.Read(buf)
		received <- buf[:n]
	}()

	addr := ln.Addr().(*net.TCPAddr)
	c := NewClient()
	c.Register("tcp-test", "127.0.0.1", addr.Port)

	zpl := []byte("^XA^FO50,30^A0N,30,30^FDHello^FS^XZ")
	if err := c.Print("tcp-test", zpl); err != nil {
		t.Fatalf("print failed: %v", err)
	}

	select {
	case data := <-received:
		if string(data) != string(zpl) {
			t.Fatalf("received %q, want %q", data, zpl)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for data")
	}
}

func TestPrintTemplate_Renders(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	received := make(chan []byte, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 4096)
		n, _ := conn.Read(buf)
		received <- buf[:n]
	}()

	addr := ln.Addr().(*net.TCPAddr)
	c := NewClient()
	c.Register("tpl-test", "127.0.0.1", addr.Port)
	c.RegisterTemplate("bobbin", DefaultBobbinLabelTemplate())

	data := map[string]any{
		"LotCode":   "FDY-001",
		"GradeCode": "AAA",
		"Weight":    "3.500",
		"Timestamp": "2024-06-15",
		"BarCode":   "FDY001AAA",
	}
	if err := c.PrintTemplate("tpl-test", "bobbin", data); err != nil {
		t.Fatalf("print template failed: %v", err)
	}

	select {
	case got := <-received:
		s := string(got)
		if len(s) == 0 {
			t.Fatal("received empty data")
		}
		// verify key fields present in rendered ZPL
		for _, want := range []string{"FDY-001", "AAA", "3.500", "FDY001AAA"} {
			found := false
			for i := 0; i <= len(s)-len(want); i++ {
				if s[i:i+len(want)] == want {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("rendered ZPL missing %q", want)
			}
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for rendered data")
	}
}

func TestStatus_ReflectsRegistered(t *testing.T) {
	c := NewClient()
	c.Register("p1", "10.0.0.1", 9100)

	status := c.Status()
	if len(status) != 1 {
		t.Fatalf("expected 1 status entry, got %d", len(status))
	}
	if status[0]["code"] != "p1" {
		t.Errorf("code = %v", status[0]["code"])
	}
	if status[0]["online"] != false {
		t.Error("expected offline by default")
	}
}

func TestDefaultTemplates_NotEmpty(t *testing.T) {
	if len(DefaultBobbinLabelTemplate()) == 0 {
		t.Error("bobbin template empty")
	}
	if len(DefaultPalletLabelTemplate()) == 0 {
		t.Error("pallet template empty")
	}
}
