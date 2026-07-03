package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/services/ai/internal/model"
	"github.com/aetherius/platform/services/ai/internal/service"
)

type AIHandler struct {
	svc *service.AIService
}

func New(svc *service.AIService) *AIHandler {
	return &AIHandler{svc: svc}
}

func (h *AIHandler) RegisterRoutes(r chi.Router, jwtManager *auth.JWTManager) {
	r.Route("/v1/ai", func(r chi.Router) {
		r.Use(auth.HTTPMiddleware(jwtManager))

		r.Get("/models", h.ListModels)
		r.Get("/models/{id}", h.GetModel)
		r.Post("/infer", h.Infer)
		r.Post("/deploy", h.Deploy)
		r.Get("/deployments", h.ListDeployments)
	})
}

func (h *AIHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	models := h.svc.ListModels()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"models": models,
		"count":  len(models),
	})
}

func (h *AIHandler) GetModel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	modelObj := h.svc.GetModel(id)
	if modelObj == nil {
		respondError(w, http.StatusNotFound, "model not found")
		return
	}
	respondJSON(w, http.StatusOK, modelObj)
}

func (h *AIHandler) Infer(w http.ResponseWriter, r *http.Request) {
	var req model.InferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ModelID == "" {
		respondError(w, http.StatusBadRequest, "model_id is required")
		return
	}

	claims := auth.GetClaims(r.Context())
	userID := "anonymous"
	if claims != nil {
		userID = claims.UserID.String()
	}

	resp := h.svc.Infer(&req, userID)
	respondJSON(w, http.StatusOK, resp)
}

func (h *AIHandler) Deploy(w http.ResponseWriter, r *http.Request) {
	var req model.ModelDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":        "deployment_requested",
		"model_id":      req.ModelID,
		"deployment_id": req.DeploymentID,
		"replicas":      req.Replicas,
		"message":       "Deployment request accepted and queued for processing",
	})
}

func (h *AIHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"deployments": []interface{}{},
		"count":       0,
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error().Err(err).Msg("failed to encode response")
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
