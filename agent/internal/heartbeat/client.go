package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type Client struct {
	serverURL   string
	nodeID      string
	nodeToken   string
	httpClient  *http.Client
	interval    time.Duration
	collector   MetricsCollector
}

type MetricsCollector interface {
	GetGPUUtilization() ([]float64, error)
	GetGPUTemps() ([]float64, error)
	GetVRAMUsed() ([]int64, error)
	GetCPUUtilization() (float64, error)
	GetRAMUsedGB() (int64, error)
	GetDiskUsedGB() (int64, error)
	GetNetworkRXBytes() (int64, error)
	GetNetworkTXBytes() (int64, error)
	GetLoadAverage() (float64, error)
	GetUptimeSeconds() (int64, error)
	GetRunningContainers() ([]string, error)
}

func NewClient(serverURL, nodeID, nodeToken string, collector MetricsCollector) *Client {
	return &Client{
		serverURL:  serverURL,
		nodeID:     nodeID,
		nodeToken:  nodeToken,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		interval:   5 * time.Second,
		collector:  collector,
	}
}

type HeartbeatPayload struct {
	NodeID            string    `json:"node_id"`
	NodeToken         string    `json:"node_token"`
	GPUUtil           []float64 `json:"gpu_util"`
	GPUTemps          []float64 `json:"gpu_temps"`
	VRAMUsed          []int64   `json:"vram_used"`
	CPUUtil           float64   `json:"cpu_util"`
	RAMUsedGB         int64     `json:"ram_used_gb"`
	DiskUsedGB        int64     `json:"disk_used_gb"`
	NetworkRXBytes    int64     `json:"network_rx_bytes"`
	NetworkTXBytes    int64     `json:"network_tx_bytes"`
	LoadAvg           float64   `json:"load_avg"`
	UptimeSeconds     int64     `json:"uptime_seconds"`
	RunningContainers []string  `json:"running_containers"`
}

type HeartbeatResponse struct {
	Accepted                bool   `json:"accepted"`
	NextHeartbeatIntervalMs int    `json:"next_heartbeat_interval_ms"`
}

func (c *Client) Start(ctx context.Context) error {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	log.Info().Str("node_id", c.nodeID).Dur("interval", c.interval).
		Msg("heartbeat client started")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.sendHeartbeat(); err != nil {
				log.Error().Err(err).Msg("heartbeat failed")
			}
		}
	}
}

func (c *Client) sendHeartbeat() error {
	gpuUtil, _ := c.collector.GetGPUUtilization()
	gpuTemps, _ := c.collector.GetGPUTemps()
	vramUsed, _ := c.collector.GetVRAMUsed()
	cpuUtil, _ := c.collector.GetCPUUtilization()
	ramUsed, _ := c.collector.GetRAMUsedGB()
	diskUsed, _ := c.collector.GetDiskUsedGB()
	rxBytes, _ := c.collector.GetNetworkRXBytes()
	txBytes, _ := c.collector.GetNetworkTXBytes()
	loadAvg, _ := c.collector.GetLoadAverage()
	uptime, _ := c.collector.GetUptimeSeconds()
	containers, _ := c.collector.GetRunningContainers()

	payload := HeartbeatPayload{
		NodeID:            c.nodeID,
		NodeToken:         c.nodeToken,
		GPUUtil:           gpuUtil,
		GPUTemps:          gpuTemps,
		VRAMUsed:          vramUsed,
		CPUUtil:           cpuUtil,
		RAMUsedGB:         ramUsed,
		DiskUsedGB:        diskUsed,
		NetworkRXBytes:    rxBytes,
		NetworkTXBytes:    txBytes,
		LoadAvg:           loadAvg,
		UptimeSeconds:     uptime,
		RunningContainers: containers,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal heartbeat: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.serverURL+"/v1/nodes/heartbeat",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat rejected: %d", resp.StatusCode)
	}

	var hbResp HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&hbResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if hbResp.NextHeartbeatIntervalMs > 0 {
		c.interval = time.Duration(hbResp.NextHeartbeatIntervalMs) * time.Millisecond
	}

	return nil
}
