package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile 在测试目录里写一个文件。
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// copyInstances 把仓库里的真实点表复制到测试目录，保持相对路径结构。
func copyInstances(t *testing.T, dstDir string) {
	t.Helper()
	src := filepath.Join(instancesDir(), "spinning_sides.yaml")
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	writeFile(t, dstDir, "devices/instances/spinning_sides.yaml", string(raw))
}

// TestMergeInstanceFiles_AddsDevices 验证点表里的设备被合并进 edge 配置。
func TestMergeInstanceFiles_AddsDevices(t *testing.T) {
	dir := t.TempDir()
	copyInstances(t, dir)
	writeFile(t, dir, "edge.yaml", `edge:
  id: "edge-winding-01"
  type: "winding"
  plc:
    poll_interval_ms: 500
    instance_files:
      - "devices/instances/spinning_sides.yaml"
`)

	cfg, err := LoadEdge(dir)
	if err != nil {
		t.Fatalf("LoadEdge: %v", err)
	}
	if len(cfg.Edge.PLC.Devices) != 8 {
		t.Fatalf("devices = %d, want 8", len(cfg.Edge.PLC.Devices))
	}
	if cfg.Edge.PLC.Devices[0].Code != "winder-L01" {
		t.Fatalf("first device = %q, want winder-L01", cfg.Edge.PLC.Devices[0].Code)
	}
	if cfg.Edge.PLC.Devices[0].Twin == nil {
		t.Fatal("merged device should carry twin layout")
	}
	if len(cfg.Edge.PLC.Devices[0].PollGroups) != 1 {
		t.Fatal("merged device should carry poll_groups")
	}
}

// TestMergeInstanceFiles_KeepsConnectionFromEdge 验证连接参数以 edge.yaml 为准。
func TestMergeInstanceFiles_KeepsConnectionFromEdge(t *testing.T) {
	dir := t.TempDir()
	copyInstances(t, dir)
	writeFile(t, dir, "edge.yaml", `edge:
  id: "edge-winding-01"
  type: "winding"
  plc:
    devices:
      - code: "winder-L01"
        host: "10.9.9.9"
        port: 102
        protocol: "simulation"
    instance_files:
      - "devices/instances/spinning_sides.yaml"
`)

	cfg, err := LoadEdge(dir)
	if err != nil {
		t.Fatalf("LoadEdge: %v", err)
	}
	var found *PLCDevice
	for i := range cfg.Edge.PLC.Devices {
		if cfg.Edge.PLC.Devices[i].Code == "winder-L01" {
			found = &cfg.Edge.PLC.Devices[i]
			break
		}
	}
	if found == nil {
		t.Fatal("winder-L01 not found")
	}
	// 连接参数保留 edge.yaml 的覆盖
	if found.Host != "10.9.9.9" {
		t.Errorf("host = %q, want 10.9.9.9", found.Host)
	}
	if found.Protocol != "simulation" {
		t.Errorf("protocol = %q, want simulation", found.Protocol)
	}
	// 布局来自点表
	if found.Twin == nil || found.Twin.Stride != 198 {
		t.Errorf("twin layout not merged: %+v", found.Twin)
	}
	// 不应出现重复的同 code 设备
	count := 0
	for _, d := range cfg.Edge.PLC.Devices {
		if d.Code == "winder-L01" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("winder-L01 appears %d times, want 1", count)
	}
}

// TestMergeInstanceFiles_StrictDBMismatch 验证 strict 模式下 DB 号不一致会失败。
func TestMergeInstanceFiles_StrictDBMismatch(t *testing.T) {
	dir := t.TempDir()
	copyInstances(t, dir)
	writeFile(t, dir, "edge.yaml", `edge:
  id: "edge-winding-01"
  type: "winding"
  plc:
    strict: true
    devices:
      - code: "winder-L01"
        host: "10.9.9.9"
        port: 102
        poll_groups:
          - name: "g"
            db: 999
            length: 198
            repeat: { count: 1, stride: 198 }
    instance_files:
      - "devices/instances/spinning_sides.yaml"
`)

	_, err := LoadEdge(dir)
	if err == nil {
		t.Fatal("expected error for DB mismatch in strict mode")
	}
}

// TestMergeInstanceFiles_StrictDBMatch 验证 strict 模式下 DB 号一致可通过。
func TestMergeInstanceFiles_StrictDBMatch(t *testing.T) {
	dir := t.TempDir()
	copyInstances(t, dir)
	writeFile(t, dir, "edge.yaml", `edge:
  id: "edge-winding-01"
  type: "winding"
  plc:
    strict: true
    devices:
      - code: "winder-L01"
        host: "10.9.9.9"
        port: 102
        poll_groups:
          - name: "g"
            db: 555
            length: 198
            repeat: { count: 1, stride: 198 }
    instance_files:
      - "devices/instances/spinning_sides.yaml"
`)

	if _, err := LoadEdge(dir); err != nil {
		t.Fatalf("LoadEdge: %v", err)
	}
}

// TestMergeInstanceFiles_MissingFile 验证引用不存在的点表会报错。
func TestMergeInstanceFiles_MissingFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "edge.yaml", `edge:
  id: "edge-winding-01"
  type: "winding"
  plc:
    instance_files:
      - "devices/instances/nope.yaml"
`)

	_, err := LoadEdge(dir)
	if err == nil {
		t.Fatal("expected error for missing instance file")
	}
}

// TestMergeInstanceFiles_Disabled 验证不配置 instance_files 时行为不变。
func TestMergeInstanceFiles_Disabled(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "edge.yaml", `edge:
  id: "edge-sim-01"
  type: "winding"
  plc:
    devices:
      - code: "sim-1"
        host: "127.0.0.1"
        port: 102
        protocol: "simulation"
`)

	cfg, err := LoadEdge(dir)
	if err != nil {
		t.Fatalf("LoadEdge: %v", err)
	}
	if len(cfg.Edge.PLC.Devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(cfg.Edge.PLC.Devices))
	}
}
