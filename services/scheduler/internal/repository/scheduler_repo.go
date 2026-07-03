package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aetherius/platform/services/scheduler/internal/model"
)

type SchedulerRepository struct {
	pool *pgxpool.Pool
}

func NewSchedulerRepository(pool *pgxpool.Pool) *SchedulerRepository {
	return &SchedulerRepository{pool: pool}
}

func (r *SchedulerRepository) CreateDeployment(ctx context.Context, d *model.Deployment) error {
	portsJSON, _ := json.Marshal(d.Ports)
	envJSON, _ := json.Marshal(d.Env)

	query := `INSERT INTO deployments (
		id, user_id, node_id, image, gpu_required, vram_required_gb, ram_required_gb,
		disk_required_gb, ports, env, status, cost_per_hour, region, assigned_at,
		created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
	)`

	_, err := r.pool.Exec(ctx, query,
		d.ID, d.UserID, d.NodeID, d.Image, d.GPURequired, d.VRAMRequiredGB,
		d.RAMRequiredGB, d.DiskRequiredGB, portsJSON, envJSON, d.Status,
		d.CostPerHour, d.Region, d.AssignedAt, d.CreatedAt, d.UpdatedAt,
	)
	return err
}

func (r *SchedulerRepository) ListDeploymentsByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Deployment, error) {
	query := `SELECT id, user_id, node_id, image, gpu_required, vram_required_gb,
		ram_required_gb, disk_required_gb, ports, env, status, cost_per_hour,
		region, assigned_at, created_at, updated_at
		FROM deployments WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDeployments(rows)
}

func (r *SchedulerRepository) ListPendingDeployments(ctx context.Context) ([]*model.Deployment, error) {
	query := `SELECT id, user_id, node_id, image, gpu_required, vram_required_gb,
		ram_required_gb, disk_required_gb, ports, env, status, cost_per_hour,
		region, assigned_at, created_at, updated_at
		FROM deployments WHERE status = $1 ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, model.DeployStatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDeployments(rows)
}

func (r *SchedulerRepository) GetDeploymentByID(ctx context.Context, id uuid.UUID) (*model.Deployment, error) {
	query := `SELECT id, user_id, node_id, image, gpu_required, vram_required_gb,
		ram_required_gb, disk_required_gb, ports, env, status, cost_per_hour,
		region, assigned_at, created_at, updated_at
		FROM deployments WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	return scanDeployment(row)
}

func (r *SchedulerRepository) AssignNode(ctx context.Context, deploymentID uuid.UUID, nodeID uuid.UUID) error {
	query := `UPDATE deployments SET node_id = $1, status = $2, assigned_at = $3, updated_at = $4 WHERE id = $5`
	_, err := r.pool.Exec(ctx, query, nodeID, model.DeployStatusScheduling, time.Now(), time.Now(), deploymentID)
	return err
}

func (r *SchedulerRepository) UpdateStatus(ctx context.Context, deploymentID uuid.UUID, status model.DeploymentStatus) error {
	query := `UPDATE deployments SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, status, time.Now(), deploymentID)
	return err
}

type scannable interface {
	Scan(dest ...any) error
}

func scanDeployments(rows pgx.Rows) ([]*model.Deployment, error) {
	var deployments []*model.Deployment
	for rows.Next() {
		d, err := scanDeploymentRow(rows)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, d)
	}
	return deployments, rows.Err()
}

func scanDeployment(row scannable) (*model.Deployment, error) {
	return scanDeploymentRow(row)
}

func scanDeploymentRow(row scannable) (*model.Deployment, error) {
	d := &model.Deployment{}
	var nodeID *uuid.UUID
	var portsJSON, envJSON []byte
	var assignedAt *time.Time

	err := row.Scan(
		&d.ID, &d.UserID, &nodeID, &d.Image, &d.GPURequired, &d.VRAMRequiredGB,
		&d.RAMRequiredGB, &d.DiskRequiredGB, &portsJSON, &envJSON, &d.Status,
		&d.CostPerHour, &d.Region, &assignedAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	d.NodeID = nodeID
	d.AssignedAt = assignedAt

	if len(portsJSON) > 0 {
		json.Unmarshal(portsJSON, &d.Ports)
	}
	if len(envJSON) > 0 {
		json.Unmarshal(envJSON, &d.Env)
	}

	return d, nil
}
