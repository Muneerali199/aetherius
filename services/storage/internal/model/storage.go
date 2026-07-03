package model

import (
	"time"

	"github.com/google/uuid"
)

type StorageObject struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	DeploymentID *uuid.UUID `json:"deployment_id,omitempty" db:"deployment_id"`
	Bucket       string     `json:"bucket" db:"bucket"`
	Key          string     `json:"key" db:"key"`
	Filename     string     `json:"filename" db:"filename"`
	ContentType  string     `json:"content_type" db:"content_type"`
	Size         int64      `json:"size" db:"size"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}
