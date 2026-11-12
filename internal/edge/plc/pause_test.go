package plc

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/config"
)

// TestPause_StopsPollingAndEvents 验证备机角色下不产生任何 PLC 事件。
//
// 为什么必须有这个测试：主备两台机器若同时轮询同一个 PLC 块，
// 同一次落筒会各产生一条事件。中心按 (edge_id, event_id) 去重，
// 而两台机器的 edge_id 不同——去不掉，落筒会被记两次。
// 所以"备机不读"必须是硬约束，不能只靠部署时记得别开。
func TestPause_StopsPollingAndEvents(t *testing.T) {
	var events atomic.Int64

	cfg := testManagerCfg(config.PLCDevice{
		Code:       "plc-sim",
		Protocol:   "simulation",
		ReadBlocks: []config.PLCReadBlock{{DB: 1, Start: 0, Size: 16, Tag: "status"}},
	})
	cfg.Edge.PLC.PollIntervalMs = 20

	m := NewManager(cfg)
	m.SetEventHandler(func(string, map[string]any) { events.Add(1) })

	m.Pause()
	if !m.Paused() {
		t.Fatal("Pause() 之后 Paused() 应为 true")
	}

	go m.Start()
	defer m.Stop()

	// 暂停期间：连接可以建，但一次事件都不该有
	time.Sleep(400 * time.Millisecond)
	if n := events.Load(); n != 0 {
		t.Fatalf("暂停期间产生了 %d 条事件，备机会与主机重复上报", n)
	}

	// 恢复后：仿真数据每次都变，必须重新开始上报
	m.Resume()
	if m.Paused() {
		t.Fatal("Resume() 之后 Paused() 应为 false")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if events.Load() > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("恢复轮询后 2 秒内没有任何事件，升主后产线会静默")
}
