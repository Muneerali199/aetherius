package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/services/billing/internal/model"
	"github.com/aetherius/platform/services/billing/internal/service"
)

type BillingHandler struct {
	svc *service.BillingService
}

func NewBillingHandler(svc *service.BillingService) *BillingHandler {
	return &BillingHandler{svc: svc}
}

func (h *BillingHandler) RegisterRoutes(r chi.Router, jwtManager *auth.JWTManager) {
	protected := r.With(auth.HTTPMiddleware(jwtManager))
	protected.Get("/v1/billing/invoices", h.ListInvoices)
	protected.Get("/v1/billing/invoices/{id}", h.GetInvoice)
	protected.Get("/v1/billing/usage", h.ListUsage)
	protected.Get("/v1/billing/balance", h.GetBalance)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (h *BillingHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	invoices, err := h.svc.ListInvoices(r.Context(), claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("failed to list invoices")
		writeError(w, http.StatusInternalServerError, "failed to list invoices")
		return
	}

	if invoices == nil {
		invoices = []*model.Invoice{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"invoices": invoices,
	})
}

func (h *BillingHandler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	invoiceIDStr := chi.URLParam(r, "id")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invoice id")
		return
	}

	invoice, err := h.svc.GetInvoice(r.Context(), invoiceID, claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get invoice")
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}

	writeJSON(w, http.StatusOK, invoice)
}

func (h *BillingHandler) ListUsage(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var deploymentID *uuid.UUID
	if depStr := r.URL.Query().Get("deployment_id"); depStr != "" {
		parsed, err := uuid.Parse(depStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid deployment_id")
			return
		}
		deploymentID = &parsed
	}

	records, err := h.svc.ListUsage(r.Context(), claims.UserID, deploymentID)
	if err != nil {
		log.Error().Err(err).Msg("failed to list usage records")
		writeError(w, http.StatusInternalServerError, "failed to list usage records")
		return
	}

	if records == nil {
		records = []*model.UsageRecord{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"usage": records,
	})
}

func (h *BillingHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	balance, err := h.svc.GetBalance(r.Context(), claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get balance")
		writeError(w, http.StatusInternalServerError, "failed to get balance")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"balance": balance,
		"currency": "USD",
	})
}
