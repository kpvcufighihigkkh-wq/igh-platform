package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type VisionConfig struct {
	Endpoint string `yaml:"endpoint"`
	Timeout  int    `yaml:"timeout_sec"`
	Enabled  bool   `yaml:"enabled"`
}

type VisionService struct {
	config VisionConfig
	client *http.Client
}

type VisionRequest struct {
	BobbinID      string `json:"bobbin_id"`
	LotCode       string `json:"lot_code"`
	Specification string `json:"specification"`
	ProductType   string `json:"product_type"`
}

type VisionResponse struct {
	BobbinID      string  `json:"bobbin_id"`
	ImageURL      string  `json:"image_url"`
	Grade         string  `json:"grade"`
	DefectType    *string `json:"defect_type"`
	DefectDetail  *string `json:"defect_detail"`
	Confidence    float64 `json:"confidence"`
	InspectTimeMs int     `json:"inspect_time_ms"`
}

func NewVisionService(cfg VisionConfig) *VisionService {
	timeout := 10
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}
	return &VisionService{
		config: cfg,
		client: &http.Client{Timeout: time.Duration(timeout) * time.Second},
	}
}

func (s *VisionService) IsEnabled() bool {
	return s.config.Enabled && s.config.Endpoint != ""
}

func (s *VisionService) Inspect(req VisionRequest) (*VisionResponse, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("vision service disabled")
	}

	body, _ := json.Marshal(req)
	resp, err := s.client.Post(s.config.Endpoint+"/api/v1/inspect", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("vision request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("vision returned status %d", resp.StatusCode)
	}

	var result VisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode vision response: %w", err)
	}
	return &result, nil
}
