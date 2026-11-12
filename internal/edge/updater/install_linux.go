//go:build !windows

package updater

import "os"

func replaceBinary(execPath, newBinaryPath string) error {
	return os.Rename(newBinaryPath, execPath)
}
