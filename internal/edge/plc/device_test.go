package plc

import (
	"testing"
)

func TestNewProtocol_S7(t *testing.T) {
	p, err := NewProtocol("s7", "192.168.1.10", 102, 0, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil protocol")
	}
	if _, ok := p.(*s7Protocol); !ok {
		t.Fatalf("expected *s7Protocol, got %T", p)
	}
}

func TestNewProtocol_EmptyDefaultsToS7(t *testing.T) {
	p, err := NewProtocol("", "192.168.1.10", 102, 0, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := p.(*s7Protocol); !ok {
		t.Fatalf("expected *s7Protocol for empty protocol, got %T", p)
	}
}

func TestNewProtocol_Modbus(t *testing.T) {
	p, err := NewProtocol("modbus", "192.168.1.10", 502, 0, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil protocol")
	}
	if _, ok := p.(*modbusProtocol); !ok {
		t.Fatalf("expected *modbusProtocol, got %T", p)
	}
}

func TestNewProtocol_Unsupported(t *testing.T) {
	_, err := NewProtocol("profinet", "192.168.1.10", 102, 0, 0)
	if err == nil {
		t.Fatal("expected error for unsupported protocol")
	}
	if got := err.Error(); got != "unsupported protocol: profinet" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestReadBlock_Fields(t *testing.T) {
	rb := ReadBlock{DB: 1, Start: 0, Size: 100, Tag: "status"}
	if rb.DB != 1 || rb.Start != 0 || rb.Size != 100 || rb.Tag != "status" {
		t.Fatalf("unexpected field values: %+v", rb)
	}
}
