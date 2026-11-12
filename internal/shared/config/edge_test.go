package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEdgeHTTP_Addr(t *testing.T) {
	tests := []struct {
		host string
		port int
		want string
	}{
		{"0.0.0.0", 8080, "0.0.0.0:8080"},
		{"127.0.0.1", 3000, "127.0.0.1:3000"},
		{"", 0, ":0"},
	}
	for _, tt := range tests {
		h := EdgeHTTP{Host: tt.host, Port: tt.port}
		if got := h.Addr(); got != tt.want {
			t.Errorf("Addr() = %q, want %q", got, tt.want)
		}
	}
}

func writeEdgeYAML(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "edge.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadEdge_Defaults(t *testing.T) {
	dir := t.TempDir()
	writeEdgeYAML(t, dir, `edge:
  id: "test-node"
  type: "winding"
`)

	cfg, err := LoadEdge(dir)
	if err != nil {
		t.Fatalf("LoadEdge: %v", err)
	}
	if cfg.Edge.ID != "test-node" {
		t.Errorf("ID = %q, want test-node", cfg.Edge.ID)
	}
	if cfg.Edge.DataDir != "./data" {
		t.Errorf("DataDir = %q, want ./data", cfg.Edge.DataDir)
	}
	if cfg.Edge.HTTP.Port != 8080 {
		t.Errorf("HTTP.Port = %d, want 8080", cfg.Edge.HTTP.Port)
	}
	if cfg.Edge.HTTP.Host != "0.0.0.0" {
		t.Errorf("HTTP.Host = %q, want 0.0.0.0", cfg.Edge.HTTP.Host)
	}
	if cfg.Edge.Server.HeartbeatSec != 10 {
		t.Errorf("HeartbeatSec = %d, want 10", cfg.Edge.Server.HeartbeatSec)
	}
	if cfg.Edge.Server.SyncIntervalSec != 30 {
		t.Errorf("SyncIntervalSec = %d, want 30", cfg.Edge.Server.SyncIntervalSec)
	}
	if cfg.Edge.PLC.PollIntervalMs != 1000 {
		t.Errorf("PollIntervalMs = %d, want 1000", cfg.Edge.PLC.PollIntervalMs)
	}
	if cfg.Edge.HA.Mode != "standalone" {
		t.Errorf("HA.Mode = %q, want standalone", cfg.Edge.HA.Mode)
	}
}

func TestLoadEdge_FullConfig(t *testing.T) {
	dir := t.TempDir()
	writeEdgeYAML(t, dir, `edge:
  id: "edge-sort-01"
  type: "sorting"
  data_dir: "/opt/data"
  http:
    host: "192.168.1.5"
    port: 9090
  server:
    url: "http://10.0.0.1:8099"
    api_key: "sk-test"
    heartbeat_sec: 5
    sync_interval_sec: 60
  plc:
    poll_interval_ms: 500
    devices:
      - code: "plc-01"
        host: "192.168.1.10"
        port: 502
        protocol: "modbus"
  printers:
    - code: "lbl-01"
      host: "192.168.1.50"
      port: 9100
  ha:
    mode: "active-standby"
    peer: "192.168.1.22"
    vip: "192.168.1.200"
    priority: 50
  line:
    code: "SORT01"
    name: "Sorting Line"
`)

	cfg, err := LoadEdge(dir)
	if err != nil {
		t.Fatalf("LoadEdge: %v", err)
	}
	if cfg.Edge.ID != "edge-sort-01" {
		t.Errorf("ID = %q", cfg.Edge.ID)
	}
	if cfg.Edge.Type != "sorting" {
		t.Errorf("Type = %q", cfg.Edge.Type)
	}
	if cfg.Edge.DataDir != "/opt/data" {
		t.Errorf("DataDir = %q", cfg.Edge.DataDir)
	}
	if cfg.Edge.HTTP.Port != 9090 {
		t.Errorf("HTTP.Port = %d", cfg.Edge.HTTP.Port)
	}
	if cfg.Edge.Server.URL != "http://10.0.0.1:8099" {
		t.Errorf("Server.URL = %q", cfg.Edge.Server.URL)
	}
	if cfg.Edge.Server.HeartbeatSec != 5 {
		t.Errorf("HeartbeatSec = %d", cfg.Edge.Server.HeartbeatSec)
	}
	if cfg.Edge.PLC.PollIntervalMs != 500 {
		t.Errorf("PollIntervalMs = %d", cfg.Edge.PLC.PollIntervalMs)
	}
	if len(cfg.Edge.PLC.Devices) != 1 {
		t.Fatalf("expected 1 PLC device, got %d", len(cfg.Edge.PLC.Devices))
	}
	if cfg.Edge.PLC.Devices[0].Protocol != "modbus" {
		t.Errorf("Device protocol = %q", cfg.Edge.PLC.Devices[0].Protocol)
	}
	if len(cfg.Edge.Printers) != 1 {
		t.Fatalf("expected 1 printer, got %d", len(cfg.Edge.Printers))
	}
	if cfg.Edge.HA.Mode != "active-standby" {
		t.Errorf("HA.Mode = %q", cfg.Edge.HA.Mode)
	}
	if cfg.Edge.HA.Priority != 50 {
		t.Errorf("HA.Priority = %d", cfg.Edge.HA.Priority)
	}
	if cfg.Edge.Line.Code != "SORT01" {
		t.Errorf("Line.Code = %q", cfg.Edge.Line.Code)
	}
}

func TestLoadEdge_FileNotFound(t *testing.T) {
	_, err := LoadEdge(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestLoadEdge_EnvExpansion(t *testing.T) {
	dir := t.TempDir()
	writeEdgeYAML(t, dir, `edge:
  id: "test"
  type: "winding"
  server:
    url: "${TEST_IGH_SERVER_URL}"
    api_key: "${TEST_IGH_API_KEY}"
`)

	t.Setenv("TEST_IGH_SERVER_URL", "http://10.0.0.5:8099")
	t.Setenv("TEST_IGH_API_KEY", "sk-expanded")
	cfg, err := LoadEdge(dir)
	if err != nil {
		t.Fatalf("LoadEdge: %v", err)
	}
	if cfg.Edge.Server.URL != "http://10.0.0.5:8099" {
		t.Errorf("expected env-expanded URL, got %q", cfg.Edge.Server.URL)
	}
	if cfg.Edge.Server.APIKey != "sk-expanded" {
		t.Errorf("expected env-expanded API key, got %q", cfg.Edge.Server.APIKey)
	}
}
