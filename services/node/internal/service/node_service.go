package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/queue"
	"github.com/aetherius/platform/services/node/internal/model"
	"github.com/aetherius/platform/services/node/internal/repository"
)

type NodeService struct {
	repo    *repository.NodeRepository
	queue   *queue.Client
}

func NewNodeService(repo *repository.NodeRepository, q *queue.Client) *NodeService {
	return &NodeService{
		repo:  repo,
		queue: q,
	}
}

func (s *NodeService) RegisterNode(ctx context.Context, providerID uuid.UUID, hardware HardwareInfo, agentVersion, publicIP string) (*model.Node, string, error) {
	fingerprint := s.generateFingerprint(hardware)
	nodeToken := generateNodeToken()

	node := &model.Node{
		ID:                 uuid.New(),
		ProviderID:         providerID,
		Status:             model.NodeStatusPending,
		HardwareFingerprint: fingerprint,
		TotalGPU:           len(hardware.GPUs),
		AvailableGPU:       len(hardware.GPUs),
		TotalVRAMGB:        s.sumVRAM(hardware.GPUs),
		AvailableVRAMGB:    s.sumVRAM(hardware.GPUs),
		TotalRAMGB:         hardware.RAM.TotalBytes / (1024 * 1024 * 1024),
		AvailableRAMGB:     hardware.RAM.TotalBytes / (1024 * 1024 * 1024),
		TotalDiskGB:        hardware.Disk.TotalBytes / (1024 * 1024 * 1024),
		AvailableDiskGB:    hardware.Disk.TotalBytes / (1024 * 1024 * 1024),
		CPUModel:           hardware.CPU.Model,
		CPUCores:           hardware.CPU.Cores,
		GPUModels:          s.extractGPUNames(hardware.GPUs),
		NetworkSpeedMbps:   hardware.Network.SpeedMbps,
		PublicIP:           publicIP,
		Region:             hardware.Network.Region,
		Country:            hardware.Network.Country,
		City:               hardware.Network.City,
		Latitude:           hardware.Network.Latitude,
		Longitude:          hardware.Network.Longitude,
		CUDAVersion:        hardware.CUDAVersion,
		DockerVersion:      hardware.DockerVersion,
		OSName:             hardware.OSName,
		AgentVersion:       agentVersion,
		NodeToken:          nodeToken,
	}

	if err := s.repo.Create(ctx, node); err != nil {
		return nil, "", fmt.Errorf("create node: %w", err)
	}

	// Publish node registered event
	payload, _ := json.Marshal(map[string]interface{}{
		"node_id": node.ID.String(),
		"status":  node.Status,
	})
	s.queue.Publish(context.Background(), queue.ExchangeDomainEvents,
		"node.registered", queue.Event{
			Type:      "node.registered",
			Source:    "node-service",
			Payload:   payload,
			Timestamp: time.Now(),
		})

	log.Info().Str("node_id", node.ID.String()).Str("provider", providerID.String()).
		Int("gpus", node.TotalGPU).Msg("node registered")

	return node, nodeToken, nil
}

func (s *NodeService) ProcessHeartbeat(ctx context.Context, nodeID uuid.UUID, hb HeartbeatData) error {
	node, err := s.repo.GetByID(ctx, nodeID)
	if err != nil {
		return err
	}

	// Update node status if it was offline
	if node.Status == model.NodeStatusOffline || node.Status == model.NodeStatusPending {
		s.repo.UpdateHeartbeat(ctx, nodeID, model.NodeStatusActive)
	}

	// Update available resources based on heartbeat
	availVRAM := node.TotalVRAMGB
	for i, used := range hb.VRAMUsed {
		if i < len(hb.GPUUtil) {
			availVRAM -= used / (1024 * 1024 * 1024)
		}
	}
	availRAM := node.TotalRAMGB - (hb.RAMUsedGB)
	availDisk := node.TotalDiskGB - hb.DiskUsedGB
	s.repo.UpdateResources(ctx, nodeID, node.TotalGPU-len(hb.RunningContainers), availVRAM, availRAM, availDisk)

	// Record heartbeat
	heartbeat := &model.Heartbeat{
		ID:               uuid.New(),
		NodeID:           nodeID,
		Status:           model.NodeStatusActive,
		GPUUtilization:   hb.GPUUtil,
		GPUTemp:          hb.GPUTemps,
		VRAMUsed:         hb.VRAMUsed,
		CPUUtilization:   hb.CPUUtil,
		RAMUsedGB:        hb.RAMUsedGB,
		DiskUsedGB:       hb.DiskUsedGB,
		NetworkRXBytes:   hb.NetworkRXBytes,
		NetworkTXBytes:   hb.NetworkTXBytes,
		LoadAverage:      hb.LoadAvg,
		UptimeSeconds:    hb.UptimeSeconds,
		RunningContainer: len(hb.RunningContainers),
		ReportedAt:       time.Now(),
	}
	s.repo.InsertHeartbeat(ctx, heartbeat)

	// Publish heartbeat event
	payload, _ := json.Marshal(map[string]interface{}{
		"node_id":           nodeID.String(),
		"gpu_utilization":   hb.GPUUtil,
		"cpu_utilization":   hb.CPUUtil,
		"running_containers": len(hb.RunningContainers),
	})
	s.queue.Publish(context.Background(), queue.ExchangeDomainEvents,
		queue.RoutingKeyNodeHeartbeat, queue.Event{
			Type:      "node.heartbeat",
			Source:    "node-service",
			Payload:   payload,
			Timestamp: time.Now(),
		})

	return nil
}

func (s *NodeService) FindAvailableNodes(ctx context.Context, reqGPU int, reqVRAM, reqRAM, reqDisk int64, region string, limit int) ([]ScoredNode, error) {
	nodes, err := s.repo.GetAvailable(ctx, reqGPU, reqVRAM, reqRAM, reqDisk, region, limit)
	if err != nil {
		return nil, err
	}

	scored := make([]ScoredNode, 0, len(nodes))
	for _, n := range nodes {
		score := s.calculateScore(n)
		price := s.estimatePrice(n)
		scored = append(scored, ScoredNode{
			Node:                 n,
			Score:               score,
			EstimatedPricePerHour: price,
			EstimatedLatencyMs:   s.estimateLatency(n),
		})
	}

	// Sort by score descending
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[i].Score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	return scored, nil
}

func (s *NodeService) calculateScore(node *model.Node) float64 {
	reliability := node.ReputationScore * 0.4
	availability := float64(node.AvailableGPU) / float64(node.TotalGPU) * 0.3
	utilization := 1.0 - math.Abs(0.5-float64(node.AvailableGPU)/float64(node.TotalGPU)) * 0.2
	speedBonus := math.Min(node.NetworkSpeedMbps/1000.0, 1.0) * 0.1

	return reliability + availability + utilization + speedBonus
}

func (s *NodeService) estimatePrice(node *model.Node) float64 {
	basePrice := 0.10 // $0.10/hr base for 1 GPU
	gpuMultiplier := float64(node.TotalGPU) * 0.08
	vramMultiplier := float64(node.TotalVRAMGB) * 0.001
	return basePrice + gpuMultiplier + vramMultiplier
}

func (s *NodeService) estimateLatency(node *model.Node) float64 {
	// Simplified latency estimation based on region
	latencies := map[string]float64{
		"us-east": 5, "us-west": 10, "eu-west": 15,
		"eu-central": 20, "ap-southeast": 50, "ap-northeast": 45,
	}
	if lat, ok := latencies[node.Region]; ok {
		return lat
	}
	return 100
}

func (s *NodeService) generateFingerprint(hw HardwareInfo) string {
	data := fmt.Sprintf("%s-%d-%d-%s-%s-%s",
		hw.CPU.Model, hw.CPU.Cores, hw.RAM.TotalBytes,
		hw.Disk.Filesystem, hw.Network.PublicIP, hw.OSName)
	hash := make([]byte, 32)
	copy(hash, []byte(data))
	return hex.EncodeToString(hash[:16])
}

func (s *NodeService) sumVRAM(gpus []GPUInfo) int64 {
	var total int64
	for _, gpu := range gpus {
		total += gpu.VRAMBytes
	}
	return total / (1024 * 1024 * 1024)
}

func (s *NodeService) extractGPUNames(gpus []GPUInfo) []string {
	names := make([]string, len(gpus))
	for i, gpu := range gpus {
		names[i] = gpu.Model
	}
	return names
}

type HardwareInfo struct {
	GPUs          []GPUInfo
	CPU           CPUInfo
	RAM           MemoryInfo
	Disk          DiskInfo
	Network       NetworkInfo
	CUDAVersion   string
	ROCmVersion   string
	DockerVersion string
	OSName        string
}

type GPUInfo struct {
	Model     string
	VRAMBytes int64
	Cores     int
}

type CPUInfo struct {
	Model string
	Cores int
}

type MemoryInfo struct {
	TotalBytes int64
}

type DiskInfo struct {
	TotalBytes int64
	Filesystem string
}

type NetworkInfo struct {
	SpeedMbps float64
	PublicIP  string
	Region    string
	Country   string
	City      string
	Latitude  float64
	Longitude float64
}

type HeartbeatData struct {
	GPUUtil           []float64
	GPUTemps          []float64
	VRAMUsed          []int64
	CPUUtil           float64
	RAMUsedGB         int64
	DiskUsedGB        int64
	NetworkRXBytes    int64
	NetworkTXBytes    int64
	LoadAvg           float64
	UptimeSeconds     int64
	RunningContainers []string
}

type ScoredNode struct {
	Node                 *model.Node
	Score               float64
	EstimatedPricePerHour float64
	EstimatedLatencyMs   float64
}

func generateNodeToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "nt_" + hex.EncodeToString(b)
}

type NodeInfo struct {
	ID               uuid.UUID          `json:"id"`
	Status           string             `json:"status"`
	TotalGPU         int                `json:"total_gpu"`
	AvailableGPU     int                `json:"available_gpu"`
	TotalVRAMGB      int64              `json:"total_vram_gb"`
	TotalRAMGB       int64              `json:"total_ram_gb"`
	TotalDiskGB      int64              `json:"total_disk_gb"`
	CPUModel         string             `json:"cpu_model"`
	CPUCores         int                `json:"cpu_cores"`
	GPUModels        []string           `json:"gpu_models"`
	OSName           string             `json:"os_name"`
	Region           string             `json:"region"`
	FirstSeen        time.Time          `json:"first_seen"`
	LastHeartbeat    time.Time          `json:"last_heartbeat"`
	CreatedAt        time.Time          `json:"created_at"`
}

func (s *NodeService) GetNode(ctx context.Context, nodeID uuid.UUID) (*NodeInfo, error) {
	node, err := s.repo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	return nodeToInfo(node), nil
}

func (s *NodeService) ListNodes(ctx context.Context, providerID uuid.UUID) ([]*NodeInfo, error) {
	nodes, err := s.repo.ListByProviderID(ctx, providerID)
	if err != nil {
		return nil, err
	}
	infos := make([]*NodeInfo, len(nodes))
	for i, n := range nodes {
		infos[i] = nodeToInfo(n)
	}
	return infos, nil
}

func (s *NodeService) UpdateNodeStatus(ctx context.Context, nodeID uuid.UUID, status string) error {
	return s.repo.UpdateStatus(ctx, nodeID, model.NodeStatus(status))
}

func nodeToInfo(n *model.Node) *NodeInfo {
	return &NodeInfo{
		ID:            n.ID,
		Status:        string(n.Status),
		TotalGPU:      n.TotalGPU,
		AvailableGPU:  n.AvailableGPU,
		TotalVRAMGB:   n.TotalVRAMGB,
		TotalRAMGB:    n.TotalRAMGB,
		TotalDiskGB:   n.TotalDiskGB,
		CPUModel:      n.CPUModel,
		CPUCores:      n.CPUCores,
		GPUModels:     n.GPUModels,
		OSName:        n.OSName,
		Region:        n.Region,
		FirstSeen:     n.FirstSeen,
		LastHeartbeat: n.LastHeartbeat,
		CreatedAt:     n.CreatedAt,
	}
}

// Background task: mark nodes as offline if no heartbeat for 60s
func (s *NodeService) CheckOfflineNodes(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		nodes, err := s.repo.ListByStatus(ctx, model.NodeStatusActive)
		if err != nil {
			log.Error().Err(err).Msg("failed to list active nodes for offline check")
			continue
		}

		for _, node := range nodes {
			if time.Since(node.LastHeartbeat) > 60*time.Second {
				s.repo.UpdateStatus(ctx, node.ID, model.NodeStatusOffline)

				payload, _ := json.Marshal(map[string]string{
					"node_id": node.ID.String(),
					"reason":  "heartbeat timeout",
				})
				s.queue.Publish(context.Background(), queue.ExchangeDomainEvents,
					queue.RoutingKeyNodeOffline, queue.Event{
						Type:      "node.offline",
						Source:    "node-service",
						Payload:   payload,
						Timestamp: time.Now(),
					})

				log.Warn().Str("node_id", node.ID.String()).Msg("node marked offline due to heartbeat timeout")

				// Publish to dead letter for scheduler to handle failover
				s.queue.Publish(context.Background(), queue.ExchangeDeadLetter,
					"node.offline", queue.Event{
						Type:    "node.offline",
						Source:  "node-service",
						Payload: payload,
					})
			}
		}
	}
}

// Ensure NodeService implements error properly
var ErrInsufficientResources = errors.New("insufficient node resources")
