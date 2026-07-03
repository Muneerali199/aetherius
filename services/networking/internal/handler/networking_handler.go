package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/services/networking/internal/model"
	"github.com/aetherius/platform/services/networking/internal/service"
)

type NetworkingHandler struct {
	svc *service.NetworkingService
}

func NewNetworkingHandler(svc *service.NetworkingService) *NetworkingHandler {
	return &NetworkingHandler{svc: svc}
}

func (h *NetworkingHandler) RegisterRoutes(r chi.Router, jwtManager *auth.JWTManager) {
	protected := r.With(auth.HTTPMiddleware(jwtManager))

	protected.Post("/v1/networking/vpn/create", h.CreateVPNSession)
	protected.Get("/v1/networking/vpn/sessions", h.ListVPNSessions)
	protected.Delete("/v1/networking/vpn/sessions/{id}", h.DeleteVPNSession)
	protected.Get("/v1/networking/config", h.GetNetworkConfig)
	protected.Post("/v1/networking/firewall/rules", h.AddFirewallRule)
	protected.Get("/v1/networking/firewall/rules", h.ListFirewallRules)
	protected.Delete("/v1/networking/firewall/rules/{id}", h.DeleteFirewallRule)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (h *NetworkingHandler) CreateVPNSession(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		NodeID string `json:"node_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	nodeID, err := uuid.Parse(req.NodeID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node_id")
		return
	}

	peer := h.svc.CreateVPNSession(claims.UserID, nodeID)
	writeJSON(w, http.StatusCreated, peer)
}

func (h *NetworkingHandler) ListVPNSessions(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	peers := h.svc.GetVPNSessions(claims.UserID)
	writeJSON(w, http.StatusOK, peers)
}

func (h *NetworkingHandler) DeleteVPNSession(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	peerID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid peer id")
		return
	}

	if err := h.svc.DeleteVPNSession(peerID, claims.UserID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "session deleted"})
}

func (h *NetworkingHandler) GetNetworkConfig(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	cfg := h.svc.GetNetworkConfig(claims.UserID)
	writeJSON(w, http.StatusOK, cfg)
}

func (h *NetworkingHandler) AddFirewallRule(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var rule model.FirewallRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	created := h.svc.AddFirewallRule(claims.UserID, &rule)
	writeJSON(w, http.StatusCreated, created)
}

func (h *NetworkingHandler) ListFirewallRules(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	rules := h.svc.ListFirewallRules(claims.UserID)
	writeJSON(w, http.StatusOK, rules)
}

func (h *NetworkingHandler) DeleteFirewallRule(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	ruleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule id")
		return
	}

	if err := h.svc.DeleteFirewallRule(ruleID, claims.UserID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "rule deleted"})
}
