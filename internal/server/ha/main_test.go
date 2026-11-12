package ha

import (
	"os"
	"testing"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

// TestMain 初始化日志：Start 与角色变更都会写日志，
// 而 logger.L 未初始化时是 nil，直接调用会 panic。
func TestMain(m *testing.M) {
	logger.Init("error")
	os.Exit(m.Run())
}
