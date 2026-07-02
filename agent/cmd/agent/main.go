package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/agent/internal/hardware"
	"github.com/aetherius/platform/agent/internal/heartbeat"
)

type MetricsCollector struct {
	detector *hardware.Detector
}

func (m *MetricsCollector) GetGPUUtilization() ([]float64, error) {
	return nil, nil
}

func (m *MetricsCollector) GetGPUTemps() ([]float64, error) {
	return nil, nil
}

func (m *MetricsCollector) GetVRAMUsed() ([]int64, error) {
	return nil, nil
}

func (m *MetricsCollector) GetCPUUtilization() (float64, error) {
	return 0, nil
}

func (m *MetricsCollector) GetRAMUsedGB() (int64, error) {
	return 0, nil
}

func (m *MetricsCollector) GetDiskUsedGB() (int64, error) {
	return 0, nil
}

func (m *MetricsCollector) GetNetworkRXBytes() (int64, error) {
	return 0, nil
}

func (m *MetricsCollector) GetNetworkTXBytes() (int64, error) {
	return 0, nil
}

func (m *MetricsCollector) GetLoadAverage() (float64, error) {
	return 0, nil
}

func (m *MetricsCollector) GetUptimeSeconds() (int64, error) {
	return 0, nil
}

func (m *MetricsCollector) GetRunningContainers() ([]string, error) {
	return nil, nil
}

type RegistrationResponse struct {
	NodeID    string `json:"node_id"`
	NodeToken string `json:"node_token"`
	Status    string `json:"status"`
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("starting aetherius node agent")

	serverURL := getEnv("AETHERIUS_SERVER_URL", "http://localhost:8082")
	providerIDStr := getEnv("AETHERIUS_PROVIDER_ID", "")
	agentVersion := getEnv("AETHERIUS_AGENT_VERSION", "0.1.0")

	if providerIDStr == "" {
		log.Fatal().Msg("AETHERIUS_PROVIDER_ID is required")
	}

	providerID, err := uuid.Parse(providerIDStr)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid AETHERIUS_PROVIDER_ID")
	}

	detector := hardware.NewDetector()
	hwInfo, err := detector.DetectAll()
	if err != nil {
		log.Fatal().Err(err).Msg("hardware detection failed")
	}

	log.Info().Int("gpus", len(hwInfo.GPUs)).Int("cpu_cores", hwInfo.CPU.Cores).
		Str("os", hwInfo.OSName).Msg("hardware detected")

	// Register with the platform
	nodeID, nodeToken, err := registerWithPlatform(serverURL, providerID, hwInfo, agentVersion)
	if err != nil {
		log.Fatal().Err(err).Msg("registration failed")
	}

	log.Info().Str("node_id", nodeID).Msg("registered with platform")

	// Start heartbeat
	collector := &MetricsCollector{detector: detector}
	hbClient := heartbeat.NewClient(serverURL, nodeID, nodeToken, collector)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Info().Msg("shutting down agent")
		cancel()
	}()

	if err := hbClient.Start(ctx); err != nil {
		log.Error().Err(err).Msg("heartbeat stopped")
	}
}

func registerWithPlatform(serverURL string, providerID uuid.UUID, hw *hardware.HardwareInfo, agentVersion string) (string, string, error) {
	payload := map[string]interface{}{
		"provider_id":   providerID.String(),
		"agent_version": agentVersion,
		"public_ip":     "",
		"hardware": map[string]interface{}{
			"gpus":    hw.GPUs,
			"cpu":     hw.CPU,
			"ram":     hw.RAM,
			"disk":    hw.Disk,
			"network": hw.Network,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("marshal: %w", err)
	}

	resp, err := http.Post(serverURL+"/v1/nodes/register", "application/json", nil)
	if err != nil {
		return "", "", fmt.Errorf("register: %w", err)
	}
	defer resp.Body.Close()
	_ = body

	var regResp RegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return "", "", fmt.Errorf("decode: %w", err)
	}

	return regResp.NodeID, regResp.NodeToken, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
