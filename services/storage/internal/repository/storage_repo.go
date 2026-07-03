package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aetherius/platform/services/storage/internal/model"
)

type StorageRepository struct {
	pool *pgxpool.Pool
}

func NewStorageRepository(pool *pgxpool.Pool) *StorageRepository {
	return &StorageRepository{pool: pool}
}

func (r *StorageRepository) Insert(ctx context.Context, obj *model.StorageObject) error {
	query := `
		INSERT INTO storage_objects (id, user_id, deployment_id, bucket, key, filename, content_type, size, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		obj.ID, obj.UserID, obj.DeploymentID, obj.Bucket, obj.Key,
		obj.Filename, obj.ContentType, obj.Size, obj.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert storage object: %w", err)
	}
	return nil
}

func (r *StorageRepository) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*model.StorageObject, error) {
	query := `
		SELECT id, user_id, deployment_id, bucket, key, filename, content_type, size, created_at
		FROM storage_objects
		WHERE id = $1 AND user_id = $2
	`
	obj := &model.StorageObject{}
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&obj.ID, &obj.UserID, &obj.DeploymentID, &obj.Bucket, &obj.Key,
		&obj.Filename, &obj.ContentType, &obj.Size, &obj.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get storage object: %w", err)
	}
	return obj, nil
}

func (r *StorageRepository) ListByUserID(ctx context.Context, userID uuid.UUID, bucket, prefix string) ([]*model.StorageObject, error) {
	query := `
		SELECT id, user_id, deployment_id, bucket, key, filename, content_type, size, created_at
		FROM storage_objects
		WHERE user_id = $1
	`
	args := []any{userID}
	argIdx := 2

	if bucket != "" {
		query += fmt.Sprintf(" AND bucket = $%d", argIdx)
		args = append(args, bucket)
		argIdx++
	}
	if prefix != "" {
		query += fmt.Sprintf(" AND key LIKE $%d", argIdx)
		args = append(args, prefix+"%")
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list storage objects: %w", err)
	}
	defer rows.Close()

	var objects []*model.StorageObject
	for rows.Next() {
		obj := &model.StorageObject{}
		if err := rows.Scan(
			&obj.ID, &obj.UserID, &obj.DeploymentID, &obj.Bucket, &obj.Key,
			&obj.Filename, &obj.ContentType, &obj.Size, &obj.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan storage object: %w", err)
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

func (r *StorageRepository) DeleteByIDAndUser(ctx context.Context, id, userID uuid.UUID) error {
	query := `DELETE FROM storage_objects WHERE id = $1 AND user_id = $2`
	tag, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("delete storage object: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("storage object not found")
	}
	return nil
}
