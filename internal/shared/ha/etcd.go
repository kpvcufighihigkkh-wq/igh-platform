// Package ha 的 etcd 部分：只实现本场景需要的三个操作。
//
// 为什么不用官方 client：
//   - 避免引入一个只为四行逻辑服务的大依赖
//   - hactl 必须能在 Windows 工控机上"单文件零依赖"运行
//
// 使用 etcd v3 的 HTTP/JSON 网关（etcd 默认开启，端口与客户端端口一致）。
// 注意：etcd 的 JSON 网关对 int64 字段以**字符串**返回，对 key/value 使用 **base64**。
package ha

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// EtcdClient 是最小化的 etcd v3 客户端。
type EtcdClient struct {
	endpoints []string
	http      *http.Client
}

// NewEtcdClient 用集群配置里全部节点的客户端地址构造客户端。
func NewEtcdClient(nodes []Node) *EtcdClient {
	eps := make([]string, 0, len(nodes))
	for _, n := range nodes {
		eps = append(eps, strings.TrimRight(n.Addr, "/"))
	}
	return &EtcdClient{
		endpoints: eps,
		http:      &http.Client{Timeout: 3 * time.Second},
	}
}

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// post 依次尝试各端点，直到某个成功。
func (c *EtcdClient) post(ctx context.Context, path string, body any, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	var lastErr error
	for _, ep := range c.endpoints {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep+path, bytes.NewReader(raw))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("etcd %s: HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(data)))
			continue
		}
		if out != nil {
			if err := json.Unmarshal(data, out); err != nil {
				lastErr = fmt.Errorf("etcd %s: decode: %w", path, err)
				continue
			}
		}
		return nil
	}
	if lastErr == nil {
		lastErr = errors.New("etcd: no endpoints configured")
	}
	return lastErr
}

// GrantLease 申请一个租约，返回租约 ID。
func (c *EtcdClient) GrantLease(ctx context.Context, ttlSeconds int) (int64, error) {
	var resp struct {
		ID  any    `json:"ID"`
		TTL string `json:"TTL"`
	}
	if err := c.post(ctx, "/v3/lease/grant", map[string]any{"TTL": ttlSeconds}, &resp); err != nil {
		return 0, err
	}
	return toInt64(resp.ID)
}

// RevokeLease 撤销租约（键会随之消失）。
func (c *EtcdClient) RevokeLease(ctx context.Context, leaseID int64) error {
	return c.post(ctx, "/v3/lease/revoke", map[string]any{"ID": leaseID}, nil)
}

// Acquire 原子地抢占 leader 键：仅当键不存在（create_revision == 0）时写入。
//
// 返回 acquired=true 表示抢到；false 表示已被其他节点持有。
func (c *EtcdClient) Acquire(ctx context.Context, key, nodeID string, leaseID int64) (bool, error) {
	req := map[string]any{
		"compare": []any{map[string]any{
			"target": "CREATE",
			"key":    b64(key),
			"result": "EQUAL",
			"create_revision": "0",
		}},
		"success": []any{map[string]any{
			"request_put": map[string]any{
				"key":   b64(key),
				"value": b64(nodeID),
				"lease": leaseID,
			},
		}},
		"failure": []any{},
	}
	var resp struct {
		Succeeded bool `json:"succeeded"`
	}
	if err := c.post(ctx, "/v3/kv/txn", req, &resp); err != nil {
		return false, err
	}
	return resp.Succeeded, nil
}

// Renew 以"新租约替换旧租约"的方式续约。
//
// 仅在键的租约仍是我们自己的旧租约时才成功——这保证了：
// 一旦我们丢失租约（键已消失或已被别人抢走），续约必然失败。
func (c *EtcdClient) Renew(ctx context.Context, key, nodeID string, oldLease, newLease int64) (bool, error) {
	req := map[string]any{
		"compare": []any{map[string]any{
			"target": "LEASE",
			"key":    b64(key),
			// lease 是 int64 字段，只能用十进制字符串，**不能** base64。
			// 之前这里写成 b64(strconv.FormatInt(...))，base64 结果多以 'N'
			// 开头，etcd 直接回 HTTP 400 "invalid character 'N'"，
			// 续约永远失败 → 永远判定为"租约未知" → 每台机器都把自己围栏掉。
			// 已用 curl 对拍：数字字符串成功，base64 失败。
			"lease":  strconv.FormatInt(oldLease, 10),
			"result": "EQUAL",
		}},
		"success": []any{map[string]any{
			"request_put": map[string]any{
				"key":   b64(key),
				"value": b64(nodeID),
				"lease": newLease,
			},
		}},
		"failure": []any{},
	}
	var resp struct {
		Succeeded bool `json:"succeeded"`
	}
	if err := c.post(ctx, "/v3/kv/txn", req, &resp); err != nil {
		return false, err
	}
	return resp.Succeeded, nil
}

// Holder 返回当前 leader 键的持有者 ID 与租约 ID。
// 键不存在时返回 ("", 0, nil)。
func (c *EtcdClient) Holder(ctx context.Context, key string) (string, int64, error) {
	var resp struct {
		Kvs []struct {
			Value string `json:"value"`
			Lease string `json:"lease"`
		} `json:"kvs"`
	}
	req := map[string]any{"key": b64(key)}
	if err := c.post(ctx, "/v3/kv/range", req, &resp); err != nil {
		return "", 0, err
	}
	if len(resp.Kvs) == 0 {
		return "", 0, nil
	}
	val, err := base64.StdEncoding.DecodeString(resp.Kvs[0].Value)
	if err != nil {
		return "", 0, err
	}
	lease, _ := strconv.ParseInt(resp.Kvs[0].Lease, 10, 64)
	return string(val), lease, nil
}

func toInt64(v any) (int64, error) {
	switch x := v.(type) {
	case float64:
		return int64(x), nil
	case string:
		return strconv.ParseInt(x, 10, 64)
	case json.Number:
		return x.Int64()
	default:
		return 0, fmt.Errorf("etcd: unexpected int64 encoding %T", v)
	}
}
