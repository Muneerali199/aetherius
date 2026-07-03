package model

import (
	"time"
)

type SystemHealth struct {
	Status    string          `json:"status"`
	Uptime    string          `json:"uptime"`
	Version   string          `json:"version"`
	Services  []ServiceStatus `json:"services"`
	Timestamp time.Time       `json:"timestamp"`
}

type ServiceStatus struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Latency   int64     `json:"latency_ms"`
	LastCheck time.Time `json:"last_check"`
}

type MetricsSnapshot struct {
	ActiveNodes        int     `json:"active_nodes"`
	PendingNodes       int     `json:"pending_nodes"`
	TotalDeployments   int     `json:"total_deployments"`
	RunningDeployments int     `json:"running_deployments"`
	TotalGPU           int     `json:"total_gpu"`
	UsedGPU            int     `json:"used_gpu"`
	TotalRAMGB         float64 `json:"total_ram_gb"`
	UsedRAMGB          float64 `json:"used_ram_gb"`
	TotalStorageTB     float64 `json:"total_storage_tb"`
	UsedStorageTB      float64 `json:"used_storage_tb"`
	ActiveUsers        int     `json:"active_users"`
	Revenue24h         float64 `json:"revenue_24h"`
	RevenueTotal       float64 `json:"revenue_total"`
}

type Alert struct {
	ID           string    `json:"id"`
	Severity     string    `json:"severity"`
	Title        string    `json:"title"`
	Message      string    `json:"message"`
	Service      string    `json:"service"`
	Timestamp    time.Time `json:"timestamp"`
	Acknowledged bool      `json:"acknowledged"`
}
