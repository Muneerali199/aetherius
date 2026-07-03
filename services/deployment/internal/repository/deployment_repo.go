package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aetherius/platform/services/deployment/internal/model"
)

type DeploymentRepo struct {
	pool *pgxpool.Pool
}

func NewDeploymentRepo(pool *pgxpool.Pool) *DeploymentRepo {
	return &DeploymentRepo{pool: pool}
}

const (
	insertDeployment = `INSERT INTO deployments
		(id, user_id, node_id, image, gpu_required, vram_required_gb, ram_required_gb,
		 disk_required_gb, ports, env, status, cost_per_hour, region, assigned_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

	selectDeployment = `SELECT id, user_id, node_id, image, gpu_required, vram_required_gb,
		ram_required_gb, disk_required_gb, ports, env, status, cost_per_hour, region,
		assigned_at, created_at, updated_at FROM deployments`

	updateStatus = `UPDATE deployments SET status = $1, updated_at = $2 WHERE id = $3`

	deleteDeployment = `DELETE FROM deployments WHERE id = $1`
)

func (r *DeploymentRepo) Create(ctx context.Context, d *model.Deployment) error {
	d.ID = uuid.New()
	now := time.Now().UTC()
	d.CreatedAt = now
	d.UpdatedAt = now

	_, err := r.pool.Exec(ctx, insertDeployment,
		d.ID, d.UserID, d.NodeID, d.Image, d.GPURequired, d.VRAMRequiredGB,
		d.RAMRequiredGB, d.DiskRequiredGB, d.Ports, d.Env, d.Status,
		d.CostPerHour, d.Region, d.AssignedAt, d.CreatedAt, d.UpdatedAt,
	)
	return err
}

func (r *DeploymentRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Deployment, error) {
	rows, err := r.pool.Query(ctx, selectDeployment+" WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	d, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Deployment])
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeploymentRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Deployment, error) {
	rows, err := r.pool.Query(ctx, selectDeployment+" WHERE user_id = $1 ORDER BY created_at DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ds, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[model.Deployment])
	if err != nil {
		return nil, err
	}
	return ds, nil
}

func (r *DeploymentRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeploymentStatus) error {
	_, err := r.pool.Exec(ctx, updateStatus, status, time.Now().UTC(), id)
	return err
}

func (r *DeploymentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, deleteDeployment, id)
	return err
}
