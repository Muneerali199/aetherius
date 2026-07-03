package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aetherius/platform/services/marketplace/internal/model"
	"github.com/aetherius/platform/services/marketplace/internal/repository"
)

type MarketplaceService struct {
	repo *repository.MarketplaceRepository
}

func NewMarketplaceService(repo *repository.MarketplaceRepository) *MarketplaceService {
	return &MarketplaceService{repo: repo}
}

func (s *MarketplaceService) CreateListing(
	ctx context.Context,
	ownerID, nodeID uuid.UUID,
	gpuModel string,
	gpuCount int,
	vramGB, ramGB, diskGB int64,
	pricePerHour float64,
	region string,
) (*model.Listing, error) {
	listing := &model.Listing{
		ID:           uuid.New(),
		NodeID:       nodeID,
		OwnerID:      ownerID,
		GPUModel:     gpuModel,
		GPUCount:     gpuCount,
		VRAMGB:       vramGB,
		RAMGB:        ramGB,
		DiskGB:       diskGB,
		PricePerHour: pricePerHour,
		Region:       region,
		Status:       model.ListingStatusActive,
	}

	if err := s.repo.CreateListing(ctx, listing); err != nil {
		return nil, fmt.Errorf("create listing: %w", err)
	}

	return listing, nil
}

func (s *MarketplaceService) ListActiveListings(
	ctx context.Context,
	region string,
	minGPU int,
	maxPrice float64,
) ([]*model.Listing, error) {
	return s.repo.ListActiveListings(ctx, region, minGPU, maxPrice)
}

func (s *MarketplaceService) RentListing(ctx context.Context, listingID, renterID uuid.UUID) (*model.Order, error) {
	listing, err := s.repo.GetListingByID(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("get listing: %w", err)
	}

	if listing.OwnerID == renterID {
		return nil, errors.New("cannot rent your own listing")
	}

	if listing.Status != model.ListingStatusActive {
		return nil, fmt.Errorf("listing is not available (current status: %s)", listing.Status)
	}

	now := time.Now()
	order := &model.Order{
		ID:        uuid.New(),
		ListingID: listingID,
		RenterID:  renterID,
		Status:    model.OrderStatusPending,
		StartedAt: &now,
	}

	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	if err := s.repo.UpdateListingStatus(ctx, listingID, model.ListingStatusRented); err != nil {
		return nil, fmt.Errorf("update listing status: %w", err)
	}

	order.Status = model.OrderStatusActive

	return order, nil
}

func (s *MarketplaceService) ListOrders(ctx context.Context, renterID uuid.UUID) ([]*model.Order, error) {
	return s.repo.ListOrdersByRenter(ctx, renterID)
}

func (s *MarketplaceService) CancelOrder(ctx context.Context, orderID, userID uuid.UUID) error {
	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	if order.RenterID != userID {
		return errors.New("you can only cancel your own orders")
	}

	if order.Status != model.OrderStatusPending && order.Status != model.OrderStatusActive {
		return fmt.Errorf("cannot cancel order in %s status", order.Status)
	}

	now := time.Now()
	if err := s.repo.UpdateOrderStatus(ctx, orderID, model.OrderStatusCancelled, &now); err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	// Revert listing status back to active when order is cancelled
	if err := s.repo.UpdateListingStatus(ctx, order.ListingID, model.ListingStatusActive); err != nil {
		return fmt.Errorf("update listing status: %w", err)
	}

	return nil
}
