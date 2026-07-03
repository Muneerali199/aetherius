package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aetherius/platform/services/billing/internal/model"
)

var (
	ErrInvoiceNotFound = errors.New("invoice not found")
)

type BillingRepository struct {
	pool *pgxpool.Pool
}

func NewBillingRepository(pool *pgxpool.Pool) *BillingRepository {
	return &BillingRepository{pool: pool}
}

func (r *BillingRepository) CreateInvoice(ctx context.Context, invoice *model.Invoice) error {
	query := `
		INSERT INTO invoices (id, user_id, deployment_id, amount, currency, status, description, due_date, paid_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		invoice.ID, invoice.UserID, invoice.DeploymentID,
		invoice.Amount, invoice.Currency, invoice.Status,
		invoice.Description, invoice.DueDate, invoice.PaidAt,
	).Scan(&invoice.CreatedAt, &invoice.UpdatedAt)
	return err
}

func (r *BillingRepository) ListInvoicesByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Invoice, error) {
	query := `
		SELECT id, user_id, deployment_id, amount, currency, status, description, due_date, paid_at, created_at, updated_at
		FROM invoices
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*model.Invoice
	for rows.Next() {
		var inv model.Invoice
		if err := rows.Scan(
			&inv.ID, &inv.UserID, &inv.DeploymentID,
			&inv.Amount, &inv.Currency, &inv.Status,
			&inv.Description, &inv.DueDate, &inv.PaidAt,
			&inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, err
		}
		invoices = append(invoices, &inv)
	}
	return invoices, rows.Err()
}

func (r *BillingRepository) GetInvoiceByID(ctx context.Context, id uuid.UUID) (*model.Invoice, error) {
	query := `
		SELECT id, user_id, deployment_id, amount, currency, status, description, due_date, paid_at, created_at, updated_at
		FROM invoices WHERE id = $1`

	var inv model.Invoice
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&inv.ID, &inv.UserID, &inv.DeploymentID,
		&inv.Amount, &inv.Currency, &inv.Status,
		&inv.Description, &inv.DueDate, &inv.PaidAt,
		&inv.CreatedAt, &inv.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvoiceNotFound
	}
	return &inv, err
}

func (r *BillingRepository) UpdateInvoiceStatus(ctx context.Context, id uuid.UUID, status model.InvoiceStatus, paidAt *time.Time) error {
	query := `UPDATE invoices SET status = $1, paid_at = $2, updated_at = $3 WHERE id = $4`
	result, err := r.pool.Exec(ctx, query, status, paidAt, time.Now(), id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrInvoiceNotFound
	}
	return nil
}

func (r *BillingRepository) CreateUsageRecord(ctx context.Context, record *model.UsageRecord) error {
	query := `
		INSERT INTO usage_records (id, user_id, deployment_id, gpu_hours, vram_gb_hours, ram_gb_hours, cost, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, query,
		record.ID, record.UserID, record.DeploymentID,
		record.GPUHours, record.VRAMGBHours, record.RAMGBHours,
		record.Cost, record.RecordedAt,
	)
	return err
}

func (r *BillingRepository) ListUsageByUserID(ctx context.Context, userID uuid.UUID) ([]*model.UsageRecord, error) {
	query := `
		SELECT id, user_id, deployment_id, gpu_hours, vram_gb_hours, ram_gb_hours, cost, recorded_at
		FROM usage_records
		WHERE user_id = $1
		ORDER BY recorded_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*model.UsageRecord
	for rows.Next() {
		var rec model.UsageRecord
		if err := rows.Scan(
			&rec.ID, &rec.UserID, &rec.DeploymentID,
			&rec.GPUHours, &rec.VRAMGBHours, &rec.RAMGBHours,
			&rec.Cost, &rec.RecordedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, &rec)
	}
	return records, rows.Err()
}

func (r *BillingRepository) ListUsageByDeployment(ctx context.Context, userID, deploymentID uuid.UUID) ([]*model.UsageRecord, error) {
	query := `
		SELECT id, user_id, deployment_id, gpu_hours, vram_gb_hours, ram_gb_hours, cost, recorded_at
		FROM usage_records
		WHERE user_id = $1 AND deployment_id = $2
		ORDER BY recorded_at DESC`

	rows, err := r.pool.Query(ctx, query, userID, deploymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*model.UsageRecord
	for rows.Next() {
		var rec model.UsageRecord
		if err := rows.Scan(
			&rec.ID, &rec.UserID, &rec.DeploymentID,
			&rec.GPUHours, &rec.VRAMGBHours, &rec.RAMGBHours,
			&rec.Cost, &rec.RecordedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, &rec)
	}
	return records, rows.Err()
}

func (r *BillingRepository) GetUnpaidTotal(ctx context.Context, userID uuid.UUID) (float64, error) {
	query := `SELECT COALESCE(SUM(amount), 0) FROM invoices WHERE user_id = $1 AND status = 'pending'`
	var total float64
	err := r.pool.QueryRow(ctx, query, userID).Scan(&total)
	return total, err
}
