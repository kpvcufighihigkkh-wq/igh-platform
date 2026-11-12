package main

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/ha"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// 这个文件的测试分成两类：
//
//  1. 纯逻辑：不碰数据库，任何环境都能跑（CI）。
//  2. 集成：需要一个可写的 PostgreSQL，通过 IGH_TEST_PG_DSN 提供；
//     没设就 skip。CI 上没有数据库，跳过是正确行为而不是"测试没写"。
//
// 集成测试的意义：hardFence 的正确性无法靠读代码判断——
// "SET default_transaction_read_only" 看起来很像围栏，实际上是空操作。
// 只有真的连上去写一行、发现被拒绝，才算验证过。

func testAgent(cfg *ha.ClusterConfig) *Agent {
	return &Agent{cfg: cfg}
}

// TestWriteStandbyConfig 验证备库配置写对了。
func TestWriteStandbyConfig(t *testing.T) {
	dir := t.TempDir()
	a := testAgent(&ha.ClusterConfig{})

	dsn := "host=192.168.10.2 port=5432 user=replicator password=secret dbname=igh_platform sslmode=disable"
	if err := a.writeStandbyConfig(ha.Node{ID: "server-b"}, dir, dsn); err != nil {
		t.Fatalf("writeStandbyConfig: %v", err)
	}

	// standby.signal 存在是"以恢复模式启动"的唯一标志，没有它 postgres
	// 会以主库身份启动 —— 那正是要防的事故。
	if _, err := os.Stat(filepath.Join(dir, "standby.signal")); err != nil {
		t.Fatalf("缺少 standby.signal: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "postgresql.auto.conf"))
	if err != nil {
		t.Fatalf("读 postgresql.auto.conf: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "primary_conninfo") {
		t.Fatalf("postgresql.auto.conf 缺少 primary_conninfo:\n%s", content)
	}
	if !strings.Contains(content, "host=192.168.10.2") {
		t.Fatalf("primary_conninfo 指向了错误的地址:\n%s", content)
	}
}

// TestWriteStandbyConfig_EscapesQuotes 验证 DSN 里的单引号被转义。
//
// libpq 的 conninfo 用单引号包裹整个值，密码里含单引号时不转义会让整个
// postgresql.auto.conf 解析失败，备库直接起不来。这类口令现场确实在用。
func TestWriteStandbyConfig_EscapesQuotes(t *testing.T) {
	dir := t.TempDir()
	a := testAgent(&ha.ClusterConfig{})

	dsn := "host=10.0.0.2 user=repl password=pa'ss dbname=igh"
	if err := a.writeStandbyConfig(ha.Node{ID: "server-b"}, dir, dsn); err != nil {
		t.Fatalf("writeStandbyConfig: %v", err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "postgresql.auto.conf"))
	if !strings.Contains(string(raw), `pa''ss`) {
		t.Fatalf("单引号未转义:\n%s", raw)
	}
}

// TestPgRewindPath 验证 pg_rewind 路径推导。
func TestPgRewindPath(t *testing.T) {
	t.Run("显式配置优先", func(t *testing.T) {
		a := testAgent(&ha.ClusterConfig{})
		a.cfg.Database.PgRewind = `C:\custom\pg_rewind.exe`
		if got := a.pgRewindPath(); got != `C:\custom\pg_rewind.exe` {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("从 pg_ctl 推导", func(t *testing.T) {
		a := testAgent(&ha.ClusterConfig{})
		a.cfg.Database.PgCtl = filepath.Join("C:", "Program Files", "PostgreSQL", "16", "bin", "pg_ctl.exe")
		got := a.pgRewindPath()
		if filepath.Base(got) != "pg_rewind.exe" {
			t.Fatalf("期望同目录的 pg_rewind.exe，得到 %q", got)
		}
	})

	t.Run("都没有时退回 PATH", func(t *testing.T) {
		a := testAgent(&ha.ClusterConfig{})
		if got := a.pgRewindPath(); got != "pg_rewind" && got != "pg_rewind.exe" {
			t.Fatalf("got %q", got)
		}
	})
}

// TestPeerDBAddr 验证对端地址推导与模拟环境的覆盖。
func TestPeerDBAddr(t *testing.T) {
	a := testAgent(&ha.ClusterConfig{})
	a.cfg.Database.Port = 5432

	host, port := a.peerDBAddr(ha.Node{ID: "server-b", Addr: "http://192.168.10.2:2379"})
	if host != "192.168.10.2" || port != 5432 {
		t.Fatalf("got %s:%d, want 192.168.10.2:5432", host, port)
	}

	// 本机模拟两个节点时，两台都在 127.0.0.1，只能靠端口区分
	t.Setenv("IGH_PEER_DB_HOST", "127.0.0.1")
	t.Setenv("IGH_PEER_DB_PORT", "15432")
	host, port = a.peerDBAddr(ha.Node{ID: "server-b", Addr: "http://127.0.0.1:2381"})
	if host != "127.0.0.1" || port != 15432 {
		t.Fatalf("环境变量覆盖失效: got %s:%d", host, port)
	}
}

// TestPeerNode 验证对端选择会排除见证节点。
//
// 见证节点是复用的一台边端 PC，上面没有 PostgreSQL。
// 误选它会让回退试图连一个不存在的库，然后永久卡在"对端未知"。
func TestPeerNode(t *testing.T) {
	cfg := &ha.ClusterConfig{Nodes: []ha.Node{
		{ID: "server-a", Addr: "http://10.0.0.1:2379", Role: "server"},
		{ID: "server-b", Addr: "http://10.0.0.2:2379", Role: "server"},
		{ID: "witness", Addr: "http://10.0.0.3:2379", Role: "witness"},
	}}

	for _, self := range []string{"server-a", "server-b"} {
		a := &Agent{cfg: cfg, nodeID: self}
		peer, ok := a.peerNode()
		if !ok {
			t.Fatalf("%s: 没找到对端", self)
		}
		if peer.ID == self || peer.Role != "server" {
			t.Fatalf("%s: 选错对端 %+v", self, peer)
		}
	}
}

// ---------------------------------------------------------------------------
// 集成测试
// ---------------------------------------------------------------------------

// openTestDB 连接 IGH_TEST_PG_DSN 指定的数据库；没配就跳过。
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("IGH_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("未设置 IGH_TEST_PG_DSN，跳过集成测试")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("连接测试库失败: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Ping 失败（测试库不可达）: %v", err)
	}
	return db
}

// TestHardFence_BlocksWrites 是本文件最关键的测试。
//
// 它验证的是：围栏之后，**另一条连接**（模拟连接池里的老连接）
// 也无法写入。旧的 fence() 实现通不过这个测试——
// 它只 SET 了自己那条会话，业务连接完全不受影响。
func TestHardFence_BlocksWrites(t *testing.T) {
	dsn := os.Getenv("IGH_TEST_PG_DSN")
	db := openTestDB(t)
	defer db.Close()

	// 测试结束后恢复可写，否则会把这个库永久变成只读，影响后续测试
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx, "ALTER SYSTEM SET default_transaction_read_only = off")
		_, _ = db.ExecContext(ctx, "SELECT pg_reload_conf()")
	}()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS fence_probe (
		id serial PRIMARY KEY, note text)`); err != nil {
		t.Fatalf("建表失败（需要可写的测试库）: %v", err)
	}
	defer db.Exec(`DROP TABLE IF EXISTS fence_probe`)

	a := &Agent{cfg: &ha.ClusterConfig{}, db: db, logger: nopLogger()}
	if err := a.hardFence(context.Background()); err != nil {
		t.Fatalf("hardFence: %v", err)
	}

	// 用一条全新连接验证：新连接会读到 reload 后的只读默认值
	fresh, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer fresh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var ro string
	if err := fresh.QueryRowContext(ctx, "SHOW default_transaction_read_only").Scan(&ro); err != nil {
		t.Fatalf("SHOW: %v", err)
	}
	if ro != "on" {
		t.Fatalf("围栏后新连接仍可写：default_transaction_read_only=%s", ro)
	}

	if _, err := fresh.ExecContext(ctx, "INSERT INTO fence_probe(note) VALUES ($1)", "should-fail"); err == nil {
		t.Fatal("围栏后写入成功了，围栏无效")
	}

	// serve() 应当把只读解除
	if err := a.serve(context.Background()); err != nil {
		t.Fatalf("serve: %v", err)
	}
	fresh2, _ := sql.Open("pgx", dsn)
	defer fresh2.Close()
	if err := fresh2.QueryRowContext(ctx, "SHOW default_transaction_read_only").Scan(&ro); err != nil {
		t.Fatalf("SHOW: %v", err)
	}
	if ro != "off" {
		t.Fatalf("serve 之后仍为只读：%s", ro)
	}
	if _, err := fresh2.ExecContext(ctx, "INSERT INTO fence_probe(note) VALUES ($1)", "ok"); err != nil {
		t.Fatalf("serve 之后仍无法写入: %v", err)
	}
}

// TestReadDBState 验证主备状态识别。
func TestReadDBState(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	a := &Agent{cfg: &ha.ClusterConfig{}, db: db, logger: nopLogger()}
	got := a.readDBState(context.Background())
	if got != ha.DBPrimary && got != ha.DBStandbyCaughtUp && got != ha.DBStandbyLagging {
		t.Fatalf("readDBState 返回未知状态: %s", got)
	}
	t.Logf("IGH_TEST_PG_DSN 指向的库状态: %s", got)
}

// nopLogger 让 Agent 在测试里能打日志而不输出到终端。
func nopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
