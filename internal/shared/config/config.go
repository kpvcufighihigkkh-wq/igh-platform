package config

import (
	"fmt"
	"os"
	"path/filepath"
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
}

type Database struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	Name           string `mapstructure:"name"`
	User           string `mapstructure:"user"`
	Password       string `mapstructure:"password"`
	MaxConnections int    `mapstructure:"max_connections"`
}

func (d Database) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		d.Host, d.Port, d.User, d.Password, d.Name)
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
