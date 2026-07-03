package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/services/scheduler/internal/model"
	"github.com/aetherius/platform/services/scheduler/internal/repository"
	"github.com/aetherius/platform/services/scheduler/internal/scheduler"
)

type SchedulerHandler struct {
	repo  *repository.SchedulerRepository
	sched *scheduler.Scheduler
}

func NewSchedulerHandler(repo *repository.SchedulerRepository, sched *scheduler.Scheduler) *SchedulerHandler {
	return &SchedulerHandler{repo: repo, sched: sched}
}

func (h *SchedulerHandler) RegisterRoutes(r chi.Router, jwtManager *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.HTTPMiddleware(jwtManager))
		r.Get("/v1/deployments", h.ListDeployments)
		r.Get("/v1/deployments/{id}", h.GetDeployment)
		r.Post("/v1/deployments", h.CreateDeployment)
		r.Post("/v1/deployments/{id}/stop", h.StopDeployment)
	})
}

func (h *SchedulerHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	deployments, err := h.repo.ListDeploymentsByUserID(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if deployments == nil {
		deployments = []*model.Deployment{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deployments)
}

func (h *SchedulerHandler) GetDeployment(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid deployment id"}`, http.StatusBadRequest)
		return
	}

	d, err := h.repo.GetDeploymentByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"deployment not found"}`, http.StatusNotFound)
		return
	}

	if d.UserID != claims.UserID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d)
}

type createDeploymentRequest struct {
	Image          string            `json:"image"`
	GPURequired    int               `json:"gpu_required"`
	VRAMRequiredGB int64             `json:"vram_required_gb"`
	RAMRequiredGB  int64             `json:"ram_required_gb"`
	DiskRequiredGB int64             `json:"disk_required_gb"`
	Ports          map[string]int    `json:"ports,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	Region         string            `json:"region,omitempty"`
}

func (h *SchedulerHandler) CreateDeployment(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req createDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	now := time.Now()
	d := &model.Deployment{
		ID:             uuid.New(),
		UserID:         claims.UserID,
		Image:          req.Image,
		GPURequired:    req.GPURequired,
		VRAMRequiredGB: req.VRAMRequiredGB,
		RAMRequiredGB:  req.RAMRequiredGB,
		DiskRequiredGB: req.DiskRequiredGB,
		Ports:          req.Ports,
		Env:            req.Env,
		Status:         model.DeployStatusPending,
		Region:         req.Region,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := h.repo.CreateDeployment(r.Context(), d); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	h.sched.Enqueue(&scheduler.DeploymentRequest{
		ID:     d.ID,
		UserID: d.UserID,
		Resources: scheduler.ResourceRequest{
			GPUCount:    d.GPURequired,
			VRAMBytes:   d.VRAMRequiredGB * 1024 * 1024 * 1024,
			RAMBytes:    d.RAMRequiredGB * 1024 * 1024 * 1024,
			DiskBytes:   d.DiskRequiredGB * 1024 * 1024 * 1024,
		},
		Placement: scheduler.PlacementPreference{
			Region: d.Region,
		},
		CreatedAt: d.CreatedAt,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(d)
}

func (h *SchedulerHandler) StopDeployment(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid deployment id"}`, http.StatusBadRequest)
		return
	}

	d, err := h.repo.GetDeploymentByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"deployment not found"}`, http.StatusNotFound)
		return
	}

	if d.UserID != claims.UserID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	if err := h.repo.UpdateStatus(r.Context(), id, model.DeployStatusStopping); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopping"})
}
