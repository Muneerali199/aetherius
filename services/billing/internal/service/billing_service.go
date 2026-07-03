package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aetherius/platform/services/billing/internal/model"
	"github.com/aetherius/platform/services/billing/internal/repository"
)

const (
	baseRate        = 0.10
	gpuRate         = 0.08
	vramRate        = 0.001
	ramRate         = 0.002
	defaultCurrency = "USD"
)

type BillingService struct {
	repo *repository.BillingRepository
}

func NewBillingService(repo *repository.BillingRepository) *BillingService {
	return &BillingService{repo: repo}
}

func (s *BillingService) CalculateCost(gpuCount, vramGB, ramGB int, hours float64) float64 {
	cost := baseRate*hours +
		float64(gpuCount)*gpuRate*hours +
		float64(vramGB)*vramRate*hours +
		float64(ramGB)*ramRate*hours
	return cost
}

func (s *BillingService) RecordUsage(ctx context.Context, userID, deploymentID uuid.UUID, gpuCount, vramGB, ramGB int, hours float64) (*model.UsageRecord, error) {
	cost := s.CalculateCost(gpuCount, vramGB, ramGB, hours)

	record := &model.UsageRecord{
		ID:           uuid.New(),
		UserID:       userID,
		DeploymentID: deploymentID,
		GPUHours:     float64(gpuCount) * hours,
		VRAMGBHours:  float64(vramGB) * hours,
		RAMGBHours:   float64(ramGB) * hours,
		Cost:         cost,
		RecordedAt:   time.Now(),
	}

	if err := s.repo.CreateUsageRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("record usage: %w", err)
	}

	return record, nil
}

func (s *BillingService) GenerateInvoice(ctx context.Context, userID, deploymentID uuid.UUID, amount float64, description string) (*model.Invoice, error) {
	invoice := &model.Invoice{
		ID:           uuid.New(),
		UserID:       userID,
		DeploymentID: &deploymentID,
		Amount:       amount,
		Currency:     defaultCurrency,
		Status:       model.InvoiceStatusPending,
		Description:  description,
		DueDate:      time.Now().Add(30 * 24 * time.Hour),
	}

	if err := s.repo.CreateInvoice(ctx, invoice); err != nil {
		return nil, fmt.Errorf("generate invoice: %w", err)
	}

	return invoice, nil
}

func (s *BillingService) ListInvoices(ctx context.Context, userID uuid.UUID) ([]*model.Invoice, error) {
	return s.repo.ListInvoicesByUserID(ctx, userID)
}

func (s *BillingService) GetInvoice(ctx context.Context, invoiceID, userID uuid.UUID) (*model.Invoice, error) {
	invoice, err := s.repo.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if invoice.UserID != userID {
		return nil, fmt.Errorf("invoice not found")
	}
	return invoice, nil
}

func (s *BillingService) ListUsage(ctx context.Context, userID uuid.UUID, deploymentID *uuid.UUID) ([]*model.UsageRecord, error) {
	if deploymentID != nil {
		return s.repo.ListUsageByDeployment(ctx, userID, *deploymentID)
	}
	return s.repo.ListUsageByUserID(ctx, userID)
}

func (s *BillingService) GetBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	return s.repo.GetUnpaidTotal(ctx, userID)
}
