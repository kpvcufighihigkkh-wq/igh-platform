package printer

import (
	"fmt"
	"net"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

type Printer struct {
	Code string
	Host string
	Port int
}

type Client struct {
	printers map[string]*Printer
}

func NewClient() *Client {
	return &Client{printers: make(map[string]*Printer)}
}

func (c *Client) Register(code, host string, port int) {
	c.printers[code] = &Printer{Code: code, Host: host, Port: port}
}

func (c *Client) Print(printerCode string, zplData []byte) error {
	p, ok := c.printers[printerCode]
	if !ok {
		return fmt.Errorf("printer %s not found", printerCode)
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", p.Host, p.Port), 3*time.Second)
	if err != nil {
		return fmt.Errorf("connect to printer %s: %w", printerCode, err)
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write(zplData); err != nil {
		return fmt.Errorf("write to printer %s: %w", printerCode, err)
	}

	logger.L.Infow("label printed", "printer", printerCode, "bytes", len(zplData))
	return nil
}
