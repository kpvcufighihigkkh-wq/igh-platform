package updater

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

type Status struct {
	Available    bool   `json:"available"`
	CurrentVer   string `json:"current_version"`
	TargetVer    string `json:"target_version,omitempty"`
	DownloadURL  string `json:"download_url,omitempty"`
	Checksum     string `json:"checksum,omitempty"`
	DownloadPct  int    `json:"download_pct"`
	State        string `json:"state"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type Updater struct {
	mu          sync.RWMutex
	currentVer  string
	status      Status
	serverURL   string
	apiKey      string
	dataDir     string
	execPath    string
}

func New(currentVer, serverURL, apiKey, dataDir string) *Updater {
	execPath, _ := os.Executable()
	return &Updater{
		currentVer: currentVer,
		serverURL:  serverURL,
		apiKey:     apiKey,
		dataDir:    dataDir,
		execPath:   execPath,
		status: Status{
			CurrentVer: currentVer,
			State:      "idle",
		},
	}
}

// NotifyNewVersion implements sync.UpdateNotifier.
func (u *Updater) NotifyNewVersion(targetVer, downloadURL, checksum string) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.status.State == "downloading" || u.status.State == "ready" || u.status.State == "installing" {
		return
	}

	u.status = Status{
		Available:   true,
		CurrentVer:  u.currentVer,
		TargetVer:   targetVer,
		DownloadURL: downloadURL,
		Checksum:    checksum,
		State:       "available",
	}
}

func (u *Updater) Status() Status {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.status
}

func (u *Updater) StartDownload() error {
	u.mu.Lock()
	if u.status.State != "available" {
		u.mu.Unlock()
		return fmt.Errorf("cannot download in state %q", u.status.State)
	}
	u.status.State = "downloading"
	u.status.DownloadPct = 0
	downloadURL := u.status.DownloadURL
	checksum := u.status.Checksum
	targetVer := u.status.TargetVer
	u.mu.Unlock()

	go u.doDownload(downloadURL, checksum, targetVer)
	return nil
}

func (u *Updater) Install() error {
	u.mu.Lock()
	if u.status.State != "ready" {
		u.mu.Unlock()
		return fmt.Errorf("cannot install in state %q", u.status.State)
	}
	u.status.State = "installing"
	u.mu.Unlock()

	go u.doInstall()
	return nil
}

func (u *Updater) updateDir() string {
	return filepath.Join(u.dataDir, "update")
}

func (u *Updater) backupDir() string {
	return filepath.Join(u.dataDir, "backup")
}

func (u *Updater) downloadedPath() string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return filepath.Join(u.updateDir(), "igh-edge.new"+ext)
}

func (u *Updater) doDownload(downloadURL, expectedChecksum, targetVer string) {
	fullURL := u.serverURL + downloadURL

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		u.setError(fmt.Sprintf("create request: %v", err))
		return
	}
	req.Header.Set("X-API-Key", u.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		u.setError(fmt.Sprintf("download: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		u.setError(fmt.Sprintf("download failed: HTTP %d", resp.StatusCode))
		return
	}

	os.MkdirAll(u.updateDir(), 0o755)
	destPath := u.downloadedPath()

	out, err := os.Create(destPath)
	if err != nil {
		u.setError(fmt.Sprintf("create file: %v", err))
		return
	}

	hasher := sha256.New()
	writer := io.MultiWriter(out, hasher)

	totalSize := resp.ContentLength
	var written int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := writer.Write(buf[:n])
			if writeErr != nil {
				out.Close()
				os.Remove(destPath)
				u.setError(fmt.Sprintf("write: %v", writeErr))
				return
			}
			written += int64(n)
			if totalSize > 0 {
				pct := int(written * 100 / totalSize)
				u.mu.Lock()
				u.status.DownloadPct = pct
				u.mu.Unlock()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			out.Close()
			os.Remove(destPath)
			u.setError(fmt.Sprintf("read: %v", readErr))
			return
		}
	}
	out.Close()

	actualChecksum := fmt.Sprintf("sha256:%x", hasher.Sum(nil))
	if expectedChecksum != "" && actualChecksum != expectedChecksum {
		os.Remove(destPath)
		u.setError(fmt.Sprintf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum))
		return
	}

	u.mu.Lock()
	u.status.State = "ready"
	u.status.DownloadPct = 100
	u.mu.Unlock()

	if logger.L != nil {
		logger.L.Infow("update downloaded and verified", "version", targetVer)
	}
}

func (u *Updater) doInstall() {
	os.MkdirAll(u.backupDir(), 0o755)

	backupPath := filepath.Join(u.backupDir(), filepath.Base(u.execPath)+".bak")
	if err := copyFile(u.execPath, backupPath); err != nil {
		u.setError(fmt.Sprintf("backup failed: %v", err))
		return
	}

	markerPath := filepath.Join(u.updateDir(), ".updating")
	os.WriteFile(markerPath, []byte(u.status.TargetVer), 0o644)

	newBinary := u.downloadedPath()

	if err := replaceBinary(u.execPath, newBinary); err != nil {
		u.setError(fmt.Sprintf("replace binary failed: %v", err))
		os.Remove(markerPath)
		return
	}

	if logger.L != nil {
		logger.L.Infow("binary replaced, exiting for restart", "target", u.status.TargetVer)
	}

	os.Exit(0)
}

func (u *Updater) setError(msg string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.status.State = "failed"
	u.status.ErrorMessage = msg
	if logger.L != nil {
		logger.L.Warnw("update error", "error", msg)
	}
}

// CheckRollback checks if we're recovering from a failed update.
// Call on startup before other components.
func (u *Updater) CheckRollback() {
	markerPath := filepath.Join(u.updateDir(), ".updating")
	data, err := os.ReadFile(markerPath)
	if err != nil {
		return
	}

	targetVer := string(data)
	if logger.L != nil {
		logger.L.Infow("update marker found, checking health", "target_version", targetVer)
	}

	if u.currentVer == targetVer {
		os.Remove(markerPath)
		cleanupDir(u.backupDir())
		cleanupDir(u.updateDir())
		if logger.L != nil {
			logger.L.Infow("update successful, cleanup done", "version", targetVer)
		}
		return
	}

	if logger.L != nil {
		logger.L.Warnw("update may have failed, current version doesn't match target",
			"current", u.currentVer, "target", targetVer)
	}

	backupPath := filepath.Join(u.backupDir(), filepath.Base(u.execPath)+".bak")
	if _, err := os.Stat(backupPath); err == nil {
		if logger.L != nil {
			logger.L.Warnw("rolling back to backup", "backup", backupPath)
		}
		if err := replaceBinary(u.execPath, backupPath); err == nil {
			os.Remove(markerPath)
			os.Exit(1)
		}
	}

	os.Remove(markerPath)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func cleanupDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		os.Remove(filepath.Join(dir, e.Name()))
	}
}
