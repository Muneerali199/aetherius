package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/aetherius/platform/services/notification/internal/model"
)

type NotificationHandler struct{}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

func (h *NotificationHandler) RegisterRoutes(r chi.Router) {
	r.Post("/v1/notifications/send", h.SendNotification)
}

type sendRequest struct {
	Type    model.NotificationType `json:"type"`
	UserID  string                 `json:"user_id"`
	Title   string                 `json:"title"`
	Message string                 `json:"message"`
}

func (h *NotificationHandler) SendNotification(w http.ResponseWriter, r *http.Request) {
	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	n := model.Notification{
		Type:    req.Type,
		UserID:  req.UserID,
		Title:   req.Title,
		Message: req.Message,
	}

	fmt.Printf("[%s] AD-HOC NOTIFICATION %s: %s\n", time.Now().Format(time.RFC3339), n.Type, n.Message)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}
