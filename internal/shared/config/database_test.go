package config

import (
	"strings"
	"testing"
)

// 单地址必须和改动前逐字一致：现有部署全靠它，多一个参数都可能改变行为。
func TestDatabaseDSN_SingleHostUnchanged(t *testing.T) {
	d := Database{Host: "127.0.0.1", Port: 5432, Name: "igh_platform", User: "igh", Password: "pw"}
	want := "host=127.0.0.1 port=5432 user=igh password=pw dbname=igh_platform sslmode=disable"
	if got := d.DSN(); got != want {
		t.Fatalf("DSN() =\n  %q\nwant\n  %q", got, want)
	}
	if d.MultiHost() {
		t.Fatal("没配 hosts 时不应视为多地址")
	}
}

// 多地址：主机列表与端口列表一一对应，并且只接受可写库。
func TestDatabaseDSN_MultiHost(t *testing.T) {
	d := Database{
		Hosts: []string{"10.0.0.1:15432", "10.0.0.2:15433"},
		Port:  5432, Name: "igh_platform", User: "igh", Password: "pw",
	}
	got := d.DSN()

	for _, part := range []string{
		"host=10.0.0.1,10.0.0.2",
		"port=15432,15433",
		// 备库与被围栏的旧主库都会报 transaction_read_only=on，借此排除
		"target_session_attrs=read-write",
		// 某台机器整机掉电时 TCP 会挂到超时，不设上限会拖垮切换时间
		"connect_timeout=2",
	} {
		if !strings.Contains(got, part) {
			t.Errorf("DSN 缺少 %q\n  got: %s", part, got)
		}
	}
	if !d.MultiHost() {
		t.Fatal("配了 2 个 hosts 应视为多地址")
	}
}

// 没写端口的地址用 database.port 兜底，再没有就用 5432。
func TestDatabaseDSN_MultiHostDefaultPort(t *testing.T) {
	d := Database{Hosts: []string{"10.0.0.1", "10.0.0.2:6543"}, Port: 5433, Name: "x", User: "u"}
	if got := d.DSN(); !strings.Contains(got, "port=5433,6543") {
		t.Errorf("缺省端口应取 database.port，got: %s", got)
	}

	d.Port = 0
	if got := d.DSN(); !strings.Contains(got, "port=5432,6543") {
		t.Errorf("database.port 也没配时应取 5432，got: %s", got)
	}
}

// hosts 优先于 host：两者都写时以列表为准，避免"改了 hosts 结果没生效"。
func TestDatabaseDSN_HostsTakePrecedence(t *testing.T) {
	d := Database{Host: "old-host", Hosts: []string{"a:1", "b:2"}, Name: "x", User: "u"}
	got := d.DSN()
	if strings.Contains(got, "old-host") {
		t.Errorf("配了 hosts 时不应再使用 host，got: %s", got)
	}
}
