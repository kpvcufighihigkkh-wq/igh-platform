package updater

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew_InitialState(t *testing.T) {
	u := New("v1.0.0", "http://localhost", "key", t.TempDir())
	s := u.Status()

	if s.State != "idle" {
		t.Errorf("state = %q, want %q", s.State, "idle")
	}
	if s.CurrentVer != "v1.0.0" {
		t.Errorf("current_ver = %q, want %q", s.CurrentVer, "v1.0.0")
	}
	if s.Available {
		t.Error("should not be available initially")
	}
}

func TestNotifyNewVersion_SetsAvailable(t *testing.T) {
	u := New("v1.0.0", "http://localhost", "key", t.TempDir())

	u.NotifyNewVersion("v1.1.0", "/download/v1.1.0", "sha256:abc")
	s := u.Status()

	if s.State != "available" {
		t.Errorf("state = %q, want %q", s.State, "available")
	}
	if !s.Available {
		t.Error("should be available after notification")
	}
	if s.TargetVer != "v1.1.0" {
		t.Errorf("target_ver = %q, want %q", s.TargetVer, "v1.1.0")
	}
}

func TestNotifyNewVersion_IgnoredDuringDownload(t *testing.T) {
	u := New("v1.0.0", "http://localhost", "key", t.TempDir())

	u.mu.Lock()
	u.status.State = "downloading"
	u.mu.Unlock()

	u.NotifyNewVersion("v2.0.0", "/download/v2.0.0", "sha256:xyz")
	s := u.Status()

	if s.State != "downloading" {
		t.Errorf("state = %q, want %q", s.State, "downloading")
	}
	if s.TargetVer == "v2.0.0" {
		t.Error("should not update target during download")
	}
}

func TestStartDownload_FailsInWrongState(t *testing.T) {
	u := New("v1.0.0", "http://localhost", "key", t.TempDir())

	if err := u.StartDownload(); err == nil {
		t.Error("should fail in idle state")
	}
}

func TestInstall_FailsInWrongState(t *testing.T) {
	u := New("v1.0.0", "http://localhost", "key", t.TempDir())

	if err := u.Install(); err == nil {
		t.Error("should fail in idle state")
	}
}

func TestStartDownload_Success(t *testing.T) {
	binaryContent := []byte("fake-binary-content-for-testing")
	h := sha256.Sum256(binaryContent)
	checksum := fmt.Sprintf("sha256:%x", h)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(binaryContent)))
		w.Write(binaryContent)
	}))
	defer srv.Close()

	dir := t.TempDir()
	u := New("v1.0.0", srv.URL, "test-key", dir)

	u.NotifyNewVersion("v1.1.0", "/api/v1/edge/release/download/v1.1.0", checksum)

	if err := u.StartDownload(); err != nil {
		t.Fatalf("start download: %v", err)
	}

	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for download, state=%q", u.Status().State)
		default:
			s := u.Status()
			if s.State == "ready" {
				if s.DownloadPct != 100 {
					t.Errorf("download_pct = %d, want 100", s.DownloadPct)
				}
				return
			}
			if s.State == "failed" {
				t.Fatalf("download failed: %s", s.ErrorMessage)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func TestStartDownload_BadChecksum(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("some-content"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	u := New("v1.0.0", srv.URL, "", dir)
	u.NotifyNewVersion("v1.1.0", "/download", "sha256:bad-checksum")

	u.StartDownload()

	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout")
		default:
			s := u.Status()
			if s.State == "failed" {
				if s.ErrorMessage == "" {
					t.Error("expected error message for bad checksum")
				}
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func TestCheckRollback_NoMarker(t *testing.T) {
	dir := t.TempDir()
	u := New("v1.0.0", "http://localhost", "key", dir)
	u.CheckRollback()
}

func TestCheckRollback_SuccessfulUpdate(t *testing.T) {
	dir := t.TempDir()
	updateDir := filepath.Join(dir, "update")
	os.MkdirAll(updateDir, 0o755)

	markerPath := filepath.Join(updateDir, ".updating")
	os.WriteFile(markerPath, []byte("v1.1.0"), 0o644)

	u := New("v1.1.0", "http://localhost", "key", dir)
	u.CheckRollback()

	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Error("marker file should be removed after successful update")
	}
}
