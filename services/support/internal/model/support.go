package model

import (
	"time"

	"github.com/google/uuid"
)

type TicketStatus string

const (
	TicketOpen      TicketStatus = "open"
	TicketPending   TicketStatus = "pending"
	TicketResolved  TicketStatus = "resolved"
	TicketClosed    TicketStatus = "closed"
)

type TicketPriority string

const (
	PriorityLow      TicketPriority = "low"
	PriorityMedium   TicketPriority = "medium"
	PriorityHigh     TicketPriority = "high"
	PriorityCritical TicketPriority = "critical"
)

type Ticket struct {
	ID        uuid.UUID      `json:"id" db:"id"`
	UserID    uuid.UUID      `json:"user_id" db:"user_id"`
	Subject   string         `json:"subject" db:"subject"`
	Status    TicketStatus   `json:"status" db:"status"`
	Priority  TicketPriority `json:"priority" db:"priority"`
	Category  string         `json:"category" db:"category"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

type TicketMessage struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TicketID  uuid.UUID `json:"ticket_id" db:"ticket_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Content   string    `json:"content" db:"content"`
	IsStaff   bool      `json:"is_staff" db:"is_staff"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
