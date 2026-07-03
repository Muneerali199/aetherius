package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/services/marketplace/internal/model"
	"github.com/aetherius/platform/services/marketplace/internal/service"
)

type MarketplaceHandler struct {
	svc *service.MarketplaceService
}

func NewMarketplaceHandler(svc *service.MarketplaceService) *MarketplaceHandler {
	return &MarketplaceHandler{svc: svc}
}

func (h *MarketplaceHandler) RegisterRoutes(r chi.Router, jwtManager *auth.JWTManager) {
	r.Get("/v1/listings", h.ListActiveListings)

	protected := r.With(auth.HTTPMiddleware(jwtManager))
	protected.Post("/v1/listings", h.CreateListing)
	protected.Post("/v1/listings/{id}/rent", h.RentListing)
	protected.Get("/v1/orders", h.ListOrders)
	protected.Post("/v1/orders/{id}/cancel", h.CancelOrder)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type CreateListingRequest struct {
	NodeID       string  `json:"node_id"`
	GPUModel     string  `json:"gpu_model"`
	GPUCount     int     `json:"gpu_count"`
	VRAMGB       int64   `json:"vram_gb"`
	RAMGB        int64   `json:"ram_gb"`
	DiskGB       int64   `json:"disk_gb"`
	PricePerHour float64 `json:"price_per_hour"`
	Region       string  `json:"region"`
}

func (h *MarketplaceHandler) CreateListing(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req CreateListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	nodeID, err := uuid.Parse(req.NodeID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node_id")
		return
	}

	listing, err := h.svc.CreateListing(
		r.Context(), claims.UserID, nodeID,
		req.GPUModel, req.GPUCount, req.VRAMGB, req.RAMGB,
		req.DiskGB, req.PricePerHour, req.Region,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to create listing")
		writeError(w, http.StatusInternalServerError, "failed to create listing")
		return
	}

	writeJSON(w, http.StatusCreated, listing)
}

func (h *MarketplaceHandler) ListActiveListings(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	minGPUStr := r.URL.Query().Get("min_gpu")
	maxPriceStr := r.URL.Query().Get("max_price")

	minGPU, _ := strconv.Atoi(minGPUStr)
	maxPrice, _ := strconv.ParseFloat(maxPriceStr, 64)

	listings, err := h.svc.ListActiveListings(r.Context(), region, minGPU, maxPrice)
	if err != nil {
		log.Error().Err(err).Msg("failed to list active listings")
		writeError(w, http.StatusInternalServerError, "failed to list listings")
		return
	}

	if listings == nil {
		listings = []*model.Listing{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"listings": listings,
	})
}

func (h *MarketplaceHandler) RentListing(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	listingIDStr := chi.URLParam(r, "id")
	listingID, err := uuid.Parse(listingIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid listing id")
		return
	}

	order, err := h.svc.RentListing(r.Context(), listingID, claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("failed to rent listing")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, order)
}

func (h *MarketplaceHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orders, err := h.svc.ListOrders(r.Context(), claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("failed to list orders")
		writeError(w, http.StatusInternalServerError, "failed to list orders")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"orders": orders,
	})
}

func (h *MarketplaceHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orderIDStr := chi.URLParam(r, "id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	if err := h.svc.CancelOrder(r.Context(), orderID, claims.UserID); err != nil {
		log.Error().Err(err).Msg("failed to cancel order")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "order cancelled"})
}
