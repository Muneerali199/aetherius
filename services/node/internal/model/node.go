package model

import (
	"time"

	"github.com/google/uuid"
)

type NodeStatus string

const (
	NodeStatusPending     NodeStatus = "pending"
	NodeStatusActive      NodeStatus = "active"
	NodeStatusPaused      NodeStatus = "paused"
	NodeStatusMaintenance NodeStatus = "maintenance"
	NodeStatusOffline     NodeStatus = "offline"
	NodeStatusBanned      NodeStatus = "banned"
)

type Node struct {
	ID                 uuid.UUID  `json:"id" db:"id"`
	ProviderID         uuid.UUID  `json:"provider_id" db:"provider_id"`
	Status             NodeStatus `json:"status" db:"status"`
	HardwareFingerprint string     `json:"hardware_fingerprint" db:"hardware_fingerprint"`
	BenchmarkScore     float64    `json:"benchmark_score" db:"benchmark_score"`
	ReputationScore    float64    `json:"reputation_score" db:"reputation_score"`
	TotalGPU           int        `json:"total_gpu" db:"total_gpu"`
	AvailableGPU       int        `json:"available_gpu" db:"available_gpu"`
	TotalVRAMGB        int64      `json:"total_vram_gb" db:"total_vram_gb"`
	AvailableVRAMGB    int64      `json:"available_vram_gb" db:"available_vram_gb"`
	TotalRAMGB         int64      `json:"total_ram_gb" db:"total_ram_gb"`
	AvailableRAMGB     int64      `json:"available_ram_gb" db:"available_ram_gb"`
	TotalDiskGB        int64      `json:"total_disk_gb" db:"total_disk_gb"`
	AvailableDiskGB    int64      `json:"available_disk_gb" db:"available_disk_gb"`
	CPUModel           string     `json:"cpu_model" db:"cpu_model"`
	CPUCores           int        `json:"cpu_cores" db:"cpu_cores"`
	GPUModels          []string   `json:"gpu_models" db:"gpu_models"`
	NetworkSpeedMbps   float64    `json:"network_speed_mbps" db:"network_speed_mbps"`
	AgentURL           string     `json:"agent_url,omitempty" db:"agent_url"`
	PublicIP           string     `json:"public_ip,omitempty" db:"public_ip"`
	Region             string     `json:"region" db:"region"`
	Country            string     `json:"country" db:"country"`
	City               string     `json:"city,omitempty" db:"city"`
	Latitude           float64    `json:"latitude" db:"latitude"`
	Longitude          float64    `json:"longitude" db:"longitude"`
	CUDAVersion        string     `json:"cuda_version,omitempty" db:"cuda_version"`
	DockerVersion      string     `json:"docker_version,omitempty" db:"docker_version"`
	OSName             string     `json:"os_name" db:"os_name"`
	AgentVersion       string     `json:"agent_version" db:"agent_version"`
	NodeToken          string     `json:"-" db:"node_token"`
	FirstSeen          time.Time  `json:"first_seen" db:"first_seen"`
	LastHeartbeat      time.Time  `json:"last_heartbeat" db:"last_heartbeat"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

type GPUUnit struct {
	ID            uuid.UUID `json:"id" db:"id"`
	NodeID        uuid.UUID `json:"node_id" db:"node_id"`
	GPUIndex      int       `json:"gpu_index" db:"gpu_index"`
	Model         string    `json:"model" db:"model"`
	VRAMGB        int64     `json:"vram_gb" db:"vram_gb"`
	VRAMType      string    `json:"vram_type" db:"vram_type"`
	CUDACores     int       `json:"cuda_cores" db:"cuda_cores"`
	TensorCores   int       `json:"tensor_cores" db:"tensor_cores"`
	ClockSpeedMHz int       `json:"clock_speed_mhz" db:"clock_speed_mhz"`
	UUID          string    `json:"uuid" db:"uuid"`
	Status        string    `json:"status" db:"status"`
}

type SSHKey struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	PublicKey   string    `json:"public_key" db:"public_key"`
	Fingerprint string    `json:"fingerprint" db:"fingerprint"`
	IsDefault   bool      `json:"is_default" db:"is_default"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type Heartbeat struct {
	ID               uuid.UUID `json:"id" db:"id"`
	NodeID           uuid.UUID `json:"node_id" db:"node_id"`
	Status           NodeStatus `json:"status" db:"status"`
	GPUUtilization   []float64 `json:"gpu_utilization" db:"gpu_utilization"`
	GPUTemp          []float64 `json:"gpu_temp" db:"gpu_temp"`
	VRAMUsed         []int64   `json:"vram_used" db:"vram_used"`
	CPUUtilization   float64   `json:"cpu_utilization" db:"cpu_utilization"`
	RAMUsedGB        int64     `json:"ram_used_gb" db:"ram_used_gb"`
	DiskUsedGB       int64     `json:"disk_used_gb" db:"disk_used_gb"`
	NetworkRXBytes   int64     `json:"network_rx_bytes" db:"network_rx_bytes"`
	NetworkTXBytes   int64     `json:"network_tx_bytes" db:"network_tx_bytes"`
	LoadAverage      float64   `json:"load_average" db:"load_average"`
	UptimeSeconds    int64     `json:"uptime_seconds" db:"uptime_seconds"`
	RunningContainer int       `json:"running_containers" db:"running_containers"`
	ReportedAt       time.Time `json:"reported_at" db:"reported_at"`
}
