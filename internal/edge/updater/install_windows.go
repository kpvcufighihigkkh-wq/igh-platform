//go:build windows

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func replaceBinary(execPath, newBinaryPath string) error {
	dir := filepath.Dir(execPath)
	scriptPath := filepath.Join(dir, "apply_update.bat")

	absExec, _ := filepath.Abs(execPath)
	absNew, _ := filepath.Abs(newBinaryPath)

	script := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
copy /Y "%s" "%s"
del "%s"
del "%%~f0"
`, absNew, absExec, absNew)

	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		return fmt.Errorf("write update script: %w", err)
	}

	cmd := exec.Command("cmd", "/C", "start", "/b", scriptPath)
	cmd.Dir = dir
	return cmd.Start()
}
