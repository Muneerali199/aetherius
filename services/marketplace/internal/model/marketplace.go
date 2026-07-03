package model

import (
	"time"

	"github.com/google/uuid"
)

type ListingStatus string

const (
	ListingStatusActive   ListingStatus = "active"
	ListingStatusRented   ListingStatus = "rented"
	ListingStatusInactive ListingStatus = "inactive"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusActive    OrderStatus = "active"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type Listing struct {
	ID           uuid.UUID     `json:"id" db:"id"`
	NodeID       uuid.UUID     `json:"node_id" db:"node_id"`
	OwnerID      uuid.UUID     `json:"owner_id" db:"owner_id"`
	GPUModel     string        `json:"gpu_model" db:"gpu_model"`
	GPUCount     int           `json:"gpu_count" db:"gpu_count"`
	VRAMGB       int64         `json:"vram_gb" db:"vram_gb"`
	RAMGB        int64         `json:"ram_gb" db:"ram_gb"`
	DiskGB       int64         `json:"disk_gb" db:"disk_gb"`
	PricePerHour float64       `json:"price_per_hour" db:"price_per_hour"`
	Region       string        `json:"region" db:"region"`
	Status       ListingStatus `json:"status" db:"status"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at" db:"updated_at"`
}

type Order struct {
	ID        uuid.UUID   `json:"id" db:"id"`
	ListingID uuid.UUID   `json:"listing_id" db:"listing_id"`
	RenterID  uuid.UUID   `json:"renter_id" db:"renter_id"`
	Status    OrderStatus `json:"status" db:"status"`
	StartedAt *time.Time  `json:"started_at,omitempty" db:"started_at"`
	EndedAt   *time.Time  `json:"ended_at,omitempty" db:"ended_at"`
	CreatedAt time.Time   `json:"created_at" db:"created_at"`
}
