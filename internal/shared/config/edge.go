package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type EdgeConfig struct {
	Edge EdgeSection `mapstructure:"edge"`
}

type EdgeSection struct {
	ID       string        `mapstructure:"id"`
	Type     string        `mapstructure:"type"`
	Version  string        `mapstructure:"-"`
	DataDir  string        `mapstructure:"data_dir"`
	// ConfigDir 是配置目录的绝对路径，由 LoadEdge 填入，不从 YAML 读。
	// 用于定位 labels/templates 下的标签模板：模板跟配置一起分发，
	// 不能靠"当前工作目录"去猜，服务方式启动时工作目录并不可靠。
	ConfigDir string        `mapstructure:"-"`
	HTTP     EdgeHTTP      `mapstructure:"http"`
	Server   EdgeServer    `mapstructure:"server"`
	PLC      EdgePLC       `mapstructure:"plc"`
	Printers []EdgePrinter `mapstructure:"printers"`
	// Palletizers 把打包机 PLC 与打包机号/厂编码绑定，用于生成栈板编码与打标。
	// 只有包装工位需要；其它工位留空即可。
	Palletizers []EdgePalletizer `mapstructure:"palletizers"`
	// Warehouses 把仓库 PLC 与仓库编码绑定，用于把 PLC 的库位信号翻译成出入库事件。
	// 只有仓储工位需要；其它工位留空即可。
	Warehouses []EdgeWarehouse `mapstructure:"warehouses"`
	HA         EdgeHA          `mapstructure:"ha"`
	Line       EdgeLine        `mapstructure:"line"`
	// BobbinsPerDoff 是一次落筒产出的锭数。
	// 来源：线路主数据 configs/production_lines.yaml 中对应线路的 bobbins_per_doff。
	// 没有代码默认值：不同机型锭数不同，缺配置时落筒事件会被拒绝上报。
	BobbinsPerDoff int `mapstructure:"bobbins_per_doff"`
	// BobbinLabelPrinter 是丝锭标签打印机的编码（对应 printers 段的 code）。
	//
	// 留空表示本工位不打印丝锭标签。这是默认值：
	// 现场有没有装标签机取决于实际配置，不能假定一定装了。
	//
	// 打印失败**不影响**落筒事件上报——丝锭已经真实产出了，
	// 数据比纸重要。失败只打 error 日志，现场可从 /edge/print 补打。
	BobbinLabelPrinter string `mapstructure:"bobbin_label_printer"`
}

type EdgeHTTP struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

func (h EdgeHTTP) Addr() string {
	return fmt.Sprintf("%s:%d", h.Host, h.Port)
}

type EdgeServer struct {
	// URL 是单个中心地址，保持向后兼容；配了 URLs 时以 URLs 为准。
	URL string `mapstructure:"url"`
	// URLs 是中心地址列表，边端按顺序尝试，连不上就换下一个。
	//
	// 为什么需要：双机热备下如果只配一台中心服务器的 IP，那台挂了边端
	// 就永久失联——哪怕另一台活得好好的、VIP 也漂过去了。
	// 配成列表后，边端在连接失败时自动切换，不需要人工改配置重启。
	//
	// 建议第一项写 VIP（跟着 keepalived 走），后面写两台实机地址兜底。
	URLs            []string `mapstructure:"urls"`
	APIKey          string   `mapstructure:"api_key"`
	HeartbeatSec    int      `mapstructure:"heartbeat_sec"`
	SyncIntervalSec int      `mapstructure:"sync_interval_sec"`
}

// Endpoints 返回去重后的中心地址列表（去掉空串与末尾斜杠）。
//
// 单地址部署返回只有一项的列表，调用方无需区分两种模式。
func (s EdgeServer) Endpoints() []string {
	raw := s.URLs
	if len(raw) == 0 {
		raw = []string{s.URL}
	}
	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))
	for _, u := range raw {
		u = strings.TrimRight(strings.TrimSpace(u), "/")
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, u)
	}
	return out
}

type EdgePLC struct {
	PollIntervalMs int         `mapstructure:"poll_interval_ms"`
	Devices        []PLCDevice `mapstructure:"devices"`
	// InstanceFiles 是要合并进来的真实点表文件（相对 configDir）。
	// 例如 ["devices/instances/spinning_sides.yaml"]。
	// 合并规则：按 code 去重；文件中的设备与 edge.devices 同一 code 时，
	// 以文件（真实点表）为准，避免手写配置覆盖生成的点表。
	InstanceFiles []string `mapstructure:"instance_files"`
	// Strict 为 true 时，点表里的 DB 号必须与 edge.devices 中的同 code 设备一致，
	// 否则启动失败。生产环境建议开启。
	Strict bool `mapstructure:"strict"`
	// ProtocolOverride 非空时，覆盖所有设备的协议。
	// 典型用途：开发机用 "simulation" 跑真实点表，验证完整读路径而不连 PLC。
	// 生产环境必须留空。
	ProtocolOverride string `mapstructure:"protocol_override"`
}

type PLCDevice struct {
	Code string `mapstructure:"code" yaml:"code" json:"code"`
	Model string `mapstructure:"model" yaml:"model" json:"model"`
	Host string `mapstructure:"host" yaml:"host" json:"host"`
	Port int `mapstructure:"port" yaml:"port" json:"port"`
	Rack int `mapstructure:"rack" yaml:"rack" json:"rack"`
	Slot int `mapstructure:"slot" yaml:"slot" json:"slot"`
	Protocol string `mapstructure:"protocol" yaml:"protocol" json:"protocol"`
	ReadBlocks []PLCReadBlock `mapstructure:"read_blocks" yaml:"read_blocks" json:"read_blocks"`

	// PollGroups 是"按块读 + 按字段解释"的声明式点表。
	// 与 ReadBlocks 的区别：ReadBlocks 只关心字节变化，PollGroups 知道每个字段的语义。
	PollGroups []PLCPollGroup `mapstructure:"poll_groups" yaml:"poll_groups" json:"poll_groups"`
	// Handshake 声明 PLC↔PC 的请求-应答-确认序列。
	Handshake []PLCHandshakeStep `mapstructure:"handshake" yaml:"handshake" json:"handshake"`
	// Twin 描述双胞胎卷绕机：左右两侧共用同一个 DB，靠偏移量区分。
	Twin *PLCTwinWinder `mapstructure:"twin" yaml:"twin" json:"twin"`
}

type PLCReadBlock struct {
	DB    int    `mapstructure:"db"`
	Start int    `mapstructure:"start"`
	Size  int    `mapstructure:"size"`
	Tag   string `mapstructure:"tag"`
}

// PLCPollGroup 是一个按固定周期读取的连续数据块及其字段布局。
type PLCPollGroup struct {
	Name string `mapstructure:"name" yaml:"name" json:"name"`
	IntervalMs int `mapstructure:"interval_ms" yaml:"interval_ms" json:"interval_ms"`
	DB int `mapstructure:"db" yaml:"db" json:"db"`
	Start int `mapstructure:"start" yaml:"start" json:"start"`
	Length int `mapstructure:"length" yaml:"length" json:"length"`
	Points []PLCPoint `mapstructure:"points" yaml:"points" json:"points"`
	// Repeat 描述"同一个块在 DB 内重复 N 次"的布局（如 N 个卷绕机位）。
	Repeat *PLCRepeat `mapstructure:"repeat" yaml:"repeat" json:"repeat"`
}

// PLCRepeat 描述重复块（双胞胎卷绕机的一个机位）的布局。
//
// 一个机位（stride 字节）内，左右两侧各放一份 UDT：
//
//	偏移 0        ┌──────────┐
//	              │ 左侧 UDT │  ← sides.left.offset
//	偏移 size     ├──────────┤
//	              │ 右侧 UDT │  ← sides.right.offset
//	偏移 stride   └──────────┘
type PLCRepeat struct {
	Count int `mapstructure:"count" yaml:"count" json:"count"`
	Stride int `mapstructure:"stride" yaml:"stride" json:"stride"`
	Sides *PLCUdtSide `mapstructure:"sides" yaml:"sides" json:"sides"`
}

// PLCUdtSide 是双胞胎卷绕机左右两侧的起始偏移。
type PLCUdtSide struct {
	Left PLCSideOffset `mapstructure:"left" yaml:"left" json:"left"`
	Right PLCSideOffset `mapstructure:"right" yaml:"right" json:"right"`
}

type PLCSideOffset struct {
	Offset int `mapstructure:"offset" yaml:"offset" json:"offset"`
}

// PLCPoint 是数据块内的一个字段。
type PLCPoint struct {
	Name      string `mapstructure:"name"`
	Offset int `mapstructure:"offset" yaml:"offset" json:"offset"`
	Type      string `mapstructure:"type" yaml:"type"`           // bool/int16/uint16/int32/uint32/string/real
	Length    int    `mapstructure:"length" yaml:"length"`       // string 的字节长度
	Bit       int    `mapstructure:"bit" yaml:"bit"`             // bool 的位号
	Direction string `mapstructure:"direction" yaml:"direction"` // read/write
}

// PLCTwinWinder 描述双胞胎卷绕机（FDY）的左右两侧偏移。
//
// 语义来源：h028.sql 中 spinning_sides.settings.plc.dbs.Winder。
// 两侧必须收敛到同一步长：
//
//	offset_even + udt_size + offset_end_even == offset_odd + udt_size + offset_end_odd
//
// 生成的配置用 Sides 表达"哪一侧对应哪个偏移"，本结构保留原始四个值，
// 便于启动校验与现场核对。
type PLCTwinWinder struct {
	// UdtSize 是单个 UDT 的字节数（来自 udtTMT/udtBarmag）
	UdtSize int `mapstructure:"udt_size" yaml:"udt_size" json:"udt_size"`
	// Stride 是一个机位占用的总字节数
	Stride int `mapstructure:"stride" yaml:"stride" json:"stride"`
	OffsetEven int `mapstructure:"offset_even" yaml:"offset_even" json:"offset_even"`
	OffsetEndEven int `mapstructure:"offset_end_even" yaml:"offset_end_even" json:"offset_end_even"`
	OffsetOdd int `mapstructure:"offset_odd" yaml:"offset_odd" json:"offset_odd"`
	OffsetEndOdd int `mapstructure:"offset_end_odd" yaml:"offset_end_odd" json:"offset_end_odd"`
	Count int `mapstructure:"count" yaml:"count" json:"count"`
	// Sides 指明左/右分别使用 Even 还是 Odd；nil 表示单面锭位（无左右之分）
	Sides *PLCUdtSide `mapstructure:"sides" yaml:"sides" json:"sides"`
	// Semantics 记录该映射的推导依据，便于未来追溯
	Semantics string `mapstructure:"semantics" yaml:"semantics" json:"semantics"`
}

// PLCHandshakeStep 是握手序列中的一步（等待 / 动作 / 写入）。
type PLCHandshakeStep struct {
	Wait   *PLCWait  `mapstructure:"wait"`
	Action string    `mapstructure:"action"`
	Write  *PLCWrite `mapstructure:"write"`
}

type PLCWait struct {
	Point string `mapstructure:"point"`
	Value int    `mapstructure:"value"`
}

type PLCWrite struct {
	Point string `mapstructure:"point"`
	Value int    `mapstructure:"value"`
}

type EdgePrinter struct {
	Code string `mapstructure:"code" json:"code"`
	// Device 是该打印机服务于哪台设备（对应 PLCDevice.Code）。
	// 一台设备下挂多台打标机时，靠 Device + Head 定位具体是哪一台。
	Device string `mapstructure:"device" json:"device"`
	// Head 是该打印机在设备上的头号（1 起，对应 PLC 的 Start[i]/Done[i]）。
	Head int `mapstructure:"head" json:"head"`
	Host string `mapstructure:"host" json:"host"`
	Port int    `mapstructure:"port" json:"port"`
	// Label 是打标机内的模板名（Macsa 选模板帧使用），留空则用 "mae3"
	Label string `mapstructure:"label" json:"label"`
}

// EdgePalletizer 把一台打包机 PLC 与业务标识绑定。
//
// 打包机的 UDT 布局随现场机型不同（旧系统按 palletizers.settings 逐台配），
// 所以这里只声明"哪台设备对应哪个打包机号"，信号布局留在 PLC 点表里，
// 由现场配置下发。
type EdgePalletizer struct {
	// Code 是打包机编码，仅用于日志与界面显示
	Code string `mapstructure:"code" json:"code"`
	// Device 是它绑定的 PLC 设备编码（对应 PLCDevice.Code）
	Device string `mapstructure:"device" json:"device"`
	// Number 是打包机号，用于拼栈板编码（旧库 palletizers.code：11/12/13）
	Number int `mapstructure:"number" json:"number"`
	// PlantCode 是厂编码，栈板编码的前缀（旧库 settings.palletPlantCode）。
	// 留空则该段为空，编码仍然可用，只是少了工厂标识。
	PlantCode string `mapstructure:"plant_code" json:"plant_code"`
	// LabelPrinter 是这台的栈板标签打印机编码（对应 printers 段的 code），
	// 留空表示这台不打印标签。
	LabelPrinter string `mapstructure:"label_printer" json:"label_printer"`
}

// EdgeWarehouse 把一台仓库 PLC 与仓库编码绑定。
//
// 旧库里一个 PLC（192.168.5.43）挂了 4 个仓库（DB 901~904），
// 共享一个 PLC↔PC 交互块（DB 560，偏移 0/18/36/54）。
// 信号字节布局随现场不同，所以这里只声明"哪台设备对应哪个仓库"，
// 字段布局留在 PLC 点表里，由现场配置下发。
type EdgeWarehouse struct {
	// Code 是仓库编码，用于日志与界面显示
	Code string `mapstructure:"code" json:"code"`
	// Device 是它绑定的 PLC 设备编码（对应 PLCDevice.Code）
	Device string `mapstructure:"device" json:"device"`
	// WarehouseCode 是上报中心时用的仓库编码（warehouses.code）。
	// 留空则用 Code。
	WarehouseCode string `mapstructure:"warehouse_code" json:"warehouse_code"`
}

type EdgeHA struct {
	Mode     string `mapstructure:"mode"`
	Peer     string `mapstructure:"peer"`
	VIP      string `mapstructure:"vip"`
	Priority int    `mapstructure:"priority"`
}

type EdgeLine struct {
	Code string `mapstructure:"code"`
	Name string `mapstructure:"name"`
}

func LoadEdge(configDir string) (*EdgeConfig, error) {
	v := viper.New()
	v.SetConfigName("edge")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)
	v.AutomaticEnv()
	v.SetEnvPrefix("IGH")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read edge config: %w", err)	}

	expandEnvValues(v)

	var cfg EdgeConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal edge config: %w", err)
	}

	// 记录配置目录的绝对路径。标签模板跟随配置分发，
	// 靠"当前工作目录"去猜在服务方式启动时不可靠。
	if abs, err := filepath.Abs(configDir); err == nil {
		cfg.Edge.ConfigDir = abs
	} else {
		cfg.Edge.ConfigDir = configDir
	}

	if cfg.Edge.DataDir == "" {
		cfg.Edge.DataDir = "./data"
	}
	if cfg.Edge.HTTP.Port == 0 {
		cfg.Edge.HTTP.Port = 8080
	}
	if cfg.Edge.HTTP.Host == "" {
		cfg.Edge.HTTP.Host = "0.0.0.0"
	}
	if cfg.Edge.Server.HeartbeatSec == 0 {
		cfg.Edge.Server.HeartbeatSec = 10
	}
	if cfg.Edge.Server.SyncIntervalSec == 0 {
		cfg.Edge.Server.SyncIntervalSec = 30
	}
	if cfg.Edge.PLC.PollIntervalMs == 0 {
		cfg.Edge.PLC.PollIntervalMs = 1000
	}
	if cfg.Edge.HA.Mode == "" {
		cfg.Edge.HA.Mode = "standalone"
	}

	// 合并真实点表（生成产物），并做一致性校验。
	if err := mergeInstanceFiles(&cfg, configDir); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// mergeInstanceFiles 把 edge.plc.instance_files 中列出的点表合并进设备列表。
//
// 规则：
//   - 同 code 的设备：点表文件优先（它是从遗留系统生成的权威数据）
//   - strict 模式下，若同 code 设备在 edge.yaml 里显式写了不同的 DB 号，直接报错
//
// 这样既支持"点表独立维护"，也支持"现场临时覆盖"，同时不会静默用错地址。
func mergeInstanceFiles(cfg *EdgeConfig, configDir string) error {
	existing := make(map[string]*PLCDevice, len(cfg.Edge.PLC.Devices))
	for i := range cfg.Edge.PLC.Devices {
		d := &cfg.Edge.PLC.Devices[i]
		existing[d.Code] = d
	}

	for _, rel := range cfg.Edge.PLC.InstanceFiles {
		path := rel
		if !filepath.IsAbs(path) {
			path = filepath.Join(configDir, rel)
		}
		doc, err := LoadPLCInstances(path)
		if err != nil {
			return fmt.Errorf("merge instance file %s: %w", rel, err)
		}

		for i := range doc.Devices {
			incoming := doc.Devices[i]

			prev, ok := existing[incoming.Code]
			if !ok {
				cfg.Edge.PLC.Devices = append(cfg.Edge.PLC.Devices, incoming)
				existing[incoming.Code] = &cfg.Edge.PLC.Devices[len(cfg.Edge.PLC.Devices)-1]
				continue
			}

			if cfg.Edge.PLC.Strict {
				if err := compareDeviceDBs(prev, &incoming); err != nil {
					return fmt.Errorf("device %s: %w", incoming.Code, err)
				}
			}
			// 点表优先：保留 edge.yaml 中的连接参数，用点表覆盖布局与轮询组
			merged := incoming
			if prev.Host != "" {
				merged.Host = prev.Host
			}
			if prev.Port != 0 {
				merged.Port = prev.Port
			}
			if prev.Protocol != "" {
				merged.Protocol = prev.Protocol
			}
			*prev = merged
		}
	}

	// 合并后统一校验
	for i := range cfg.Edge.PLC.Devices {
		if cfg.Edge.PLC.Strict && cfg.Edge.PLC.ProtocolOverride != "" &&
			cfg.Edge.PLC.ProtocolOverride != "simulation" {
			return fmt.Errorf("protocol_override %q is only allowed to be \"simulation\"",
				cfg.Edge.PLC.ProtocolOverride)
		}
		if cfg.Edge.PLC.ProtocolOverride != "" {
			cfg.Edge.PLC.Devices[i].Protocol = cfg.Edge.PLC.ProtocolOverride
		}
		if err := ValidatePLCDevice(&cfg.Edge.PLC.Devices[i]); err != nil {
			return fmt.Errorf("device %s: %w", cfg.Edge.PLC.Devices[i].Code, err)
		}
	}
	return nil
}

// compareDeviceDBs 比较两处配置里的 DB 号是否一致。
func compareDeviceDBs(a, b *PLCDevice) error {
	aDB := firstPollDB(a)
	bDB := firstPollDB(b)
	if aDB == 0 || bDB == 0 {
		return nil // 任一方没有 poll_groups，无从比较
	}
	if aDB != bDB {
		return fmt.Errorf("db mismatch: edge.yaml=%d, instance=%d", aDB, bDB)
	}
	return nil
}

func firstPollDB(d *PLCDevice) int {
	if len(d.PollGroups) == 0 {
		return 0
	}
	return d.PollGroups[0].DB
}
