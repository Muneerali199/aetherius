package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/aetherius/platform/services/deployment/internal/model"
	"github.com/aetherius/platform/services/deployment/internal/repository"
)

var (
	ErrNotFound      = errors.New("deployment not found")
	ErrNotOwned      = errors.New("deployment does not belong to user")
	ErrImageRequired = errors.New("image is required")
)

type CreateDeploymentRequest struct {
	Image          string `json:"image"`
	GPURequired    int    `json:"gpu_required"`
	VRAMRequiredGB int64  `json:"vram_required_gb"`
	RAMRequiredGB  int64  `json:"ram_required_gb"`
	DiskRequiredGB int64  `json:"disk_required_gb"`
	Ports          string `json:"ports"`
	Env            string `json:"env"`
	Region         string `json:"region"`
}

type DeploymentService struct {
	repo *repository.DeploymentRepo
}

func NewDeploymentService(repo *repository.DeploymentRepo) *DeploymentService {
	return &DeploymentService{repo: repo}
}

func (s *DeploymentService) CreateDeployment(ctx context.Context, userID uuid.UUID, req CreateDeploymentRequest) (*model.Deployment, error) {
	if req.Image == "" {
		return nil, ErrImageRequired
	}

	if req.Ports == "" {
		req.Ports = "{}"
	}
	if req.Env == "" {
		req.Env = "{}"
	}

	d := &model.Deployment{
		UserID:         userID,
		Image:          req.Image,
		GPURequired:    req.GPURequired,
		VRAMRequiredGB: req.VRAMRequiredGB,
		RAMRequiredGB:  req.RAMRequiredGB,
		DiskRequiredGB: req.DiskRequiredGB,
		Ports:          req.Ports,
		Env:            req.Env,
		Region:         req.Region,
		Status:         model.DeployStatusPending,
	}

	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}

	return d, nil
}

func (s *DeploymentService) ListDeployments(ctx context.Context, userID uuid.UUID) ([]*model.Deployment, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *DeploymentService) GetDeployment(ctx context.Context, id, userID uuid.UUID) (*model.Deployment, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d.UserID != userID {
		return nil, ErrNotOwned
	}
	return d, nil
}

func (s *DeploymentService) StopDeployment(ctx context.Context, id, userID uuid.UUID) error {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if d.UserID != userID {
		return ErrNotOwned
	}
	return s.repo.UpdateStatus(ctx, id, model.DeployStatusStopped)
}

func (s *DeploymentService) DeleteDeployment(ctx context.Context, id, userID uuid.UUID) error {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if d.UserID != userID {
		return ErrNotOwned
	}
	return s.repo.Delete(ctx, id)
}
