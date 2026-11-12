package plc

import (
	"fmt"

	"github.com/igh-platform/igh-platform/internal/shared/config"
)

type Protocol interface {
	Connect() error
	Disconnect()
	IsConnected() bool
	ReadArea(dbNumber int, start int, size int) ([]byte, error)
	WriteArea(dbNumber int, start int, data []byte) error
}

type ReadBlock struct {
	DB    int    `mapstructure:"db" yaml:"db"`
	Start int    `mapstructure:"start" yaml:"start"`
	Size  int    `mapstructure:"size" yaml:"size"`
	Tag   string `mapstructure:"tag" yaml:"tag"`
}

type ReadResult struct {
	DeviceCode string
	Tag        string
	DB         int
	Start      int
	Data       []byte
}

func NewProtocol(protocol, host string, port, rack, slot int) (Protocol, error) {
	return NewProtocolWithDevice(protocol, host, port, rack, slot, nil)
}

// NewProtocolWithDevice 与 NewProtocol 相同，但允许传入设备配置。
// simulation 协议会据此按真实点表生成数据；其它协议忽略 dev。
func NewProtocolWithDevice(protocol, host string, port, rack, slot int, dev *config.PLCDevice) (Protocol, error) {
	switch protocol {
	case "s7", "":
		return newS7Protocol(host, port, rack, slot), nil
	case "modbus":
		return newModbusProtocol(host, port), nil
	case "simulation":
		if dev != nil {
			return newSimulationProtocolWithProfile(NewSimProfile(*dev)), nil
		}
		return newSimulationProtocol(), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
