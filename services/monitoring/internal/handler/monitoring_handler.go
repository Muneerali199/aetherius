package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/services/monitoring/internal/model"
	"github.com/aetherius/platform/services/monitoring/internal/service"
)

type MonitoringHandler struct {
	svc *service.MonitoringService
}

func NewMonitoringHandler(svc *service.MonitoringService) *MonitoringHandler {
	return &MonitoringHandler{svc: svc}
}

func (h *MonitoringHandler) RegisterRoutes(r chi.Router, jwtManager *auth.JWTManager) {
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	protected := r.With(auth.HTTPMiddleware(jwtManager))

	protected.Get("/v1/monitoring/health", h.GetSystemHealth)
	protected.Get("/v1/monitoring/metrics", h.GetMetrics)
	protected.Get("/v1/monitoring/alerts", h.GetAlerts)
	protected.Post("/v1/monitoring/alerts/{id}/acknowledge", h.AcknowledgeAlert)
	protected.Post("/v1/monitoring/metrics/record", h.RecordMetrics)
}

func (h *MonitoringHandler) GetSystemHealth(w http.ResponseWriter, r *http.Request) {
	health := h.svc.CheckServices()
	writeJSON(w, http.StatusOK, health)
}

func (h *MonitoringHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := h.svc.GetMetrics()
	if metrics == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"message": "no metrics recorded yet"})
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (h *MonitoringHandler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := h.svc.GetAlerts()
	writeJSON(w, http.StatusOK, alerts)
}

func (h *MonitoringHandler) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.AcknowledgeAlert(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "alert acknowledged"})
}

func (h *MonitoringHandler) RecordMetrics(w http.ResponseWriter, r *http.Request) {
	var snapshot model.MetricsSnapshot
	if err := json.NewDecoder(r.Body).Decode(&snapshot); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	h.svc.RecordMetrics(&snapshot)
	log.Info().Msg("metrics recorded manually")
	writeJSON(w, http.StatusCreated, map[string]string{"message": "metrics recorded"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error().Err(err).Msg("failed to write JSON response")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
