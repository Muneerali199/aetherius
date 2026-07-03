package model

import (
	"time"

	"github.com/google/uuid"
)

type InvoiceStatus string

const (
	InvoiceStatusPending   InvoiceStatus = "pending"
	InvoiceStatusPaid      InvoiceStatus = "paid"
	InvoiceStatusOverdue   InvoiceStatus = "overdue"
	InvoiceStatusCancelled InvoiceStatus = "cancelled"
)

type Invoice struct {
	ID           uuid.UUID     `json:"id" db:"id"`
	UserID       uuid.UUID     `json:"user_id" db:"user_id"`
	DeploymentID *uuid.UUID    `json:"deployment_id,omitempty" db:"deployment_id"`
	Amount       float64       `json:"amount" db:"amount"`
	Currency     string        `json:"currency" db:"currency"`
	Status       InvoiceStatus `json:"status" db:"status"`
	Description  string        `json:"description" db:"description"`
	DueDate      time.Time     `json:"due_date" db:"due_date"`
	PaidAt       *time.Time    `json:"paid_at,omitempty" db:"paid_at"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at" db:"updated_at"`
}

type UsageRecord struct {
	ID           uuid.UUID `json:"id" db:"id"`
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	DeploymentID uuid.UUID `json:"deployment_id" db:"deployment_id"`
	GPUHours     float64   `json:"gpu_hours" db:"gpu_hours"`
	VRAMGBHours  float64   `json:"vram_gb_hours" db:"vram_gb_hours"`
	RAMGBHours   float64   `json:"ram_gb_hours" db:"ram_gb_hours"`
	Cost         float64   `json:"cost" db:"cost"`
	RecordedAt   time.Time `json:"recorded_at" db:"recorded_at"`
}
