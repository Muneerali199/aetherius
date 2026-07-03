package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/services/deployment/internal/model"
	"github.com/aetherius/platform/services/deployment/internal/service"
)

type DeploymentHandler struct {
	svc *service.DeploymentService
}

func NewDeploymentHandler(svc *service.DeploymentService) *DeploymentHandler {
	return &DeploymentHandler{svc: svc}
}

func (h *DeploymentHandler) RegisterRoutes(r chi.Router) {
	r.Get("/v1/deployments", h.ListDeployments)
	r.Get("/v1/deployments/{id}", h.GetDeployment)
	r.Post("/v1/deployments", h.CreateDeployment)
	r.Post("/v1/deployments/{id}/stop", h.StopDeployment)
	r.Delete("/v1/deployments/{id}", h.DeleteDeployment)
}

func getUserID(r *http.Request) uuid.UUID {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		return uuid.Nil
	}
	return claims.UserID
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (h *DeploymentHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "invalid user")
		return
	}

	deployments, err := h.svc.ListDeployments(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list deployments")
		return
	}

	if deployments == nil {
		deployments = []*model.Deployment{}
	}

	writeJSON(w, http.StatusOK, deployments)
}

func (h *DeploymentHandler) GetDeployment(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "invalid user")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid deployment id")
		return
	}

	d, err := h.svc.GetDeployment(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) || errors.Is(err, service.ErrNotOwned) {
			writeError(w, http.StatusNotFound, "deployment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get deployment")
		return
	}

	writeJSON(w, http.StatusOK, d)
}

func (h *DeploymentHandler) CreateDeployment(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "invalid user")
		return
	}

	var req service.CreateDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	d, err := h.svc.CreateDeployment(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrImageRequired) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create deployment")
		return
	}

	writeJSON(w, http.StatusCreated, d)
}

func (h *DeploymentHandler) StopDeployment(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "invalid user")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid deployment id")
		return
	}

	if err := h.svc.StopDeployment(r.Context(), id, userID); err != nil {
		if errors.Is(err, service.ErrNotFound) || errors.Is(err, service.ErrNotOwned) {
			writeError(w, http.StatusNotFound, "deployment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to stop deployment")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (h *DeploymentHandler) DeleteDeployment(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "invalid user")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid deployment id")
		return
	}

	if err := h.svc.DeleteDeployment(r.Context(), id, userID); err != nil {
		if errors.Is(err, service.ErrNotFound) || errors.Is(err, service.ErrNotOwned) {
			writeError(w, http.StatusNotFound, "deployment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete deployment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
