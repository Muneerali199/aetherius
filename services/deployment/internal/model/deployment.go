package model

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentStatus string

const (
	DeployStatusPending    DeploymentStatus = "pending"
	DeployStatusScheduling DeploymentStatus = "scheduling"
	DeployStatusRunning    DeploymentStatus = "running"
	DeployStatusStopping   DeploymentStatus = "stopping"
	DeployStatusStopped    DeploymentStatus = "stopped"
	DeployStatusFailed     DeploymentStatus = "failed"
)

type Deployment struct {
	ID             uuid.UUID        `json:"id" db:"id"`
	UserID         uuid.UUID        `json:"user_id" db:"user_id"`
	NodeID         *uuid.UUID       `json:"node_id,omitempty" db:"node_id"`
	Image          string           `json:"image" db:"image"`
	GPURequired    int              `json:"gpu_required" db:"gpu_required"`
	VRAMRequiredGB int64            `json:"vram_required_gb" db:"vram_required_gb"`
	RAMRequiredGB  int64            `json:"ram_required_gb" db:"ram_required_gb"`
	DiskRequiredGB int64            `json:"disk_required_gb" db:"disk_required_gb"`
	Ports          string           `json:"ports" db:"ports"`
	Env            string           `json:"env" db:"env"`
	Status         DeploymentStatus `json:"status" db:"status"`
	CostPerHour    float64          `json:"cost_per_hour" db:"cost_per_hour"`
	Region         string           `json:"region" db:"region"`
	AssignedAt     *time.Time       `json:"assigned_at,omitempty" db:"assigned_at"`
	CreatedAt      time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at" db:"updated_at"`
}
