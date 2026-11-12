package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

// Transport 是与中心通信的客户端。
//
// 支持多个中心地址：按顺序尝试，成功一次就记住这个地址（粘滞），
// 下次优先用它；连续失败才轮到下一个。
// 粘滞的意义在于：VIP 漂移后第一项可能已经指向另外一台机器，
// 每次都从头试会让每次请求都多一次连接超时。
type Transport struct {
	edgeID string
	apiKey string
	client *http.Client

	mu        sync.RWMutex
	endpoints []string
	// active 是当前正在用的地址下标。0 表示优先用列表第一项。
	active int
}

func NewTransport(cfg *config.EdgeConfig) *Transport {
	eps := cfg.Edge.Server.Endpoints()
	if len(eps) == 0 {
		// 配置里一个地址都没写。不在这里 panic：边端仍要能启动
		// （本地页面、PLC 采集都还能用），只是同步会一直失败并打日志。
		logger.L.Warn("edge: 未配置中心地址（server.url 或 server.urls），将无法与中心同步")
	}
	return &Transport{
		edgeID:    cfg.Edge.ID,
		apiKey:    cfg.Edge.Server.APIKey,
		client:    &http.Client{Timeout: 10 * time.Second},
		endpoints: eps,
	}
}

func (t *Transport) EdgeID() string { return t.edgeID }

// ServerURL 返回当前生效的中心地址，用于日志与状态展示。
func (t *Transport) ServerURL() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if len(t.endpoints) == 0 {
		return ""
	}
	return t.endpoints[t.active]
}

// rotate 切换到下一个中心地址。
//
// 只在连接层面失败（网络错误/超时/5xx）时调用。4xx 属于请求本身的问题，
// 换一台中心也一样会被拒，来回切换只会掩盖真正的错误。
func (t *Transport) rotate(failed string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.endpoints) <= 1 {
		return
	}
	if t.endpoints[t.active] != failed {
		// 别处已经切换过了，避免重复推进
		return
	}
	next := (t.active + 1) % len(t.endpoints)
	logger.L.Warnw("edge: 切换中心地址",
		"from", t.endpoints[t.active], "to", t.endpoints[next])
	t.active = next
}

// candidates 返回按当前粘滞顺序排好的地址列表。
func (t *Transport) candidates() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if len(t.endpoints) == 0 {
		return nil
	}
	out := make([]string, 0, len(t.endpoints))
	for i := 0; i < len(t.endpoints); i++ {
		out = append(out, t.endpoints[(t.active+i)%len(t.endpoints)])
	}
	return out
}

func (t *Transport) Register(stationType, version string) error {
	body := map[string]any{
		"edge_id": t.edgeID,
		"type":    stationType,
		"version": version,
	}
	_, err := t.post("/api/v1/edge/register", body)
	if err != nil {
		logger.L.Warnw("register failed", "error", err)
	}
	return err
}

func (t *Transport) Heartbeat(req types.HeartbeatRequest) (*types.HeartbeatResponse, error) {
	req.EdgeID = t.edgeID
	req.Timestamp = time.Now()
	result, err := t.post("/api/v1/edge/heartbeat", req)
	if err != nil {
		return nil, err
	}

	// post 返回的是服务端统一响应的外层 {code, data, message}，
	// 心跳内容在 data 下面。直接解外层得到的是全零值：
	// ConfigVersion 恒为空、Config 恒为 nil，配置永远下发不到边端，
	// 而且日志里一条记录都没有（processConfig 两次提前返回）。
	var resp types.HeartbeatResponse
	if data, ok := result["data"]; ok && data != nil {
		raw, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("marshal heartbeat data: %w", err)
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, fmt.Errorf("decode heartbeat response: %w", err)
		}
	}
	return &resp, nil
}

func (t *Transport) UploadBatch(events []types.EdgeEvent) (map[string]any, error) {
	body := map[string]any{
		"edge_id": t.edgeID,
		"events":  events,
	}
	return t.post("/api/v1/edge/upload/batch", body)
}

func (t *Transport) Pull(endpoint string, since *time.Time) ([]json.RawMessage, error) {
	query := "?edge_id=" + t.edgeID
	if since != nil {
		query += "&since=" + since.Format(time.RFC3339)
	}

	var lastErr error
	for _, base := range t.candidates() {
		req, _ := http.NewRequest("GET", base+endpoint+query, nil)
		req.Header.Set("X-API-Key", t.apiKey)

		resp, err := t.client.Do(req)
		if err != nil {
			lastErr = err
			t.rotate(base)
			continue
		}

		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("pull %s: status %d", endpoint, resp.StatusCode)
			t.rotate(base)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("pull %s: status %d", endpoint, resp.StatusCode)
		}

		var result struct {
			Code int               `json:"code"`
			Data []json.RawMessage `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("decode pull response: %w", err)
		}
		return result.Data, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("pull %s: 没有可用的中心地址", endpoint)
	}
	return nil, lastErr
}

func (t *Transport) AckCommand(cmdID, status, message string) error {
	body := map[string]any{
		"edge_id":    t.edgeID,
		"command_id": cmdID,
		"status":     status,
		"message":    message,
	}
	_, err := t.post("/api/v1/edge/command/ack", body)
	return err
}

func (t *Transport) IsServerReachable() bool {
	for _, base := range t.candidates() {
		req, _ := http.NewRequest("GET", base+"/health", nil)
		resp, err := t.client.Do(req)
		if err != nil {
			t.rotate(base)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return true
		}
	}
	return false
}

// post 按当前优先级尝试各中心地址。
//
// 关键点：只有"连接失败"或"5xx"才换地址。4xx 是请求本身有问题
// （API Key 错、参数不合法），换一台中心同样会被拒，
// 来回切换只会让真正的错误被大量重试日志淹没。
func (t *Transport) post(path string, body any) (map[string]any, error) {
	data, _ := json.Marshal(body)

	var lastErr error
	// 保留最后一次 5xx 的响应体：批量上传部分失败时，上层要靠
	// result["code"] / rejected_items 决定哪些事件需要重新入队。
	// 只返回 error 而不带 body，会让调用方无从判断，只能整批重传。
	var lastResult map[string]any

	for _, base := range t.candidates() {
		req, _ := http.NewRequest("POST", base+path, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", t.apiKey)

		resp, err := t.client.Do(req)
		if err != nil {
			lastErr = err
			t.rotate(base)
			continue
		}

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		// 5xx 视为"这台中心有问题"，换一台重试
		if resp.StatusCode >= 500 {
			msg := fmt.Sprintf("http %d", resp.StatusCode)
			if m, ok := result["message"].(string); ok && m != "" {
				msg += ": " + m
			}
			lastErr = fmt.Errorf("POST %s: %s", path, msg)
			lastResult = result
			t.rotate(base)
			continue
		}

		// 非 2xx 必须当失败处理：批量上传返回 4xx 表示请求本身被拒，
		// 换中心也没用，直接返回让调用方决定（重试或丢弃）。
		// 这里若静默返回成功，调用方会删掉本地队列，数据就永久丢了。
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			msg := fmt.Sprintf("http %d", resp.StatusCode)
			if m, ok := result["message"].(string); ok && m != "" {
				msg += ": " + m
			}
			return result, fmt.Errorf("POST %s: %s", path, msg)
		}
		return result, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("POST %s: 没有可用的中心地址", path)
	}
	return lastResult, lastErr
}
