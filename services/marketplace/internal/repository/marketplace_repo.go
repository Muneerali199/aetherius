package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aetherius/platform/services/marketplace/internal/model"
)

var (
	ErrListingNotFound = errors.New("listing not found")
	ErrOrderNotFound   = errors.New("order not found")
)

type MarketplaceRepository struct {
	pool *pgxpool.Pool
}

func NewMarketplaceRepository(pool *pgxpool.Pool) *MarketplaceRepository {
	return &MarketplaceRepository{pool: pool}
}

func (r *MarketplaceRepository) CreateListing(ctx context.Context, listing *model.Listing) error {
	query := `
		INSERT INTO listings (id, node_id, owner_id, gpu_model, gpu_count, vram_gb, ram_gb, disk_gb, price_per_hour, region, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		listing.ID, listing.NodeID, listing.OwnerID, listing.GPUModel,
		listing.GPUCount, listing.VRAMGB, listing.RAMGB, listing.DiskGB,
		listing.PricePerHour, listing.Region, listing.Status,
	).Scan(&listing.CreatedAt, &listing.UpdatedAt)
	return err
}

func (r *MarketplaceRepository) ListActiveListings(ctx context.Context, region string, minGPU int, maxPrice float64) ([]*model.Listing, error) {
	conditions := []string{"status = 'active'"}
	args := []interface{}{}
	argIdx := 1

	if region != "" {
		conditions = append(conditions, fmt.Sprintf("region = $%d", argIdx))
		args = append(args, region)
		argIdx++
	}
	if minGPU > 0 {
		conditions = append(conditions, fmt.Sprintf("gpu_count >= $%d", argIdx))
		args = append(args, minGPU)
		argIdx++
	}
	if maxPrice > 0 {
		conditions = append(conditions, fmt.Sprintf("price_per_hour <= $%d", argIdx))
		args = append(args, maxPrice)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT id, node_id, owner_id, gpu_model, gpu_count, vram_gb, ram_gb, disk_gb,
		       price_per_hour, region, status, created_at, updated_at
		FROM listings
		WHERE %s
		ORDER BY created_at DESC`, strings.Join(conditions, " AND "))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var listings []*model.Listing
	for rows.Next() {
		var l model.Listing
		if err := rows.Scan(
			&l.ID, &l.NodeID, &l.OwnerID, &l.GPUModel, &l.GPUCount,
			&l.VRAMGB, &l.RAMGB, &l.DiskGB, &l.PricePerHour,
			&l.Region, &l.Status, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, err
		}
		listings = append(listings, &l)
	}
	return listings, rows.Err()
}

func (r *MarketplaceRepository) GetListingByID(ctx context.Context, id uuid.UUID) (*model.Listing, error) {
	query := `
		SELECT id, node_id, owner_id, gpu_model, gpu_count, vram_gb, ram_gb, disk_gb,
		       price_per_hour, region, status, created_at, updated_at
		FROM listings WHERE id = $1`

	var l model.Listing
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&l.ID, &l.NodeID, &l.OwnerID, &l.GPUModel, &l.GPUCount,
		&l.VRAMGB, &l.RAMGB, &l.DiskGB, &l.PricePerHour,
		&l.Region, &l.Status, &l.CreatedAt, &l.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrListingNotFound
	}
	return &l, err
}

func (r *MarketplaceRepository) UpdateListingStatus(ctx context.Context, id uuid.UUID, status model.ListingStatus) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE listings SET status = $1, updated_at = $2 WHERE id = $3`,
		status, time.Now(), id,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrListingNotFound
	}
	return nil
}

func (r *MarketplaceRepository) CreateOrder(ctx context.Context, order *model.Order) error {
	query := `
		INSERT INTO orders (id, listing_id, renter_id, status, started_at, ended_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at`

	err := r.pool.QueryRow(ctx, query,
		order.ID, order.ListingID, order.RenterID, order.Status,
		order.StartedAt, order.EndedAt,
	).Scan(&order.CreatedAt)
	return err
}

func (r *MarketplaceRepository) ListOrdersByRenter(ctx context.Context, renterID uuid.UUID) ([]*model.Order, error) {
	query := `
		SELECT id, listing_id, renter_id, status, started_at, ended_at, created_at
		FROM orders WHERE renter_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, renterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(
			&o.ID, &o.ListingID, &o.RenterID, &o.Status,
			&o.StartedAt, &o.EndedAt, &o.CreatedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}
	return orders, rows.Err()
}

func (r *MarketplaceRepository) GetOrderByID(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	query := `
		SELECT id, listing_id, renter_id, status, started_at, ended_at, created_at
		FROM orders WHERE id = $1`

	var o model.Order
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&o.ID, &o.ListingID, &o.RenterID, &o.Status,
		&o.StartedAt, &o.EndedAt, &o.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	return &o, err
}

func (r *MarketplaceRepository) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status model.OrderStatus, endedAt *time.Time) error {
	query := `UPDATE orders SET status = $1`
	args := []interface{}{status}
	argIdx := 2

	if endedAt != nil {
		query += fmt.Sprintf(", ended_at = $%d", argIdx)
		args = append(args, *endedAt)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}
