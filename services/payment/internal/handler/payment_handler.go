package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/services/payment/internal/model"
	"github.com/aetherius/platform/services/payment/internal/service"
	"github.com/aetherius/platform/services/payment/internal/repository"
)

type PaymentHandler struct {
	svc *service.PaymentService
}

func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

func (h *PaymentHandler) RegisterRoutes(r chi.Router, jwtManager *auth.JWTManager) {
	webhook := r
	webhook.Post("/v1/payments/webhook", h.Webhook)

	protected := r.With(auth.HTTPMiddleware(jwtManager))
	protected.Get("/v1/payments/wallet", h.GetWallet)
	protected.Post("/v1/payments/create-intent", h.CreatePaymentIntent)
	protected.Get("/v1/payments/transactions", h.ListTransactions)
	protected.Post("/v1/payments/methods", h.AddPaymentMethod)
	protected.Get("/v1/payments/methods", h.ListPaymentMethods)
	protected.Delete("/v1/payments/methods/{id}", h.DeletePaymentMethod)
	protected.Get("/v1/payments/balance", h.GetBalance)
	protected.Post("/v1/payments/invoices/{id}/pay", h.PayInvoice)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (h *PaymentHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	wallet, err := h.svc.GetWallet(r.Context(), claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("get wallet failed")
		writeError(w, http.StatusInternalServerError, "failed to get wallet")
		return
	}
	writeJSON(w, http.StatusOK, wallet)
}

func (h *PaymentHandler) CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req model.CreatePaymentIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Amount < 50 {
		writeError(w, http.StatusBadRequest, "minimum amount is $0.50")
		return
	}
	resp, err := h.svc.CreatePaymentIntent(r.Context(), claims.UserID, &req)
	if err != nil {
		log.Error().Err(err).Msg("create payment intent failed")
		writeError(w, http.StatusInternalServerError, "failed to create payment intent")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *PaymentHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	sigHeader := r.Header.Get("Stripe-Signature")
	if err := h.svc.HandleWebhook(r.Context(), payload, sigHeader); err != nil {
		log.Error().Err(err).Msg("webhook handling failed")
		writeError(w, http.StatusBadRequest, "webhook error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *PaymentHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	txns, err := h.svc.ListTransactions(r.Context(), claims.UserID, limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("list transactions failed")
		writeError(w, http.StatusInternalServerError, "failed to list transactions")
		return
	}
	if txns == nil {
		txns = []*model.Transaction{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"transactions": txns})
}

type AddPaymentMethodRequest struct {
	StripePMID string `json:"stripe_pm_id"`
	Last4      string `json:"last4"`
	Brand      string `json:"brand"`
	ExpMonth   int    `json:"exp_month"`
	ExpYear    int    `json:"exp_year"`
}

func (h *PaymentHandler) AddPaymentMethod(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req AddPaymentMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	pm, err := h.svc.AddPaymentMethod(r.Context(), claims.UserID, req.StripePMID, req.Last4, req.Brand, req.ExpMonth, req.ExpYear)
	if err != nil {
		log.Error().Err(err).Msg("add payment method failed")
		writeError(w, http.StatusInternalServerError, "failed to add payment method")
		return
	}
	writeJSON(w, http.StatusCreated, pm)
}

func (h *PaymentHandler) ListPaymentMethods(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	pms, err := h.svc.ListPaymentMethods(r.Context(), claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("list payment methods failed")
		writeError(w, http.StatusInternalServerError, "failed to list payment methods")
		return
	}
	if pms == nil {
		pms = []*model.PaymentMethod{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"payment_methods": pms})
}

func (h *PaymentHandler) DeletePaymentMethod(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.DeletePaymentMethod(r.Context(), id, claims.UserID); err != nil {
		writeError(w, http.StatusNotFound, "payment method not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *PaymentHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	balance, err := h.svc.GetBalance(r.Context(), claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("get balance failed")
		writeError(w, http.StatusInternalServerError, "failed to get balance")
		return
	}
	unpaid, _ := h.svc.GetUnpaidInvoiceTotal(r.Context(), claims.UserID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"balance":         balance,
		"unpaid_invoices": unpaid,
		"available":       balance - unpaid,
		"currency":        "USD",
	})
}

func (h *PaymentHandler) PayInvoice(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invoice id")
		return
	}
	if err := h.svc.PayInvoice(r.Context(), claims.UserID, id); err != nil {
		if err == repository.ErrInsufficientBalance {
			writeError(w, http.StatusBadRequest, "insufficient balance")
			return
		}
		log.Error().Err(err).Msg("pay invoice failed")
		writeError(w, http.StatusInternalServerError, "failed to pay invoice")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "invoice paid"})
}
