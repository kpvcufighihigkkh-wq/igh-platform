package plc

import (
	"bytes"
	"sync"
	"sync/atomic"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

type DeviceState struct {
	Code       string `json:"code"`
	Model      string `json:"model"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Protocol   string `json:"protocol"`
	Connected  bool   `json:"connected"`
	LastReadMs int64  `json:"last_read_ms"`
	Errors     int    `json:"errors"`
	cfg        config.PLCDevice
	proto      Protocol
	lastData   map[string][]byte
	// lastValues 记录点表模式下每个字段的上一次解码值，用于变更检测。
	lastValues map[string]any
	// stop 只结束这一个设备的轮询循环，供 Reload 单独替换设备时使用。
	stop chan struct{}
	mu   sync.Mutex
}

type Manager struct {
	// mu 保护 devices 与 started：配置应用时会整体替换设备表，
	// 而 Status/Read/Write 可能同时被 HTTP 请求调用。
	mu           sync.RWMutex
	devices      map[string]*DeviceState
	pollInterval time.Duration
	stopCh       chan struct{}
	started      bool
	// paused 为真时只保持连接、不轮询、不上报事件。
	// 主备模式下备机必须是 paused：两台同时轮询同一个 PLC 时，
	// 每一次落筒都会产生两条事件，中心侧无法去重（edge_id 不同）。
	paused  atomic.Bool
	onEvent func(deviceCode string, data map[string]any)
}

func NewManager(cfg *config.EdgeConfig) *Manager {
	m := &Manager{
		devices:      make(map[string]*DeviceState),
		pollInterval: time.Duration(cfg.Edge.PLC.PollIntervalMs) * time.Millisecond,
		stopCh:       make(chan struct{}),
	}
	if m.pollInterval == 0 {
		m.pollInterval = time.Second
	}
	for _, dev := range cfg.Edge.PLC.Devices {
		m.devices[dev.Code] = newDeviceState(dev)
	}
	return m
}

// newDeviceState 由配置构造一个设备运行时状态。
func newDeviceState(dev config.PLCDevice) *DeviceState {
	protocol := dev.Protocol
	if protocol == "" {
		protocol = "s7"
	}
	return &DeviceState{
		Code:       dev.Code,
		Model:      dev.Model,
		Host:       dev.Host,
		Port:       dev.Port,
		Protocol:   protocol,
		cfg:        dev,
		lastData:   make(map[string][]byte),
		lastValues: make(map[string]any),
		stop:       make(chan struct{}),
	}
}

// Reload 用新点表整体替换设备表，无需重启进程。
//
// 语义：
//   - 旧设备全部停止轮询并断开
//   - 新设备按新配置重建（lastValues 清空，因此会重新产生一次全量 plc_points 事件，
//     这正是需要的：新点表的首轮读数必须上报，否则中心看不到新字段的初始值）
//   - 若管理器已在运行，新设备立即开始轮询
func (m *Manager) Reload(devices []config.PLCDevice) {
	next := make(map[string]*DeviceState, len(devices))
	for _, dev := range devices {
		next[dev.Code] = newDeviceState(dev)
	}

	m.mu.Lock()
	old := m.devices
	m.devices = next
	running := m.started
	m.mu.Unlock()

	for _, dev := range old {
		close(dev.stop)
		dev.mu.Lock()
		if dev.proto != nil {
			dev.proto.Disconnect()
		}
		dev.Connected = false
		dev.mu.Unlock()
	}

	if running {
		for _, dev := range next {
			go m.deviceLoop(dev)
		}
	}
	logger.L.Infow("plc devices reloaded", "old", len(old), "new", len(next))
}

// deviceSnapshot 返回当前设备表的只读视图。
func (m *Manager) deviceSnapshot() map[string]*DeviceState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.devices
}

// Device 返回指定设备的配置（点表），供需要按字段名定位地址的调用方使用。
//
// 返回的是配置副本，调用方无法通过它影响运行时状态。
func (m *Manager) Device(code string) (config.PLCDevice, bool) {
	dev, ok := m.deviceSnapshot()[code]
	if !ok {
		return config.PLCDevice{}, false
	}
	dev.mu.Lock()
	defer dev.mu.Unlock()
	return dev.cfg, true
}

func (m *Manager) SetEventHandler(fn func(string, map[string]any)) {
	m.onEvent = fn
}

// Pause 停止轮询与事件上报，但保留 PLC 连接。
//
// 主备降级时调用。保留连接是为了让升为主机后不必重新握手，
// 切换耗时里省掉一次 TCP 建立；停轮询则是必须的：
// 两台同时读同一个 DB 块时，同一次落筒会各产生一条事件，
// 中心按 (edge_id, event_id) 去重，跨机器去不掉。
func (m *Manager) Pause() {
	if m.paused.Swap(true) {
		return
	}
	logger.L.Infow("plc polling paused (standby role)")
}

// Resume 恢复轮询与事件上报。
func (m *Manager) Resume() {
	if !m.paused.Swap(false) {
		return
	}
	logger.L.Infow("plc polling resumed (primary role)")
}

// Paused 返回当前是否处于暂停轮询状态。
func (m *Manager) Paused() bool { return m.paused.Load() }

func (m *Manager) Start() {
	devices := m.deviceSnapshot()
	if len(devices) == 0 {
		logger.L.Info("plc manager: no devices configured")
		return
	}

	m.mu.Lock()
	m.started = true
	m.mu.Unlock()

	logger.L.Infow("plc manager starting", "devices", len(devices), "poll_ms", m.pollInterval.Milliseconds())

	for _, dev := range devices {
		go m.deviceLoop(dev)
	}

	<-m.stopCh
}

func (m *Manager) Stop() {
	close(m.stopCh)

	m.mu.Lock()
	m.started = false
	m.mu.Unlock()

	for _, dev := range m.deviceSnapshot() {
		close(dev.stop)
		dev.mu.Lock()
		if dev.proto != nil {
			dev.proto.Disconnect()
		}
		dev.Connected = false
		dev.mu.Unlock()
		logger.L.Infow("plc disconnected", "code", dev.Code)
	}
}

func (m *Manager) Status() map[string]any {
	result := make(map[string]any)
	for code, dev := range m.deviceSnapshot() {
		dev.mu.Lock()
		result[code] = map[string]any{
			"connected":    dev.Connected,
			"last_read_ms": dev.LastReadMs,
			"host":         dev.Host,
			"port":         dev.Port,
			"protocol":     dev.Protocol,
			"model":        dev.Model,
			"errors":       dev.Errors,
		}
		dev.mu.Unlock()
	}
	return result
}

func (m *Manager) Write(deviceCode string, dbNumber int, start int, data []byte) error {
	dev, ok := m.deviceSnapshot()[deviceCode]
	if !ok {
		return errDeviceNotFound(deviceCode)
	}
	dev.mu.Lock()
	defer dev.mu.Unlock()

	if dev.proto == nil || !dev.Connected {
		return errDeviceNotConnected(deviceCode)
	}
	return dev.proto.WriteArea(dbNumber, start, data)
}

func (m *Manager) Read(deviceCode string, dbNumber int, start int, size int) ([]byte, error) {
	dev, ok := m.deviceSnapshot()[deviceCode]
	if !ok {
		return nil, errDeviceNotFound(deviceCode)
	}
	dev.mu.Lock()
	defer dev.mu.Unlock()

	if dev.proto == nil || !dev.Connected {
		return nil, errDeviceNotConnected(deviceCode)
	}
	return dev.proto.ReadArea(dbNumber, start, size)
}

// Values 返回设备最近一次轮询解出的全部点值（键与 key 格式一致，如
// "winder_data/W3/left/DatiBobine_DaWinder.Doff_No"）。
//
// 用途：工位页面需要读 PLC 的实时值——称重值、料车 RFID、包装机已装锭数
// 都是"PLC 读到什么，页面就显示什么"，不能靠页面自己猜或让人手输。
//
// 返回拷贝：lastValues 由轮询协程持续写入，直接返回内部 map 会让调用方
// 与轮询并发读写同一个 map（Go 运行时会直接崩，不是数据竞争那么简单）。
//
// 设备未连接或还没读到数据时返回空 map 而不是错误：页面要能把
// "暂时没有值"和"设备不存在"区分开。
func (m *Manager) Values(deviceCode string) map[string]any {
	dev, ok := m.deviceSnapshot()[deviceCode]
	if !ok {
		return map[string]any{}
	}
	dev.mu.Lock()
	defer dev.mu.Unlock()

	out := make(map[string]any, len(dev.lastValues))
	for k, v := range dev.lastValues {
		out[k] = v
	}
	return out
}

// DeviceCodes 返回全部设备编码，供页面在未指定设备时遍历。
func (m *Manager) DeviceCodes() []string {
	devices := m.deviceSnapshot()
	out := make([]string, 0, len(devices))
	for code := range devices {
		out = append(out, code)
	}
	return out
}

func (m *Manager) deviceLoop(dev *DeviceState) {
	reconnectDelay := 5 * time.Second
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	m.connectDevice(dev)

	for {
		select {
		case <-m.stopCh:
			return
		case <-dev.stop:
			// 该设备被 Reload 替换掉了
			return
		case <-ticker.C:
			// 备机不读点表：读了就会按变化量上报事件，与主机重复。
			// 连接仍然维持，升主后直接恢复读取。
			if m.paused.Load() {
				continue
			}

			dev.mu.Lock()
			connected := dev.Connected
			dev.mu.Unlock()

			if !connected {
				time.Sleep(reconnectDelay)
				m.connectDevice(dev)
				continue
			}
			m.pollDevice(dev)
		}
	}
}

func (m *Manager) connectDevice(dev *DeviceState) {
	proto, err := NewProtocolWithDevice(dev.Protocol, dev.Host, dev.Port, dev.cfg.Rack, dev.cfg.Slot, &dev.cfg)
	if err != nil {
		logger.L.Warnw("plc protocol create failed", "code", dev.Code, "error", err)
		return
	}

	if err := proto.Connect(); err != nil {
		logger.L.Warnw("plc connect failed", "code", dev.Code, "host", dev.Host, "error", err)
		dev.mu.Lock()
		dev.Errors++
		dev.mu.Unlock()
		return
	}

	dev.mu.Lock()
	dev.proto = proto
	dev.Connected = true
	dev.Errors = 0
	dev.mu.Unlock()

	logger.L.Infow("plc connected", "code", dev.Code, "host", dev.Host, "protocol", dev.Protocol)
}

func (m *Manager) pollDevice(dev *DeviceState) {
	if len(dev.cfg.ReadBlocks) == 0 && len(dev.cfg.PollGroups) == 0 {
		return
	}

	// ---- 点表模式：一次块读，按字段语义解码 ----
	for _, group := range dev.cfg.PollGroups {
		if err := m.pollGroup(dev, group); err != nil {
			return
		}
	}

	// ---- 原始块模式（仿真与简单场景）----
	for _, block := range dev.cfg.ReadBlocks {
		start := time.Now()

		dev.mu.Lock()
		proto := dev.proto
		dev.mu.Unlock()

		if proto == nil {
			return
		}

		data, err := proto.ReadArea(block.DB, block.Start, block.Size)
		elapsed := time.Since(start).Milliseconds()

		dev.mu.Lock()
		dev.LastReadMs = elapsed
		if err != nil {
			dev.Connected = false
			dev.Errors++
			dev.mu.Unlock()
			logger.L.Warnw("plc read failed", "code", dev.Code, "tag", block.Tag, "error", err)
			return
		}
		dev.mu.Unlock()

		tag := block.Tag
		if tag == "" {
			tag = "data"
		}

		prev, existed := dev.lastData[tag]
		if !existed || !bytes.Equal(prev, data) {
			dev.lastData[tag] = make([]byte, len(data))
			copy(dev.lastData[tag], data)

			if m.onEvent != nil {
				m.onEvent(dev.Code, map[string]any{
					"event_type": "plc_data",
					"tag":        tag,
					"db":         block.DB,
					"start":      block.Start,
					"size":       block.Size,
					"data":       data,
					"read_ms":    elapsed,
				})
			}
		}
	}
}

// pollGroup 读取一个点表分组，解码字段并上报发生变化的部分。
//
// 关键设计：
//   - 整个分组只发起 1 次 PLC 读（避免老系统"每字段一次读"的读放大）
//   - 以 PointAddress.Key() 作为变更检测的键
//   - 上报的是**结构化字段**，不是原始字节
func (m *Manager) pollGroup(dev *DeviceState, group config.PLCPollGroup) error {
	length := group.BlockLength()
	if length <= 0 {
		logger.L.Warnw("plc poll group has zero length", "code", dev.Code, "group", group.Name)
		return nil
	}

	start := time.Now()

	dev.mu.Lock()
	proto := dev.proto
	dev.mu.Unlock()

	if proto == nil {
		return errDeviceNotConnected(dev.Code)
	}

	data, err := proto.ReadArea(group.DB, group.Start, length)
	elapsed := time.Since(start).Milliseconds()

	dev.mu.Lock()
	dev.LastReadMs = elapsed
	if err != nil {
		dev.Connected = false
		dev.Errors++
		dev.mu.Unlock()
		logger.L.Warnw("plc group read failed",
			"code", dev.Code, "group", group.Name, "db", group.DB, "error", err)
		return err
	}
	dev.mu.Unlock()

	changed := make(map[string]any)
	values := make(map[string]any)
	for _, p := range config.ExpandPoints(&dev.cfg) {
		if p.Group != group.Name {
			continue
		}
		end := p.Offset + pointWidth(p)
		if p.Offset < 0 || end > len(data) {
			logger.L.Warnw("plc point out of block",
				"code", dev.Code, "group", group.Name, "field", p.Field,
				"offset", p.Offset, "end", end, "block_len", len(data))
			continue
		}
		value, err := DecodeValue(data[p.Offset:end], p.Type, p.Length, p.Bit)
		if err != nil {
			logger.L.Warnw("plc decode failed",
				"code", dev.Code, "group", group.Name, "field", p.Field, "error", err)
			continue
		}

		key := p.Key()
		values[key] = value
		prev, existed := dev.lastValues[key]
		if !existed || !valuesEqual(prev, value) {
			dev.lastValues[key] = value
			changed[key] = value
		}
	}

	if len(changed) > 0 && m.onEvent != nil {
		m.onEvent(dev.Code, map[string]any{
			"event_type": "plc_points",
			"group":      group.Name,
			"db":         group.DB,
			"start":      group.Start,
			"length":     length,
			"read_ms":    elapsed,
			// changed 用于触发（只含本次变化）
			"changed": changed,
			// values 是本次完整快照，供业务层拼装事件时取"不常变化"的字段
			// （例如批号 CodeNumber 可能整班不变，永远不会出现在 changed 里）
			"values": values,
		})
	}
	return nil
}

// pointWidth 返回字段占用的字节数（与 config 包保持一致）。
func pointWidth(p config.PointAddress) int {
	switch p.Type {
	case "bool", "byte", "uint8", "int8":
		return 1
	case "int16", "uint16":
		return 2
	case "int32", "uint32", "float32":
		return 4
	case "float64":
		return 8
	case "string":
		return p.Length
	default:
		return 0
	}
}

func valuesEqual(a, b any) bool {
	if fa, ok := toFloat(a); ok {
		if fb, ok := toFloat(b); ok {
			return fa == fb
		}
	}
	return a == b
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int8:
		return float64(n), true
	case uint8:
		return float64(n), true
	case int16:
		return float64(n), true
	case uint16:
		return float64(n), true
	case int32:
		return float64(n), true
	case uint32:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

type plcError struct{ msg string }

func (e *plcError) Error() string             { return e.msg }
func errDeviceNotFound(code string) error     { return &plcError{"plc device not found: " + code} }
func errDeviceNotConnected(code string) error { return &plcError{"plc device not connected: " + code} }
