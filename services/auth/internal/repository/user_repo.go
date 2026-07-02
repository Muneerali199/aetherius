package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/aetherius/platform/services/auth/internal/model"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, display_name, email_verify_token)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.DisplayName, user.EmailVerifyToken,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if isDuplicateKey(err) {
			return ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, display_name, avatar_url,
		       email_verified, email_verify_token, mfa_enabled, mfa_secret,
		       mfa_backup_codes, last_login_at, created_at, updated_at, deleted_at
		FROM users WHERE id = $1 AND deleted_at IS NULL`

	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName,
		&user.AvatarURL, &user.EmailVerified, &user.EmailVerifyToken,
		&user.MFAEnabled, &user.MFASecret, &user.MFABackupCodes,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return user, err
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, display_name, avatar_url,
		       email_verified, email_verify_token, mfa_enabled, mfa_secret,
		       mfa_backup_codes, last_login_at, created_at, updated_at, deleted_at
		FROM users WHERE email = $1 AND deleted_at IS NULL`

	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName,
		&user.AvatarURL, &user.EmailVerified, &user.EmailVerifyToken,
		&user.MFAEnabled, &user.MFASecret, &user.MFABackupCodes,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return user, err
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET last_login_at = $1, updated_at = $1 WHERE id = $2`,
		time.Now(), id,
	)
	return err
}

func (r *UserRepository) VerifyEmail(ctx context.Context, token uuid.UUID) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE users SET email_verified = TRUE, email_verify_token = NULL, updated_at = NOW()
		 WHERE email_verify_token = $1 AND email_verified = FALSE`,
		token,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) EnableMFA(ctx context.Context, userID uuid.UUID, secret string, backupCodes []string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET mfa_enabled = TRUE, mfa_secret = $1, mfa_backup_codes = $2, updated_at = NOW()
		 WHERE id = $3`,
		secret, backupCodes, userID,
	)
	return err
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		passwordHash, userID,
	)
	return err
}

func (r *UserRepository) CreateSession(ctx context.Context, session *model.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, refresh_token_hash, ip_address, user_agent, device_info, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query,
		session.ID, session.UserID, session.RefreshTokenHash,
		session.IPAddress, session.UserAgent, nil, session.ExpiresAt,
	)
	return err
}

func (r *UserRepository) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
	return err
}

func (r *UserRepository) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

func (r *UserRepository) GetOAuthAccount(ctx context.Context, provider, providerUserID string) (*model.OAuthAccount, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, access_token, refresh_token, token_expires_at, created_at
		FROM oauth_accounts WHERE provider = $1 AND provider_user_id = $2`

	acc := &model.OAuthAccount{}
	err := r.pool.QueryRow(ctx, query, provider, providerUserID).Scan(
		&acc.ID, &acc.UserID, &acc.Provider, &acc.ProviderUserID,
		&acc.AccessToken, &acc.RefreshToken, &acc.TokenExpiresAt, &acc.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return acc, err
}

func (r *UserRepository) LinkOAuth(ctx context.Context, acc *model.OAuthAccount) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO oauth_accounts (id, user_id, provider, provider_user_id, access_token, refresh_token, token_expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (provider, provider_user_id) DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			token_expires_at = EXCLUDED.token_expires_at`,
		acc.ID, acc.UserID, acc.Provider, acc.ProviderUserID,
		acc.AccessToken, acc.RefreshToken, acc.TokenExpiresAt,
	)
	return err
}

func isDuplicateKey(err error) bool {
	return err != nil && contains(err.Error(), "duplicate key")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
