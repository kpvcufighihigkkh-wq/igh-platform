package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type ProjectConfig struct {
	Project  Project  `mapstructure:"project"`
	Server   Server   `mapstructure:"server"`
	HA       HA       `mapstructure:"ha"`
	Database Database `mapstructure:"database"`
	Redis    Redis    `mapstructure:"redis"`
	Shifts   []Shift  `mapstructure:"shifts"`
	LotRules LotRules `mapstructure:"lot_rules"`
	Modules  Modules  `mapstructure:"modules"`
	// Printers 是中心管理端可用的打印机。
	//
	// 与边端 edge.yaml 的 printers 是两套：边端那套给工位页面直接打标用，
	// 这里这套给中心"补打标签"用。分开是因为它们通常不是同一台设备，
	// 而且中心的打印机在厂区网络里，边端的不一定连得上中心。
	Printers []Printer `mapstructure:"printers"`
	// Archive 是数据归档策略。不配则用下面的默认值，但 enabled 默认关闭。
	Archive Archive `mapstructure:"archive"`
}

// Archive 是数据归档配置。
//
// 为什么默认关闭：归档会删数据。首次上线前必须先用 dry_run 演练一遍，
// 确认"会删什么"符合预期，再显式打开。默认开着太危险。
type Archive struct {
	Enabled bool `mapstructure:"enabled"`
	// Dir 是归档文件输出目录，相对路径按进程工作目录解析。
	Dir string `mapstructure:"dir"`
	// DoffingDays / EventDays / ERPDays 是三张表各自的保留天数。
	DoffingDays int `mapstructure:"doffing_days"`
	EventDays   int `mapstructure:"event_days"`
	ERPDays     int `mapstructure:"erp_days"`
	// Hour 是每天定时执行的小时（0-23），默认凌晨 3 点。
	// 放在夜间是为了避开产线高峰：删除会短暂持锁。
	Hour int `mapstructure:"hour"`
	// DryRun 为真时只统计不删除。首次上线建议先开这个跑一天。
	DryRun bool `mapstructure:"dry_run"`
}

// Defaults 补齐缺省值。保留期按现场确定的策略给默认值。
func (a Archive) Defaults() Archive {
	if a.Dir == "" {
		a.Dir = "./archive"
	}
	if a.DoffingDays <= 0 {
		a.DoffingDays = 365
	}
	if a.EventDays <= 0 {
		a.EventDays = 30
	}
	if a.ERPDays <= 0 {
		a.ERPDays = 15
	}
	if a.Hour <= 0 || a.Hour > 23 {
		a.Hour = 3
	}
	return a
}

// Printer 是一台打印机。
type Printer struct {
	Code string `mapstructure:"code"`
	Name string `mapstructure:"name"`
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type Project struct {
	Name     string `mapstructure:"name"`
	Code     string `mapstructure:"code"`
	Customer string `mapstructure:"customer"`
	Timezone string `mapstructure:"timezone"`
}

type Server struct {
	HTTPPort int    `mapstructure:"http_port"`
	WSPort   int    `mapstructure:"ws_port"`
	APIKey   string `mapstructure:"api_key"`
}

type HA struct {
	Mode               string `mapstructure:"mode"`
	VIP                string `mapstructure:"vip"`
	Peer               string `mapstructure:"peer"`
	HeartbeatInterface string `mapstructure:"heartbeat_interface"`
	// StateFilePath 是 keepalived 的 notify 脚本写入本机角色的文件。
	//
	// 中心双机采用"两头常驻 + VIP 漂移"：两台服务器上的 igh-server 都在跑，
	// 谁持有 VIP 谁对外服务。igh-server 读这个文件就知道自己的真实角色，
	// 而不是像以前那样"配置写了 active-standby 就假定自己是主"。
	// 为空时用默认路径 /var/lib/igh/keepalived.state。
	StateFilePath string `mapstructure:"state_file"`
}

type Database struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	// Hosts 是多地址主备列表，元素形如 "10.0.0.1:5432"。
	//
	// 配了它之后 host 不再生效：连接时按顺序尝试，且只接受**可写**的那台。
	// 数据库切主后，中心服务重连时能自己找到新主库，不需要人工改配置再重启——
	// 那正是"切换完成了、应用还在连旧库"的根因。
	Hosts          []string `mapstructure:"hosts"`
	Name           string   `mapstructure:"name"`
	User           string   `mapstructure:"user"`
	Password       string   `mapstructure:"password"`
	MaxConnections int      `mapstructure:"max_connections"`
}

// MultiHost 表示是否走多地址模式。
func (d Database) MultiHost() bool { return len(d.Hosts) > 0 }

// DSN 生成 libpq 连接串。
//
// 单地址保持原样、一字不改：现网部署全依赖这个字符串，
// 多塞一个参数都可能改变连接行为。
//
// 多地址额外做三件事：
//   - host 与 port 两个列表一一对应（libpq 两者都支持逗号分隔）
//   - target_session_attrs=read-write：跳过备库和被围栏的旧主库。
//     它们都能连上、都能读，但写入会失败；靠这条让驱动在连接阶段就淘汰。
//   - connect_timeout=2：整机掉电时 TCP 会等到内核超时，
//     不设上限的话"换下一个地址"这一步就能吃掉十几秒。
func (d Database) DSN() string {
	if !d.MultiHost() {
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			d.Host, d.Port, d.User, d.Password, d.Name)
	}

	defaultPort := d.Port
	if defaultPort == 0 {
		defaultPort = 5432
	}

	hosts := make([]string, 0, len(d.Hosts))
	ports := make([]string, 0, len(d.Hosts))
	for _, h := range d.Hosts {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		host, port := h, defaultPort
		if i := strings.LastIndex(h, ":"); i >= 0 {
			if n, err := strconv.Atoi(h[i+1:]); err == nil {
				host, port = h[:i], n
			}
		}
		hosts = append(hosts, host)
		ports = append(ports, strconv.Itoa(port))
	}

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable "+
		"target_session_attrs=read-write connect_timeout=2",
		strings.Join(hosts, ","), strings.Join(ports, ","), d.User, d.Password, d.Name)
}

type Redis struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
}

type Shift struct {
	Code  int    `mapstructure:"code"`
	Name  string `mapstructure:"name"`
	Start string `mapstructure:"start"`
	End   string `mapstructure:"end"`
}

type LotRules struct {
	PrefixSource  string `mapstructure:"prefix_source"`
	CodeFormat    string `mapstructure:"code_format"`
	BarcodeFormat string `mapstructure:"barcode_format"`
	BarcodeReset  string `mapstructure:"barcode_reset"`
}

type Modules struct {
	Winding     bool `mapstructure:"winding"`
	Quality     bool `mapstructure:"quality"`
	Warehouse   bool `mapstructure:"warehouse"`
	Packaging   bool `mapstructure:"packaging"`
	ERP         bool `mapstructure:"erp"`
	Maintenance bool `mapstructure:"maintenance"`
	Vehicle     bool `mapstructure:"vehicle"`
}

func Load(configDir string) (*ProjectConfig, error) {
	v := viper.New()
	v.SetConfigName("project")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read project config: %w", err)
	}

	expandEnvValues(v)

	var cfg ProjectConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal project config: %w", err)
	}
	return &cfg, nil
}

func ConfigDir() string {
	if dir := os.Getenv("IGH_CONFIG_DIR"); dir != "" {
		return dir
	}
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "configs")
}

func expandEnvValues(v *viper.Viper) {
	for _, key := range v.AllKeys() {
		val := v.GetString(key)
		if strings.HasPrefix(val, "${") && strings.HasSuffix(val, "}") {
			envKey := val[2 : len(val)-1]
			if envVal := os.Getenv(envKey); envVal != "" {
				v.Set(key, envVal)
			}
		}
	}
}
