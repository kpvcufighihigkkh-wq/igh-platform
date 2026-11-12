package app

import (
	"os"
	"path/filepath"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

func getDiskFreeMB(dir string) int64 {
	abs, err := filepath.Abs(dir)
	if err != nil {
		logger.L.Warnw("disk check: cannot resolve path", "dir", dir, "error", err)
		return -1
	}
	return diskFreeMB(abs)
}

func dataDirSizeMB(dir string) int64 {
	var total int64
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total / 1024 / 1024
}
