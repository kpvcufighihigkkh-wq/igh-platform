package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

type LabelService struct {
	templateDir string
}

func NewLabelService(configDir string) *LabelService {
	return &LabelService{templateDir: filepath.Join(configDir, "labels", "templates")}
}

func (s *LabelService) RenderAndPrint(templateName string, variables map[string]string, printerHost string, printerPort int) error {
	tmplPath := filepath.Join(s.templateDir, templateName)
	tmplData, err := os.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("read template %s: %w", templateName, err)
	}

	zpl := string(tmplData)
	for k, v := range variables {
		zpl = strings.ReplaceAll(zpl, "{{"+k+"}}", v)
	}

	logger.L.Infow("label rendered", "template", templateName, "printer", printerHost, "size", len(zpl))

	// In production, send to printer via TCP
	// For now, log the rendered ZPL
	_ = printerHost
	_ = printerPort
	return nil
}

func (s *LabelService) ListTemplates() ([]string, error) {
	entries, err := os.ReadDir(s.templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
