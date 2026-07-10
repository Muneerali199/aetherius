package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/services/node/internal/model"
	"github.com/aetherius/platform/services/node/internal/service"
)

type NodeHandler struct {
	svc  *service.NodeService
	jwtm *auth.JWTManager
}

func NewNodeHandler(svc *service.NodeService, jwtm *auth.JWTManager) *NodeHandler {
	return &NodeHandler{svc: svc, jwtm: jwtm}
}

func (h *NodeHandler) validateToken(tokenStr string) *auth.Claims {
	if h.jwtm == nil {
		return nil
	}
	claims, err := h.jwtm.ValidateAccess(tokenStr)
	if err != nil {
		return nil
	}
	return claims
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
	ProviderID   string                 `json:"provider_id,omitempty"`
	AgentVersion string                 `json:"agent_version"`
	AgentURL     string                 `json:"agent_url"`
	PublicIP     string                 `json:"public_ip"`
	Hardware     service.HardwareInfo   `json:"hardware"`
}

func (h *NodeHandler) RegisterNode(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req RegisterNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	node, token, err := h.svc.RegisterNode(r.Context(), claims.UserID, req.Hardware, req.AgentVersion, req.PublicIP, req.AgentURL)
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
	AgentURL          string    `json:"agent_url"`
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
		AgentURL:          req.AgentURL,
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

func (h *NodeHandler) SimpleHeartbeat(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node_id")
		return
	}

	if err := h.svc.ProcessHeartbeat(r.Context(), nodeID, service.HeartbeatData{}); err != nil {
		log.Error().Err(err).Msg("heartbeat processing failed")
		writeError(w, http.StatusInternalServerError, "heartbeat failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *NodeHandler) GetNode(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node_id")
		return
	}

	node, err := h.svc.GetNode(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}

	writeJSON(w, http.StatusOK, node)
}

func (h *NodeHandler) ListAvailableNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.svc.ListAvailableNodesPublic(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to list available nodes")
		writeError(w, http.StatusInternalServerError, "failed to list available nodes")
		return
	}

	if nodes == nil {
		nodes = []*service.AvailableNode{}
	}

	writeJSON(w, http.StatusOK, nodes)
}

func (h *NodeHandler) ListNodes(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	nodes, err := h.svc.ListNodes(r.Context(), claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("failed to list nodes")
		writeError(w, http.StatusInternalServerError, "failed to list nodes")
		return
	}

	if nodes == nil {
		nodes = []*service.NodeInfo{}
	}

	writeJSON(w, http.StatusOK, nodes)
}

func (h *NodeHandler) PauseNode(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node_id")
		return
	}

	if err := h.svc.UpdateNodeStatus(r.Context(), nodeID, "paused"); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to pause node")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "paused"})
}

func (h *NodeHandler) ResumeNode(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node_id")
		return
	}

	if err := h.svc.UpdateNodeStatus(r.Context(), nodeID, "active"); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resume node")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "active"})
}

type DeploymentInfo struct {
	ID             string `json:"id"`
	UserID         string `json:"user_id"`
	Image          string `json:"image"`
	GPURequired    int    `json:"gpu_required"`
	VRAMRequiredGB int64  `json:"vram_required_gb"`
	RAMRequiredGB  int64  `json:"ram_required_gb"`
	DiskRequiredGB int64  `json:"disk_required_gb"`
	Ports          string `json:"ports,omitempty"`
	Env            string `json:"env,omitempty"`
	Status         string `json:"status"`
	CostPerHour    float64 `json:"cost_per_hour"`
}

func (h *NodeHandler) ListNodeDeployments(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node_id")
		return
	}

	deployments, err := h.svc.ListNodeDeployments(r.Context(), nodeID)
	if err != nil {
		log.Error().Err(err).Msg("failed to list node deployments")
		writeError(w, http.StatusInternalServerError, "failed to list deployments")
		return
	}

	infos := make([]DeploymentInfo, 0, len(deployments))
	for _, d := range deployments {
		info := DeploymentInfo{
			ID:             d.ID.String(),
			UserID:         d.UserID.String(),
			Image:          d.Image,
			GPURequired:    d.GPURequired,
			VRAMRequiredGB: d.VRAMRequiredGB,
			RAMRequiredGB:  d.RAMRequiredGB,
			DiskRequiredGB: d.DiskRequiredGB,
			Status:         string(d.Status),
			CostPerHour:    d.CostPerHour,
		}
		if d.Ports != nil {
			info.Ports = string(d.Ports)
		}
		if d.Env != nil {
			info.Env = string(d.Env)
		}
		infos = append(infos, info)
	}

	writeJSON(w, http.StatusOK, infos)
}

type SSHKeyRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

func (h *NodeHandler) ListSSHKeys(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	keys, err := h.svc.ListSSHKeys(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list SSH keys")
		return
	}
	if keys == nil {
		keys = []*model.SSHKey{}
	}
	writeJSON(w, http.StatusOK, keys)
}

func (h *NodeHandler) AddSSHKey(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req SSHKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	key, err := h.svc.AddSSHKey(r.Context(), claims.UserID, req.Name, req.PublicKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, key)
}

func (h *NodeHandler) DeleteSSHKey(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	keyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid key id")
		return
	}

	if err := h.svc.DeleteSSHKey(r.Context(), keyID, claims.UserID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *NodeHandler) GetDefaultSSHKey(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	key, err := h.svc.GetDefaultSSHKey(r.Context(), claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"public_key": ""})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"public_key": key.PublicKey})
}

type UpdateDeploymentStatusRequest struct {
	Status string `json:"status"`
}

func (h *NodeHandler) UpdateDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	deploymentID, err := uuid.Parse(chi.URLParam(r, "deploymentId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid deployment_id")
		return
	}

	var req UpdateDeploymentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	validStatuses := map[string]bool{"running": true, "stopped": true, "failed": true, "pulling": true}
	if !validStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}

	if err := h.svc.UpdateDeploymentStatus(r.Context(), deploymentID, req.Status); err != nil {
		log.Error().Err(err).Msg("failed to update deployment status")
		writeError(w, http.StatusInternalServerError, "failed to update deployment")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *NodeHandler) WorkspaceTerminal(w http.ResponseWriter, r *http.Request) {
	// Validate auth from query param (WebSocket can't set custom headers)
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	claims := h.validateToken(tokenStr)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	depID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid deployment id", http.StatusBadRequest)
		return
	}

	dep, err := h.svc.GetDeploymentByID(r.Context(), depID)
	if err != nil {
		http.Error(w, "deployment not found", http.StatusNotFound)
		return
	}

	if dep.UserID != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if dep.NodeID == nil {
		http.Error(w, "deployment not assigned to a node", http.StatusPreconditionFailed)
		return
	}

	node, err := h.svc.GetNodeByID(r.Context(), *dep.NodeID)
	if err != nil {
		http.Error(w, "node not found", http.StatusNotFound)
		return
	}

	if node.AgentURL == "" {
		http.Error(w, "node agent URL not available", http.StatusPreconditionFailed)
		return
	}

	// Upgrade the user's connection to WebSocket
	userConn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("websocket upgrade failed")
		return
	}
	defer userConn.Close()

	// Container name on the agent
	containerName := fmt.Sprintf("aetherius-%s", dep.ID.String()[:12])

	// Connect to the agent's terminal WebSocket
	agentWSURL := url.URL{
		Scheme: "ws",
		Host:   node.AgentURL[7:], // strip http:// prefix
		Path:   "/terminal/" + containerName,
	}

	agentConn, _, err := websocket.DefaultDialer.Dial(agentWSURL.String(), nil)
	if err != nil {
		log.Error().Err(err).Str("agent_url", agentWSURL.String()).Msg("failed to connect to agent terminal")
		userConn.WriteJSON(map[string]string{"error": "failed to connect to container terminal"})
		return
	}
	defer agentConn.Close()

	// Bidirectional proxy
	done := make(chan struct{})

	// User -> Agent
	go func() {
		defer agentConn.Close()
		for {
			_, msg, err := userConn.ReadMessage()
			if err != nil {
				close(done)
				return
			}
			if err := agentConn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				close(done)
				return
			}
		}
	}()

	// Agent -> User
	go func() {
		for {
			_, msg, err := agentConn.ReadMessage()
			if err != nil {
				close(done)
				return
			}
			if err := userConn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				close(done)
				return
			}
		}
	}()

	<-done
}
