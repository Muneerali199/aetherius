package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/agent/internal/deployment"
	"github.com/aetherius/platform/agent/internal/hardware"
	"github.com/aetherius/platform/agent/internal/heartbeat"
	"github.com/aetherius/platform/agent/internal/metrics"
	"github.com/aetherius/platform/agent/internal/terminal"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("starting aetherius node agent")

	serverURL := getEnv("AETHERIUS_SERVER_URL", "http://localhost:8082")
	apiToken := getEnv("AETHERIUS_TOKEN", "")
	agentVersion := getEnv("AETHERIUS_AGENT_VERSION", "go-agent-1.0.0")

	if apiToken == "" {
		log.Fatal().Msg("AETHERIUS_TOKEN is required")
	}

	detector := hardware.NewDetector()
	hwInfo, err := detector.DetectAll()
	if err != nil {
		log.Fatal().Err(err).Msg("hardware detection failed")
	}

	log.Info().Int("gpus", len(hwInfo.GPUs)).Int("cpu_cores", hwInfo.CPU.Cores).
		Str("os", hwInfo.OSName).Msg("hardware detected")

	// Start terminal server
	termPort := getEnvInt("AETHERIUS_TERMINAL_PORT", 8086)
	termServer := terminal.NewServer(termPort)
	if err := termServer.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to start terminal server")
	}

	nodeID, err := registerWithPlatform(serverURL, apiToken, hwInfo, agentVersion, termPort)
	if err != nil {
		log.Fatal().Err(err).Msg("registration failed")
	}

	log.Info().Str("node_id", nodeID).Msg("registered with platform")

	hostname, _ := os.Hostname()
	agentURL := fmt.Sprintf("http://%s:%d", hostname, termPort)

	collector := metrics.NewCollector()
	hbClient := heartbeat.NewClient(serverURL, nodeID, apiToken, agentURL, collector)
	hbClient.SetInterval(30 * time.Second)

	depRunner := deployment.NewRunner(serverURL, apiToken, nodeID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Info().Msg("shutting down agent")
		termServer.Stop()
		cancel()
	}()

	go func() {
		if err := depRunner.Start(ctx); err != nil {
			log.Error().Err(err).Msg("deployment runner stopped")
		}
	}()

	if err := hbClient.Start(ctx); err != nil {
		log.Error().Err(err).Msg("heartbeat stopped")
	}
}

func registerWithPlatform(serverURL, apiToken string, hw *hardware.HardwareInfo, agentVersion string, termPort int) (string, error) {
	var gpus []map[string]interface{}
	for _, g := range hw.GPUs {
		gpus = append(gpus, map[string]interface{}{
			"model": g.Model,
			"vram_bytes": g.VRAMBytes,
			"cores": 16384,
		})
	}

	payload := map[string]interface{}{
		"agent_version": agentVersion,
		"agent_url":     fmt.Sprintf("http://%s:%d", hw.Network.PublicIP, termPort),
		"public_ip":     "",
		"hardware": map[string]interface{}{
			"gpus": gpus,
			"cpu": map[string]interface{}{
				"model": hw.CPU.Model,
				"cores": hw.CPU.Cores,
			},
			"ram": map[string]interface{}{
				"total_bytes": hw.RAM.TotalBytes,
			},
			"disk": map[string]interface{}{
				"total_bytes": hw.Disk.TotalBytes,
				"filesystem":  hw.Disk.Filesystem,
			},
			"network": map[string]interface{}{
				"speed_mbps":  hw.Network.SpeedMbps,
				"public_ip":   hw.Network.PublicIP,
				"provider_ip": "",
			},
			"cuda_version":   hw.CUDAVersion,
			"docker_version": hw.DockerVersion,
			"os_name":        hw.OSName,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest("POST", serverURL+"/v1/nodes/register", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("register request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return "", fmt.Errorf("registration failed (%d): %s", resp.StatusCode, errResp.Error)
	}

	var regResp struct {
		NodeID string `json:"node_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return regResp.NodeID, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}
