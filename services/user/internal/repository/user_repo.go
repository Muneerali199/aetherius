package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aetherius/platform/services/user/internal/model"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrKeyNotFound     = errors.New("api key not found")
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetOrCreateProfile(ctx context.Context, userID uuid.UUID) (*model.UserProfile, error) {
	profile, err := r.GetProfile(ctx, userID)
	if err == nil {
		return profile, nil
	}
	if !errors.Is(err, ErrProfileNotFound) {
		return nil, err
	}

	query := `
		INSERT INTO user_profiles (id, user_id, bio, website, github_user, timezone, notify_email)
		VALUES ($1, $2, '', '', '', 'UTC', false)
		ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
		RETURNING id, user_id, bio, website, github_user, timezone, notify_email, created_at, updated_at`

	profile = &model.UserProfile{}
	err = r.pool.QueryRow(ctx, query, uuid.New(), userID).Scan(
		&profile.ID, &profile.UserID, &profile.Bio, &profile.Website,
		&profile.GithubUser, &profile.Timezone, &profile.NotifyEmail,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	_, _ = r.pool.Exec(ctx,
		`INSERT INTO api_keys (id, user_id, name, key_prefix, key_hash)
		 VALUES ($1, $2, 'default', '', '')
		 ON CONFLICT DO NOTHING`,
		uuid.New(), userID,
	)

	return profile, nil
}

func (r *UserRepository) GetProfile(ctx context.Context, userID uuid.UUID) (*model.UserProfile, error) {
	query := `
		SELECT id, user_id, bio, website, github_user, timezone, notify_email, created_at, updated_at
		FROM user_profiles WHERE user_id = $1`

	profile := &model.UserProfile{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&profile.ID, &profile.UserID, &profile.Bio, &profile.Website,
		&profile.GithubUser, &profile.Timezone, &profile.NotifyEmail,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	return profile, err
}

func (r *UserRepository) UpdateProfile(ctx context.Context, userID uuid.UUID, bio, website, githubUser, timezone string, notifyEmail bool) (*model.UserProfile, error) {
	query := `
		UPDATE user_profiles
		SET bio = $1, website = $2, github_user = $3, timezone = $4, notify_email = $5, updated_at = NOW()
		WHERE user_id = $6
		RETURNING id, user_id, bio, website, github_user, timezone, notify_email, created_at, updated_at`

	profile := &model.UserProfile{}
	err := r.pool.QueryRow(ctx, query,
		bio, website, githubUser, timezone, notifyEmail, userID,
	).Scan(
		&profile.ID, &profile.UserID, &profile.Bio, &profile.Website,
		&profile.GithubUser, &profile.Timezone, &profile.NotifyEmail,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	return profile, err
}

func (r *UserRepository) CreateApiKey(ctx context.Context, userID uuid.UUID, name, keyPrefix, keyHash string) (*model.ApiKey, error) {
	query := `
		INSERT INTO api_keys (id, user_id, name, key_prefix, key_hash)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, name, key_prefix, key_hash, last_used, created_at`

	key := &model.ApiKey{}
	err := r.pool.QueryRow(ctx, query,
		uuid.New(), userID, name, keyPrefix, keyHash,
	).Scan(
		&key.ID, &key.UserID, &key.Name, &key.KeyPrefix, &key.KeyHash, &key.LastUsed, &key.CreatedAt,
	)
	return key, err
}

func (r *UserRepository) ListApiKeys(ctx context.Context, userID uuid.UUID) ([]*model.ApiKey, error) {
	query := `
		SELECT id, user_id, name, key_prefix, key_hash, last_used, created_at
		FROM api_keys
		WHERE user_id = $1 AND key_hash != ''
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*model.ApiKey
	for rows.Next() {
		k := &model.ApiKey{}
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.KeyHash, &k.LastUsed, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *UserRepository) DeleteApiKey(ctx context.Context, id, userID uuid.UUID) error {
	result, err := r.pool.Exec(ctx,
		`DELETE FROM api_keys WHERE id = $1 AND user_id = $2 AND key_hash != ''`,
		id, userID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrKeyNotFound
	}
	return nil
}

func (r *UserRepository) GetApiKeyByID(ctx context.Context, id, userID uuid.UUID) (*model.ApiKey, error) {
	query := `
		SELECT id, user_id, name, key_prefix, key_hash, last_used, created_at
		FROM api_keys WHERE id = $1 AND user_id = $2 AND key_hash != ''`

	key := &model.ApiKey{}
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&key.ID, &key.UserID, &key.Name, &key.KeyPrefix, &key.KeyHash, &key.LastUsed, &key.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrKeyNotFound
	}
	return key, err
}

func (r *UserRepository) TouchApiKey(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE api_keys SET last_used = NOW() WHERE id = $1`,
		id,
	)
	return err
}

func (r *UserRepository) GetApiKeyByHash(ctx context.Context, keyHash string) (*model.ApiKey, error) {
	query := `
		SELECT id, user_id, name, key_prefix, key_hash, last_used, created_at
		FROM api_keys WHERE key_hash = $1`

	key := &model.ApiKey{}
	err := r.pool.QueryRow(ctx, query, keyHash).Scan(
		&key.ID, &key.UserID, &key.Name, &key.KeyPrefix, &key.KeyHash, &key.LastUsed, &key.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrKeyNotFound
	}
	return key, err
}
