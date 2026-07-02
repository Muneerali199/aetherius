package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/services/node/internal/service"
)

type NodeHandler struct {
	svc *service.NodeService
}

func NewNodeHandler(svc *service.NodeService) *NodeHandler {
	return &NodeHandler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type RegisterNodeRequest struct {
	ProviderID   string                 `json:"provider_id"`
	AgentVersion string                 `json:"agent_version"`
	PublicIP     string                 `json:"public_ip"`
	Hardware     service.HardwareInfo   `json:"hardware"`
}

func (h *NodeHandler) RegisterNode(w http.ResponseWriter, r *http.Request) {
	var req RegisterNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	providerID, err := uuid.Parse(req.ProviderID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid provider_id")
		return
	}

	node, token, err := h.svc.RegisterNode(r.Context(), providerID, req.Hardware, req.AgentVersion, req.PublicIP)
	if err != nil {
		log.Error().Err(err).Msg("node registration failed")
		writeError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"node_id":    node.ID.String(),
		"node_token": token,
		"status":     node.Status,
	})
}

type HeartbeatRequest struct {
	NodeID            string    `json:"node_id"`
	NodeToken         string    `json:"node_token"`
	GPUUtil           []float64 `json:"gpu_util"`
	GPUTemps          []float64 `json:"gpu_temps"`
	VRAMUsed          []int64   `json:"vram_used"`
	CPUUtil           float64   `json:"cpu_util"`
	RAMUsedGB         int64     `json:"ram_used_gb"`
	DiskUsedGB        int64     `json:"disk_used_gb"`
	NetworkRXBytes    int64     `json:"network_rx_bytes"`
	NetworkTXBytes    int64     `json:"network_tx_bytes"`
	LoadAvg           float64   `json:"load_avg"`
	UptimeSeconds     int64     `json:"uptime_seconds"`
	RunningContainers []string  `json:"running_containers"`
}

func (h *NodeHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid heartbeat")
		return
	}

	nodeID, err := uuid.Parse(req.NodeID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node_id")
		return
	}

	hb := service.HeartbeatData{
		GPUUtil:           req.GPUUtil,
		GPUTemps:          req.GPUTemps,
		VRAMUsed:          req.VRAMUsed,
		CPUUtil:           req.CPUUtil,
		RAMUsedGB:         req.RAMUsedGB,
		DiskUsedGB:        req.DiskUsedGB,
		NetworkRXBytes:    req.NetworkRXBytes,
		NetworkTXBytes:    req.NetworkTXBytes,
		LoadAvg:           req.LoadAvg,
		UptimeSeconds:     req.UptimeSeconds,
		RunningContainers: req.RunningContainers,
	}

	if err := h.svc.ProcessHeartbeat(r.Context(), nodeID, hb); err != nil {
		log.Error().Err(err).Msg("heartbeat processing failed")
		writeError(w, http.StatusInternalServerError, "heartbeat failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"accepted":                  true,
		"next_heartbeat_interval_ms": 5000,
	})
}

func (h *NodeHandler) GetNode(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node_id")
		return
	}

	_ = nodeID
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *NodeHandler) ListNodes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *NodeHandler) PauseNode(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *NodeHandler) ResumeNode(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
