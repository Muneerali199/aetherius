package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/services/support/internal/model"
	"github.com/aetherius/platform/services/support/internal/service"
)

type SupportHandler struct {
	svc *service.SupportService
}

func NewSupportHandler(svc *service.SupportService) *SupportHandler {
	return &SupportHandler{svc: svc}
}

func (h *SupportHandler) RegisterRoutes(r chi.Router, jwtManager *auth.JWTManager) {
	protected := r.With(auth.HTTPMiddleware(jwtManager))
	protected.Post("/v1/support/tickets", h.CreateTicket)
	protected.Get("/v1/support/tickets", h.ListTickets)
	protected.Get("/v1/support/tickets/{id}", h.GetTicket)
	protected.Post("/v1/support/tickets/{id}/messages", h.AddMessage)
	protected.Get("/v1/support/tickets/{id}/messages", h.GetMessages)
	protected.Put("/v1/support/tickets/{id}/status", h.UpdateStatus)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type CreateTicketRequest struct {
	Subject  string              `json:"subject"`
	Category string              `json:"category"`
	Priority model.TicketPriority `json:"priority"`
	Content  string              `json:"content"`
}

func (h *SupportHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ticket, err := h.svc.CreateTicket(r.Context(), claims.UserID, req.Subject, req.Category, req.Priority, req.Content)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Error().Err(err).Msg("create ticket failed")
		writeError(w, http.StatusInternalServerError, "failed to create ticket")
		return
	}

	writeJSON(w, http.StatusCreated, ticket)
}

func (h *SupportHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	ticketIDStr := chi.URLParam(r, "id")
	ticketID, err := uuid.Parse(ticketIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	isStaff := claims.Role == "admin" || claims.Role == "staff"
	ticket, err := h.svc.GetTicket(r.Context(), ticketID, claims.UserID, isStaff)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}

	writeJSON(w, http.StatusOK, ticket)
}

func (h *SupportHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	isStaff := claims.Role == "admin" || claims.Role == "staff"
	tickets, err := h.svc.ListTickets(r.Context(), claims.UserID, isStaff)
	if err != nil {
		log.Error().Err(err).Msg("list tickets failed")
		writeError(w, http.StatusInternalServerError, "failed to list tickets")
		return
	}

	writeJSON(w, http.StatusOK, tickets)
}

type AddMessageRequest struct {
	Content string `json:"content"`
}

func (h *SupportHandler) AddMessage(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	ticketIDStr := chi.URLParam(r, "id")
	ticketID, err := uuid.Parse(ticketIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var req AddMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	isStaff := claims.Role == "admin" || claims.Role == "staff"
	msg, err := h.svc.AddMessage(r.Context(), ticketID, claims.UserID, req.Content, isStaff)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}

	writeJSON(w, http.StatusCreated, msg)
}

func (h *SupportHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	ticketIDStr := chi.URLParam(r, "id")
	ticketID, err := uuid.Parse(ticketIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	messages, err := h.svc.GetMessages(r.Context(), ticketID, claims.UserID)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}

	writeJSON(w, http.StatusOK, messages)
}

type UpdateStatusRequest struct {
	Status model.TicketStatus `json:"status"`
}

func (h *SupportHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	ticketIDStr := chi.URLParam(r, "id")
	ticketID, err := uuid.Parse(ticketIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	isStaff := claims.Role == "admin" || claims.Role == "staff"
	if err := h.svc.UpdateStatus(r.Context(), ticketID, claims.UserID, req.Status, isStaff); err != nil {
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "status updated"})
}
