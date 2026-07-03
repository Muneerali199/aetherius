package model

import (
	"time"

	"github.com/google/uuid"
)

type UserProfile struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Bio         string    `json:"bio" db:"bio"`
	Website     string    `json:"website" db:"website"`
	GithubUser  string    `json:"github_user" db:"github_user"`
	Timezone    string    `json:"timezone" db:"timezone"`
	NotifyEmail bool      `json:"notify_email" db:"notify_email"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type ApiKey struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Name      string     `json:"name" db:"name"`
	KeyPrefix string     `json:"key_prefix" db:"key_prefix"`
	KeyHash   string     `json:"-" db:"key_hash"`
	LastUsed  *time.Time `json:"last_used,omitempty" db:"last_used"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}
