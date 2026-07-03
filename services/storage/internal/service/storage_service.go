package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/services/storage/internal/model"
	"github.com/aetherius/platform/services/storage/internal/repository"
)

type StorageService struct {
	minio  *minio.Client
	repo   *repository.StorageRepository
	buckets []string
}

func NewStorageService(repo *repository.StorageRepository) (*StorageService, error) {
	endpoint := getEnv("MINIO_ENDPOINT", "localhost:9000")
	accessKey := getEnv("MINIO_ACCESS_KEY", "minioadmin")
	secretKey := getEnv("MINIO_SECRET_KEY", "minioadmin")
	useSSL, _ := strconv.ParseBool(getEnv("MINIO_USE_SSL", "false"))

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	svc := &StorageService{
		minio:   minioClient,
		repo:    repo,
		buckets: []string{"user-files", "deployment-data"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, bucket := range svc.buckets {
		if err := svc.ensureBucket(ctx, bucket); err != nil {
			return nil, fmt.Errorf("ensure bucket %s: %w", bucket, err)
		}
	}

	return svc, nil
}

func (s *StorageService) ensureBucket(ctx context.Context, name string) error {
	exists, err := s.minio.BucketExists(ctx, name)
	if err != nil {
		return err
	}
	if !exists {
		if err := s.minio.MakeBucket(ctx, name, minio.MakeBucketOptions{}); err != nil {
			return err
		}
		log.Info().Str("bucket", name).Msg("created minio bucket")
	}
	return nil
}

func (s *StorageService) UploadFile(ctx context.Context, userID uuid.UUID, bucket, filename, contentType string, data io.Reader) (*model.StorageObject, error) {
	now := time.Now()
	key := fmt.Sprintf("%s/%s/%d-%s", userID.String(), bucket, now.UnixMilli(), filename)
	objectID := uuid.New()

	info, err := s.minio.PutObject(ctx, bucket, key, data, -1, minio.PutObjectOptions{
		ContentType: contentType,
		UserMetadata: map[string]string{
			"object-id": objectID.String(),
			"user-id":   userID.String(),
			"filename":  filename,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("upload to minio: %w", err)
	}

	obj := &model.StorageObject{
		ID:          objectID,
		UserID:      userID,
		Bucket:      bucket,
		Key:         key,
		Filename:    filename,
		ContentType: contentType,
		Size:        info.Size,
		CreatedAt:   now,
	}

	if err := s.repo.Insert(ctx, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (s *StorageService) DownloadFile(ctx context.Context, objectID, userID uuid.UUID) (io.ReadCloser, string, error) {
	obj, err := s.repo.GetByIDAndUser(ctx, objectID, userID)
	if err != nil {
		return nil, "", err
	}

	reader, err := s.minio.GetObject(ctx, obj.Bucket, obj.Key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("get from minio: %w", err)
	}

	return reader, obj.ContentType, nil
}

func (s *StorageService) ListFiles(ctx context.Context, userID uuid.UUID, bucket, prefix string) ([]*model.StorageObject, error) {
	return s.repo.ListByUserID(ctx, userID, bucket, prefix)
}

func (s *StorageService) DeleteFile(ctx context.Context, objectID, userID uuid.UUID) error {
	obj, err := s.repo.GetByIDAndUser(ctx, objectID, userID)
	if err != nil {
		return err
	}

	if err := s.minio.RemoveObject(ctx, obj.Bucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove from minio: %w", err)
	}

	return s.repo.DeleteByIDAndUser(ctx, objectID, userID)
}

func (s *StorageService) GetAccessibleBuckets() []string {
	return s.buckets
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func (s *StorageService) sanitizeBucket(bucket string) bool {
	for _, b := range s.buckets {
		if strings.EqualFold(b, bucket) {
			return true
		}
	}
	return false
}
