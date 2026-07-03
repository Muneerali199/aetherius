package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/services/storage/internal/service"
)

type StorageHandler struct {
	svc *service.StorageService
}

func NewStorageHandler(svc *service.StorageService) *StorageHandler {
	return &StorageHandler{svc: svc}
}

func (h *StorageHandler) RegisterRoutes(r chi.Router, jwtManager *auth.JWTManager) {
	r.Route("/v1/storage", func(r chi.Router) {
		r.Use(auth.HTTPMiddleware(jwtManager))

		r.Post("/upload", h.UploadFile)
		r.Get("/download/{id}", h.DownloadFile)
		r.Get("/files", h.ListFiles)
		r.Delete("/files/{id}", h.DeleteFile)
		r.Get("/buckets", h.ListBuckets)
	})
}

type uploadResponse struct {
	ID       uuid.UUID `json:"id"`
	Bucket   string    `json:"bucket"`
	Key      string    `json:"key"`
	Filename string    `json:"filename"`
	Size     int64     `json:"size"`
}

func (h *StorageHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	bucket := r.FormValue("bucket")
	if bucket == "" {
		http.Error(w, "bucket is required", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	obj, err := h.svc.UploadFile(r.Context(), claims.UserID, bucket, header.Filename, contentType, file)
	if err != nil {
		log.Error().Err(err).Msg("upload file failed")
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(uploadResponse{
		ID:       obj.ID,
		Bucket:   obj.Bucket,
		Key:      obj.Key,
		Filename: obj.Filename,
		Size:     obj.Size,
	})
}

func (h *StorageHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	objectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	reader, contentType, err := h.svc.DownloadFile(r.Context(), objectID, claims.UserID)
	if err != nil {
		log.Error().Err(err).Str("object_id", objectID.String()).Msg("download file failed")
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment")
	io.Copy(w, reader)
}

func (h *StorageHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	bucket := r.URL.Query().Get("bucket")
	prefix := r.URL.Query().Get("prefix")

	objects, err := h.svc.ListFiles(r.Context(), claims.UserID, bucket, prefix)
	if err != nil {
		log.Error().Err(err).Msg("list files failed")
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(objects)
}

func (h *StorageHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	objectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.svc.DeleteFile(r.Context(), objectID, claims.UserID); err != nil {
		log.Error().Err(err).Str("object_id", objectID.String()).Msg("delete file failed")
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *StorageHandler) ListBuckets(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	buckets := h.svc.GetAccessibleBuckets()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"buckets": buckets})
}
