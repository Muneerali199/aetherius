package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	Email             string     `json:"email" db:"email"`
	PasswordHash      string     `json:"-" db:"password_hash"`
	DisplayName       string     `json:"display_name" db:"display_name"`
	AvatarURL         *string    `json:"avatar_url,omitempty" db:"avatar_url"`
	EmailVerified     bool       `json:"email_verified" db:"email_verified"`
	EmailVerifyToken  *uuid.UUID `json:"-" db:"email_verify_token"`
	MFAEnabled        bool       `json:"mfa_enabled" db:"mfa_enabled"`
	MFASecret         *string    `json:"-" db:"mfa_secret"`
	MFABackupCodes    []string   `json:"-" db:"mfa_backup_codes"`
	LastLoginAt       *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt         *time.Time `json:"-" db:"deleted_at"`
}

type Session struct {
	ID               uuid.UUID `json:"id" db:"id"`
	UserID           uuid.UUID `json:"user_id" db:"user_id"`
	RefreshTokenHash string    `json:"-" db:"refresh_token_hash"`
	IPAddress        *string   `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent        *string   `json:"user_agent,omitempty" db:"user_agent"`
	ExpiresAt        time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

type OAuthAccount struct {
	ID             uuid.UUID `json:"id" db:"id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	Provider       string    `json:"provider" db:"provider"`
	ProviderUserID string    `json:"provider_user_id" db:"provider_user_id"`
	AccessToken    *string   `json:"-" db:"access_token"`
	RefreshToken   *string   `json:"-" db:"refresh_token"`
	TokenExpiresAt *time.Time `json:"token_expires_at,omitempty" db:"token_expires_at"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}
