package archive

import (
	"os"
	"testing"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

// TestMain 初始化日志：归档过程会打日志，
// logger.L 未初始化时是 nil，直接调用会 panic。
func TestMain(m *testing.M) {
	logger.Init("error")
	os.Exit(m.Run())
}
