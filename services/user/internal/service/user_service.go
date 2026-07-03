package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"

	"github.com/aetherius/platform/services/user/internal/model"
	"github.com/aetherius/platform/services/user/internal/repository"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.UserProfile, error) {
	return s.repo.GetOrCreateProfile(ctx, userID)
}

func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, bio, website, githubUser, timezone string, notifyEmail bool) (*model.UserProfile, error) {
	return s.repo.UpdateProfile(ctx, userID, bio, website, githubUser, timezone, notifyEmail)
}

func (s *UserService) CreateApiKey(ctx context.Context, userID uuid.UUID, name string) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}

	plaintext := "ak_" + hex.EncodeToString(raw)
	hash := sha256Hex(plaintext)
	prefix := plaintext[:10]

	_, err := s.repo.CreateApiKey(ctx, userID, name, prefix, hash)
	if err != nil {
		return "", err
	}

	return plaintext, nil
}

func (s *UserService) ListApiKeys(ctx context.Context, userID uuid.UUID) ([]*model.ApiKey, error) {
	keys, err := s.repo.ListApiKeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		k.KeyHash = ""
	}
	return keys, nil
}

func (s *UserService) DeleteApiKey(ctx context.Context, keyID, userID uuid.UUID) error {
	return s.repo.DeleteApiKey(ctx, keyID, userID)
}

func sha256Hex(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
