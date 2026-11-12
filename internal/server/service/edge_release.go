package service

import (
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"gorm.io/gorm"
)

type EdgeReleaseService struct {
	db      *gorm.DB
	dataDir string
}

func NewEdgeReleaseService(db *gorm.DB, dataDir string) *EdgeReleaseService {
	return &EdgeReleaseService{db: db, dataDir: dataDir}
}

func (s *EdgeReleaseService) releaseDir() string {
	return filepath.Join(s.dataDir, "releases")
}

func (s *EdgeReleaseService) Upload(version, platform, releaseNote string, file multipart.File, header *multipart.FileHeader) (*model.EdgeRelease, error) {
	dir := s.releaseDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create release dir: %w", err)
	}

	storagePath := filepath.Join(dir, fmt.Sprintf("igh-edge-%s-%s%s", version, platform, filepath.Ext(header.Filename)))

	out, err := os.Create(storagePath)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}

	hasher := sha256.New()
	writer := io.MultiWriter(out, hasher)
	size, err := io.Copy(writer, file)
	out.Close()
	if err != nil {
		os.Remove(storagePath)
		return nil, fmt.Errorf("write file: %w", err)
	}

	checksum := fmt.Sprintf("sha256:%x", hasher.Sum(nil))

	release := &model.EdgeRelease{
		Version:     version,
		Platform:    platform,
		FileName:    header.Filename,
		FileSize:    size,
		Checksum:    checksum,
		StoragePath: storagePath,
		ReleaseNote: releaseNote,
	}
	if err := s.db.Create(release).Error; err != nil {
		os.Remove(storagePath)
		return nil, err
	}

	return release, nil
}

func (s *EdgeReleaseService) List() ([]model.EdgeRelease, error) {
	var releases []model.EdgeRelease
	err := s.db.Order("created_at desc").Find(&releases).Error
	return releases, err
}

func (s *EdgeReleaseService) Get(version string) (*model.EdgeRelease, error) {
	var release model.EdgeRelease
	if err := s.db.Where("version = ?", version).First(&release).Error; err != nil {
		return nil, err
	}
	return &release, nil
}

func (s *EdgeReleaseService) Delete(version string) error {
	var release model.EdgeRelease
	if err := s.db.Where("version = ?", version).First(&release).Error; err != nil {
		return err
	}
	os.Remove(release.StoragePath)
	return s.db.Delete(&release).Error
}

func (s *EdgeReleaseService) FilePath(version string) (string, error) {
	release, err := s.Get(version)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(release.StoragePath); err != nil {
		return "", fmt.Errorf("release file not found on disk")
	}
	return release.StoragePath, nil
}

func (s *EdgeReleaseService) SetTargetVersion(edgeCode, version string) error {
	return s.db.Model(&model.EdgeNode{}).Where("code = ?", edgeCode).
		Update("target_version", version).Error
}

func (s *EdgeReleaseService) BatchSetTargetVersion(edgeCodes []string, version string) error {
	return s.db.Model(&model.EdgeNode{}).Where("code IN ?", edgeCodes).
		Update("target_version", version).Error
}

func (s *EdgeReleaseService) ListUpdateLogs(edgeCode string, limit int) ([]model.EdgeUpdateLog, error) {
	if limit <= 0 {
		limit = 50
	}
	var logs []model.EdgeUpdateLog
	q := s.db.Order("started_at desc").Limit(limit)
	if edgeCode != "" {
		q = q.Where("edge_code = ?", edgeCode)
	}
	err := q.Find(&logs).Error
	return logs, err
}

func (s *EdgeReleaseService) RecordUpdateStart(edgeCode, fromVer, toVer string) (*model.EdgeUpdateLog, error) {
	log := &model.EdgeUpdateLog{
		EdgeCode:    edgeCode,
		FromVersion: fromVer,
		ToVersion:   toVer,
		Status:      "downloading",
		StartedAt:   time.Now(),
	}
	if err := s.db.Create(log).Error; err != nil {
		return nil, err
	}
	return log, nil
}

func (s *EdgeReleaseService) RecordUpdateResult(logID int64, status, errMsg string) error {
	now := time.Now()
	return s.db.Model(&model.EdgeUpdateLog{}).Where("id = ?", logID).Updates(map[string]any{
		"status":        status,
		"error_message": errMsg,
		"completed_at":  &now,
	}).Error
}

// GetOTAInfoForHeartbeat returns target version info for heartbeat response.
func (s *EdgeReleaseService) GetOTAInfoForHeartbeat(edgeCode, currentVersion string) (targetVer, downloadURL, checksum string) {
	var node model.EdgeNode
	if err := s.db.Where("code = ?", edgeCode).First(&node).Error; err != nil {
		return "", "", ""
	}

	if node.TargetVersion == "" || node.TargetVersion == currentVersion {
		return "", "", ""
	}

	release, err := s.Get(node.TargetVersion)
	if err != nil {
		return "", "", ""
	}

	return node.TargetVersion,
		fmt.Sprintf("/api/v1/edge/release/download/%s", node.TargetVersion),
		release.Checksum
}
