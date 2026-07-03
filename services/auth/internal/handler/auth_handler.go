package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/services/auth/internal/repository"
	"github.com/aetherius/platform/services/auth/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router, jwtManager *auth.JWTManager) {
	r.Post("/v1/auth/register", h.Register)
	r.Post("/v1/auth/login", h.Login)
	r.Post("/v1/auth/refresh", h.Refresh)
	r.Post("/v1/auth/logout", h.Logout)

	protected := r.With(auth.HTTPMiddleware(jwtManager))
	protected.Get("/v1/auth/me", h.GetMe)
	protected.Post("/v1/auth/mfa/setup", h.SetupMFA)
	protected.Post("/v1/auth/mfa/verify", h.VerifyMFA)

	r.Post("/v1/auth/password/reset/initiate", h.InitiatePasswordReset)
	r.Post("/v1/auth/password/reset/complete", h.CompletePasswordReset)
	r.Post("/v1/auth/email/verify", h.VerifyEmail)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "email, password, and display_name are required")
		return
	}

	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	user, err := h.svc.Register(r.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		if errors.Is(err, repository.ErrEmailAlreadyExists) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		log.Error().Err(err).Msg("registration failed")
		writeError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"user_id": user.ID.String(),
		"email":   user.Email,
	})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	MFACode  string `json:"mfa_code,omitempty"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tokens, user, mfaRequired, err := h.svc.Login(r.Context(), req.Email, req.Password, req.MFACode)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	if mfaRequired {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"mfa_required": true,
			"user_id":      user.ID.String(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tokens":   tokens,
		"user_id":  user.ID.String(),
		"email":    user.Email,
		"name":     user.DisplayName,
	})
}

func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	user, err := h.svc.GetUser(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":    user.ID.String(),
		"email":      user.Email,
		"name":       user.DisplayName,
		"avatar_url": user.AvatarURL,
		"created_at": user.CreatedAt,
	})
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tokens, err := h.svc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

type LogoutRequest struct {
	SessionID    string `json:"session_id,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.Logout(r.Context(), req.SessionID, req.RefreshToken); err != nil {
		log.Error().Err(err).Msg("logout failed")
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

type MFASetupResponse struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

func (h *AuthHandler) SetupMFA(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	secret, qrURL, codes, err := h.svc.SetupMFA(r.Context(), claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("MFA setup failed")
		writeError(w, http.StatusInternalServerError, "MFA setup failed")
		return
	}

	writeJSON(w, http.StatusOK, MFASetupResponse{
		Secret:      secret,
		QRCodeURL:   qrURL,
		BackupCodes: codes,
	})
}

type VerifyMFARequest struct {
	Code string `json:"code"`
}

func (h *AuthHandler) VerifyMFA(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req VerifyMFARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	valid, err := h.svc.VerifyMFA(r.Context(), claims.UserID, req.Code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "verification failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"verified": valid})
}

func (h *AuthHandler) InitiatePasswordReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.InitiatePasswordReset(r.Context(), req.Email); err != nil {
		log.Error().Err(err).Msg("password reset initiation failed")
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "if the email exists, a reset link has been sent",
	})
}

func (h *AuthHandler) CompletePasswordReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.CompletePasswordReset(r.Context(), req.Token, req.NewPassword); err != nil {
		writeError(w, http.StatusBadRequest, "reset failed: invalid or expired token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password reset successfully"})
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		writeError(w, http.StatusBadRequest, "verification token required")
		return
	}

	token, err := uuid.Parse(tokenStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid token")
		return
	}

	if err := h.svc.VerifyEmail(r.Context(), token); err != nil {
		writeError(w, http.StatusBadRequest, "invalid or expired token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "email verified"})
}
