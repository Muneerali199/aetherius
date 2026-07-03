package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/services/user/internal/service"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) RegisterRoutes(r chi.Router, jwtManager *auth.JWTManager) {
	protected := r.With(auth.HTTPMiddleware(jwtManager))
	protected.Get("/v1/user/profile", h.GetProfile)
	protected.Put("/v1/user/profile", h.UpdateProfile)
	protected.Post("/v1/user/keys", h.CreateApiKey)
	protected.Get("/v1/user/keys", h.ListApiKeys)
	protected.Delete("/v1/user/keys/{id}", h.DeleteApiKey)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	profile, err := h.svc.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		log.Error().Err(err).Str("user_id", claims.UserID.String()).Msg("get profile failed")
		writeError(w, http.StatusInternalServerError, "failed to get profile")
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

type UpdateProfileRequest struct {
	Bio         string `json:"bio"`
	Website     string `json:"website"`
	GithubUser  string `json:"github_user"`
	Timezone    string `json:"timezone"`
	NotifyEmail *bool  `json:"notify_email"`
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	notifyEmail := false
	if req.NotifyEmail != nil {
		notifyEmail = *req.NotifyEmail
	}

	profile, err := h.svc.UpdateProfile(r.Context(), claims.UserID, req.Bio, req.Website, req.GithubUser, req.Timezone, notifyEmail)
	if err != nil {
		log.Error().Err(err).Str("user_id", claims.UserID.String()).Msg("update profile failed")
		writeError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

type CreateApiKeyRequest struct {
	Name string `json:"name"`
}

type CreateApiKeyResponse struct {
	ApiKey string `json:"api_key"`
}

func (h *UserHandler) CreateApiKey(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req CreateApiKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	plaintext, err := h.svc.CreateApiKey(r.Context(), claims.UserID, req.Name)
	if err != nil {
		log.Error().Err(err).Str("user_id", claims.UserID.String()).Msg("create api key failed")
		writeError(w, http.StatusInternalServerError, "failed to create api key")
		return
	}

	writeJSON(w, http.StatusCreated, CreateApiKeyResponse{ApiKey: plaintext})
}

func (h *UserHandler) ListApiKeys(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	keys, err := h.svc.ListApiKeys(r.Context(), claims.UserID)
	if err != nil {
		log.Error().Err(err).Str("user_id", claims.UserID.String()).Msg("list api keys failed")
		writeError(w, http.StatusInternalServerError, "failed to list api keys")
		return
	}

	writeJSON(w, http.StatusOK, keys)
}

func (h *UserHandler) DeleteApiKey(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	keyIDStr := chi.URLParam(r, "id")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid key id")
		return
	}

	if err := h.svc.DeleteApiKey(r.Context(), keyID, claims.UserID); err != nil {
		log.Error().Err(err).Str("key_id", keyIDStr).Msg("delete api key failed")
		writeError(w, http.StatusNotFound, "key not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "key deleted"})
}
