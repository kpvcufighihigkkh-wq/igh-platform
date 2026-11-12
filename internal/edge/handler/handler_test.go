package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

func TestMain(m *testing.M) {
	logger.Init("debug")
	os.Exit(m.Run())
}

func newTestTransport(serverURL string) *Transport {
	cfg := &config.EdgeConfig{
		Edge: config.EdgeSection{
			ID: "test-edge",
			Server: config.EdgeServer{
				URL:    serverURL,
				APIKey: "sk-test",
			},
		},
	}
	return NewTransport(cfg)
}

func TestRegister_Success(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/edge/register" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "sk-test" {
			t.Errorf("missing API key header")
		}
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &gotBody)
		w.WriteHeader(200)
		w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()

	tr := newTestTransport(srv.URL)
	if err := tr.Register("winding", "1.0.0"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if gotBody["edge_id"] != "test-edge" {
		t.Errorf("edge_id = %v", gotBody["edge_id"])
	}
	if gotBody["type"] != "winding" {
		t.Errorf("type = %v", gotBody["type"])
	}
}

func TestRegister_ServerDown(t *testing.T) {
	tr := newTestTransport("http://127.0.0.1:1")
	err := tr.Register("winding", "1.0.0")
	if err == nil {
		t.Fatal("expected error when server is unreachable")
	}
}

func TestUploadBatch_Success(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &gotBody)
		w.WriteHeader(200)
		w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()

	tr := newTestTransport(srv.URL)
	if _, err := tr.UploadBatch(nil); err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if gotBody["edge_id"] != "test-edge" {
		t.Errorf("edge_id = %v", gotBody["edge_id"])
	}
}

// TestUploadBatch_503IsError 验证服务端"部分失败"必须被边端当成失败。
//
// 否则边端会认为整批已落库并删除本地队列，未写成功的事件永久丢失。
func TestUploadBatch_503IsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		w.Write([]byte(`{"code":503,"message":"partial failure, retry later"}`))
	}))
	defer srv.Close()

	tr := newTestTransport(srv.URL)
	result, err := tr.UploadBatch(nil)
	if err == nil {
		t.Fatal("expected error for 503, got nil")
	}
	// 错误仍然带上响应体，便于上层读取 rejected_items
	if result == nil || result["code"] != float64(503) {
		t.Errorf("result = %v, want code 503", result)
	}
}

// TestUploadBatch_400IsError 验证客户端错误同样不吞掉。
func TestUploadBatch_400IsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"code":400,"message":"invalid request"}`))
	}))
	defer srv.Close()

	tr := newTestTransport(srv.URL)
	if _, err := tr.UploadBatch(nil); err == nil {
		t.Fatal("expected error for 400, got nil")
	}
}

func TestIsServerReachable_True(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	tr := newTestTransport(srv.URL)
	if !tr.IsServerReachable() {
		t.Fatal("expected reachable")
	}
}

func TestIsServerReachable_False(t *testing.T) {
	tr := newTestTransport("http://127.0.0.1:1")
	if tr.IsServerReachable() {
		t.Fatal("expected unreachable")
	}
}

func TestPull_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("edge_id") != "test-edge" {
			t.Errorf("missing edge_id param")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"data":[{"id":1},{"id":2}]}`))
	}))
	defer srv.Close()

	tr := newTestTransport(srv.URL)
	data, err := tr.Pull("/api/v1/edge/pull/lots", nil)
	if err != nil {
		t.Fatalf("pull failed: %v", err)
	}
	if len(data) != 2 {
		t.Fatalf("expected 2 items, got %d", len(data))
	}
}

func TestPull_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	tr := newTestTransport(srv.URL)
	_, err := tr.Pull("/api/v1/edge/pull/lots", nil)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestEdgeID(t *testing.T) {
	tr := newTestTransport("http://localhost")
	if tr.EdgeID() != "test-edge" {
		t.Fatalf("EdgeID = %q", tr.EdgeID())
	}
}

// TestHeartbeat_UnwrapsDataEnvelope 验证心跳能正确解出服务端下发的配置。
//
// 服务端统一响应格式是 {code, data, message}，config_version 与 config
// 都嵌在 data 下面。早期实现把整个外层当成 HeartbeatResponse 解，
// 于是 ConfigVersion 恒为空、Config 恒为 nil，processConfig 两次提前返回——
// 现场表现为"中心一直重发配置，边端永远收不到"，且日志里一条记录都没有。
func TestHeartbeat_UnwrapsDataEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/edge/heartbeat" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"data":{
			"config_version":"7",
			"config":{"plc":[{"code":"winder-L01","host":"192.168.5.101"}]},
			"heartbeat_interval":10
		}}`))
	}))
	defer srv.Close()

	tr := newTestTransport(srv.URL)
	resp, err := tr.Heartbeat(types.HeartbeatRequest{ConfigVersion: ""})
	if err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}
	if resp.ConfigVersion != "7" {
		t.Errorf("ConfigVersion = %q, want \"7\"（没解 data 层）", resp.ConfigVersion)
	}
	if resp.Config == nil {
		t.Fatal("Config = nil（没解 data 层）")
	}
	plc, _ := resp.Config["plc"].([]any)
	if len(plc) != 1 {
		t.Errorf("config.plc 长度 = %d, want 1", len(plc))
	}
}

// TestHeartbeat_EmptyConfig 验证服务端没有配置时不应报错，也不应塞空对象。
func TestHeartbeat_EmptyConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":0,"data":{"commands":null}}`))
	}))
	defer srv.Close()

	tr := newTestTransport(srv.URL)
	resp, err := tr.Heartbeat(types.HeartbeatRequest{})
	if err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}
	if resp.ConfigVersion != "" {
		t.Errorf("ConfigVersion = %q, want empty", resp.ConfigVersion)
	}
	if resp.Config != nil {
		t.Errorf("Config = %v, want nil", resp.Config)
	}
}

// ---------------------------------------------------------------------------
// 多中心地址
// ---------------------------------------------------------------------------

func newMultiTransport(urls ...string) *Transport {
	cfg := &config.EdgeConfig{
		Edge: config.EdgeSection{
			ID: "test-edge",
			Server: config.EdgeServer{
				URLs:   urls,
				APIKey: "sk-test",
			},
		},
	}
	return NewTransport(cfg)
}

// 第一台中心返回 5xx 时，必须自动切到第二台并成功。
//
// 这是双机热备的核心：中心 A 挂了，边端不能就此停摆。
func TestPost_FailoverOn5xx(t *testing.T) {
	var hitA, hitB int

	a := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitA++
		w.WriteHeader(503)
		w.Write([]byte(`{"code":503,"message":"db down"}`))
	}))
	defer a.Close()

	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitB++
		w.Write([]byte(`{"code":0,"data":{"ok":true}}`))
	}))
	defer b.Close()

	tr := newMultiTransport(a.URL, b.URL)
	if err := tr.Register("winding", "v1"); err != nil {
		t.Fatalf("应自动切到第二台并成功，实际报错: %v", err)
	}
	if hitA == 0 {
		t.Error("第一台应该被尝试过")
	}
	if hitB != 1 {
		t.Errorf("第二台命中次数 = %d, want 1", hitB)
	}
	if got := tr.ServerURL(); got != b.URL {
		t.Errorf("粘滞地址 = %q, want %q（成功一次后应记住）", got, b.URL)
	}
}

// 连接失败（端口没人监听）同样要切换。
func TestPost_FailoverOnConnError(t *testing.T) {
	// 先监听再关掉，拿到一个确定没人监听的地址
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := dead.URL
	dead.Close()

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":0}`))
	}))
	defer ok.Close()

	tr := newMultiTransport(deadURL, ok.URL)
	if err := tr.Register("winding", "v1"); err != nil {
		t.Fatalf("连接失败后应切到第二台，实际: %v", err)
	}
}

// 4xx 是请求本身的问题，换中心也没用，必须原样返回，不能来回切。
func TestPost_NoFailoverOn4xx(t *testing.T) {
	var hitB int

	a := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"code":401,"message":"invalid api key"}`))
	}))
	defer a.Close()

	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitB++
		w.Write([]byte(`{"code":0}`))
	}))
	defer b.Close()

	tr := newMultiTransport(a.URL, b.URL)
	err := tr.Register("winding", "v1")
	if err == nil {
		t.Fatal("401 应当返回错误")
	}
	if hitB != 0 {
		t.Errorf("4xx 不应切换中心，第二台却被调用了 %d 次", hitB)
	}
	if got := tr.ServerURL(); got != a.URL {
		t.Errorf("粘滞地址不应改变，got %q", got)
	}
}

// 循环切换：两台都挂时要把所有地址都试一遍，最后报错而不是静默成功。
func TestPost_AllEndpointsDown(t *testing.T) {
	dead1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	u1 := dead1.URL
	dead1.Close()
	dead2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	u2 := dead2.URL
	dead2.Close()

	tr := newMultiTransport(u1, u2)
	if err := tr.Register("winding", "v1"); err == nil {
		t.Fatal("两台都不可达时必须报错，否则调用方会以为上报成功")
	}
}

// 成功后记住地址：下一次请求不应再打已经挂掉的那台。
func TestPost_StickyAfterSuccess(t *testing.T) {
	var hitA, hitB int

	a := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitA++
		w.WriteHeader(503)
	}))
	defer a.Close()

	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitB++
		w.Write([]byte(`{"code":0}`))
	}))
	defer b.Close()

	tr := newMultiTransport(a.URL, b.URL)
	tr.Register("winding", "v1") // 第一次：A 失败，切到 B
	tr.Register("winding", "v1") // 第二次应直接用 B

	if hitB != 2 {
		t.Errorf("第二次请求应直接用 B，hitB = %d", hitB)
	}
	if hitA != 1 {
		t.Errorf("A 只应在第一次被尝试，hitA = %d", hitA)
	}
}

func TestIsServerReachable_Failover(t *testing.T) {
	a := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer a.Close()
	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer b.Close()

	tr := newMultiTransport(a.URL, b.URL)
	if !tr.IsServerReachable() {
		t.Fatal("第二台可用时应判定为可达")
	}
}

func TestPull_Failover(t *testing.T) {
	a := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer a.Close()
	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":0,"data":[{"id":1}]}`))
	}))
	defer b.Close()

	tr := newMultiTransport(a.URL, b.URL)
	rows, err := tr.Pull("/api/v1/edge/sync/lots", nil)
	if err != nil {
		t.Fatalf("应切到第二台并成功: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("返回行数 = %d, want 1", len(rows))
	}
}
