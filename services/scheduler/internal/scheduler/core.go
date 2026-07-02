package scheduler

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Priority int

const (
	PriorityCritical Priority = iota
	PriorityHigh
	PriorityNormal
	PriorityLow
)

func (p Priority) String() string {
	switch p {
	case PriorityCritical:
		return "critical"
	case PriorityHigh:
		return "high"
	case PriorityNormal:
		return "normal"
	case PriorityLow:
		return "low"
	default:
		return "unknown"
	}
}

type ResourceRequest struct {
	GPUCount    int
	VRAMBytes   int64
	RAMBytes    int64
	DiskBytes   int64
	NetworkMbps float64
	GPUModel    string
}

type PlacementPreference struct {
	Region            string
	ProhibitedRegions []string
	MaxLatencyMs      float64
	PreferLowestPrice bool
	PreferReliable    bool
}

type DeploymentRequest struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	OrgID     uuid.UUID
	Resources ResourceRequest
	Placement PlacementPreference
	Priority  Priority
	MaxBudget int64
	CreatedAt time.Time
}

type NodeSnapshot struct {
	ID            uuid.UUID
	ProviderID    uuid.UUID
	Status        string
	TotalGPU      int
	AvailableGPU  int
	TotalVRAMGB   int64
	AvailableVRAMGB int64
	TotalRAMGB    int64
	AvailableRAMGB int64
	TotalDiskGB   int64
	AvailableDiskGB int64
	GPUModels     []string
	NetworkSpeed  float64
	Region        string
	Latitude      float64
	Longitude     float64
	Reputation    float64
	Benchmark     float64
	PricePerHour  float64
	LastHeartbeat time.Time
}

type ScheduleResult struct {
	DeploymentID      uuid.UUID
	NodeID            uuid.UUID
	Score             float64
	EstimatedCostCents int64
	EstimatedLatencyMs float64
	Confidence        float64
	Error             string
}

type Scheduler struct {
	mu           sync.RWMutex
	nodes        []*NodeSnapshot
	queue        []*DeploymentRequest
	pending      map[string]*ScheduleResult
	completed    map[string]*ScheduleResult

	// Scoring weights
	weightPrice       float64
	weightLatency     float64
	weightReliability float64
	weightGPU         float64
	weightUtilization float64
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		nodes:             make([]*NodeSnapshot, 0),
		queue:             make([]*DeploymentRequest, 0),
		pending:           make(map[string]*ScheduleResult),
		completed:         make(map[string]*ScheduleResult),
		weightPrice:       0.25,
		weightLatency:     0.20,
		weightReliability: 0.25,
		weightGPU:         0.20,
		weightUtilization: 0.10,
	}
}

func (s *Scheduler) UpdateNodes(nodes []*NodeSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes = nodes
}

func (s *Scheduler) Enqueue(req *DeploymentRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Insert in priority order
	idx := sort.Search(len(s.queue), func(i int) bool {
		return s.queue[i].Priority <= req.Priority
	})
	s.queue = append(s.queue, nil)
	copy(s.queue[idx+1:], s.queue[idx:])
	s.queue[idx] = req

	log.Info().Str("deployment_id", req.ID.String()).
		Str("priority", req.Priority.String()).
		Int("queue_length", len(s.queue)).Msg("deployment enqueued")
}

func (s *Scheduler) Schedule(ctx context.Context) []*ScheduleResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.queue) == 0 || len(s.nodes) == 0 {
		return nil
	}

	var results []*ScheduleResult
	var remaining []*DeploymentRequest

	for _, req := range s.queue {
		result := s.scheduleOne(req)
		if result != nil && result.Error == "" {
			results = append(results, result)
			s.pending[req.ID.String()] = result
		} else {
			remaining = append(remaining, req)
		}
	}

	s.queue = remaining
	return results
}

func (s *Scheduler) scheduleOne(req *DeploymentRequest) *ScheduleResult {
	// Filter nodes by hard constraints
	candidates := s.filterNodes(req)

	if len(candidates) == 0 {
		return &ScheduleResult{
			DeploymentID: req.ID,
			Error:        "no suitable nodes found",
		}
	}

	// Score each candidate
	type scored struct {
		node *NodeSnapshot
		score float64
		cost  int64
	}

	var scoredNodes []scored
	for _, node := range candidates {
		score, cost := s.calculateScore(node, req)
		scoredNodes = append(scoredNodes, scored{node: node, score: score, cost: cost})
	}

	// Sort by score descending
	sort.Slice(scoredNodes, func(i, j int) bool {
		return scoredNodes[i].score > scoredNodes[j].score
	})

	best := scoredNodes[0]
	latency := s.estimateLatency(best.node, req)
	confidence := s.calculateConfidence(best.node, best.score)

	return &ScheduleResult{
		DeploymentID:       req.ID,
		NodeID:             best.node.ID,
		Score:              best.score,
		EstimatedCostCents: best.cost,
		EstimatedLatencyMs: latency,
		Confidence:         confidence,
	}
}

// Multi-dimensional bin packing with hard constraints
func (s *Scheduler) filterNodes(req *DeploymentRequest) []*NodeSnapshot {
	var filtered []*NodeSnapshot

	for _, node := range s.nodes {
		if node.Status != "active" {
			continue
		}

		// Hard constraint: GPU count
		if node.AvailableGPU < req.Resources.GPUCount {
			continue
		}

		// Hard constraint: VRAM (with 10% buffer)
		requiredVRAM := req.Resources.VRAMBytes
		availableVRAM := node.AvailableVRAMGB * 1024 * 1024 * 1024
		if availableVRAM < int64(float64(requiredVRAM)*1.1) {
			continue
		}

		// Hard constraint: RAM
		availableRAM := node.AvailableRAMGB * 1024 * 1024 * 1024
		if availableRAM < req.Resources.RAMBytes {
			continue
		}

		// Hard constraint: Disk
		availableDisk := node.AvailableDiskGB * 1024 * 1024 * 1024
		if availableDisk < req.Resources.DiskBytes {
			continue
		}

		// Hard constraint: Region
		if req.Placement.Region != "" && node.Region != req.Placement.Region {
			continue
		}

		// Hard constraint: Prohibited regions
		prohibited := false
		for _, pr := range req.Placement.ProhibitedRegions {
			if node.Region == pr {
				prohibited = true
				break
			}
		}
		if prohibited {
			continue
		}

		// GPU model preference
		if req.Resources.GPUModel != "" {
			hasGPU := false
			for _, gpu := range node.GPUModels {
				if gpu == req.Resources.GPUModel {
					hasGPU = true
					break
				}
			}
			if !hasGPU {
				continue
			}
		}

		// Budget constraint
		estimatedCost := s.estimateCost(node, req)
		if req.MaxBudget > 0 && estimatedCost > req.MaxBudget {
			continue
		}

		filtered = append(filtered, node)
	}

	return filtered
}

// Scoring formula: weighted multi-dimensional score
func (s *Scheduler) calculateScore(node *NodeSnapshot, req *DeploymentRequest) (float64, int64) {
	cost := s.estimateCost(node, req)

	// Price score: lower is better, normalized 0-1
	maxPrice := 10.0 // $10/hr max
	priceScore := 1.0 - math.Min(node.PricePerHour/maxPrice, 1.0)

	// Latency score: lower is better
	latency := s.estimateLatency(node, req)
	maxLatency := 500.0 // 500ms max
	latencyScore := 1.0 - math.Min(latency/maxLatency, 1.0)

	// Reliability score: higher is better
	reliabilityScore := node.Reputation

	// GPU match score: exact model match = 1, same family = 0.5, anything = 0
	gpuScore := 0.0
	if req.Resources.GPUModel != "" {
		for _, gpu := range node.GPUModels {
			if gpu == req.Resources.GPUModel {
				gpuScore = 1.0
				break
			}
			if gpuFamily(gpu) == gpuFamily(req.Resources.GPUModel) {
				gpuScore = 0.5
			}
		}
	} else {
		gpuScore = 0.8 // no preference
	}

	// Utilization balance: prefer nodes with moderate utilization
	utilRatio := 1.0 - float64(node.AvailableGPU)/float64(node.TotalGPU)
	utilScore := 1.0 - math.Abs(0.5-utilRatio)*2

	// Weighted sum
	totalScore :=
		s.weightPrice*priceScore +
			s.weightLatency*latencyScore +
			s.weightReliability*reliabilityScore +
			s.weightGPU*gpuScore +
			s.weightUtilization*utilScore

	// Normalize to 0-100
	totalScore = totalScore / (s.weightPrice + s.weightLatency + s.weightReliability + s.weightGPU + s.weightUtilization) * 100

	return totalScore, cost
}

func (s *Scheduler) estimateCost(node *NodeSnapshot, req *DeploymentRequest) int64 {
	// Base cost per GPU per hour in cents
	basePerGPU := 10 // $0.10/hr
	gpuCost := int64(basePerGPU * req.Resources.GPUCount)

	// VRAM premium
	vramGB := req.Resources.VRAMBytes / (1024 * 1024 * 1024)
	vramCost := vramGB * 1 // $0.01/hr per GB

	// RAM cost
	ramGB := req.Resources.RAMBytes / (1024 * 1024 * 1024)
	ramCost := ramGB * 0.5 // $0.005/hr per GB

	total := gpuCost + vramCost + ramCost

	// Apply node price multiplier
	priceMultiplier := node.PricePerHour / 0.10 // normalized to base
	total = int64(float64(total) * priceMultiplier)

	return total
}

func (s *Scheduler) estimateLatency(node *NodeSnapshot, req *DeploymentRequest) float64 {
	if req.Placement.Region == "" || node.Region == req.Placement.Region {
		return 5.0 // Same region: ~5ms
	}

	// Geo-distance-based latency estimation
	latencyMap := map[string]map[string]float64{
		"us-east":    {"us-west": 60, "eu-west": 75, "eu-central": 90, "ap-southeast": 200},
		"us-west":    {"us-east": 60, "eu-west": 140, "eu-central": 160, "ap-southeast": 120},
		"eu-west":    {"us-east": 75, "us-west": 140, "eu-central": 20, "ap-southeast": 170},
		"eu-central": {"us-east": 90, "us-west": 160, "eu-west": 20, "ap-southeast": 190},
	}

	if regionLat, ok := latencyMap[node.Region]; ok {
		if lat, ok := regionLat[req.Placement.Region]; ok {
			return lat
		}
	}

	return 150.0 // default fallback
}

func (s *Scheduler) calculateConfidence(node *NodeSnapshot, score float64) float64 {
	// Confidence based on:
	// 1. How recent the heartbeat is
	// 2. How high the score is
	// 3. Node reputation

	heartbeatAge := time.Since(node.LastHeartbeat).Seconds()
	heartbeatConf := 1.0 - math.Min(heartbeatAge/30.0, 1.0) // 0-30s scale

	scoreConf := score / 100.0
	reputationConf := node.Reputation

	confidence := (heartbeatConf * 0.4) + (scoreConf * 0.3) + (reputationConf * 0.3)
	return math.Min(confidence, 1.0)
}

func (s *Scheduler) Complete(deploymentID uuid.UUID, result *ScheduleResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pending, deploymentID.String())
	s.completed[deploymentID.String()] = result
}

func (s *Scheduler) HandleNodeOffline(nodeID uuid.UUID) []*ScheduleResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	var affected []*ScheduleResult
	for id, result := range s.pending {
		if result.NodeID == nodeID {
			affected = append(affected, result)
			delete(s.pending, id)
		}
	}

	log.Warn().Str("node_id", nodeID.String()).
		Int("affected_deployments", len(affected)).
		Msg("node offline, deployments need rescheduling")

	return affected
}

func (s *Scheduler) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"nodes_available":  len(s.nodes),
		"queue_length":    len(s.queue),
		"pending_deployments": len(s.pending),
		"completed_today":  len(s.completed),
	}
}

func gpuFamily(model string) string {
	// Group GPUs by architecture family
	families := map[string]string{
		"H100":  "hopper",
		"H200":  "hopper",
		"A100":  "ampere",
		"A6000": "ampere",
		"RTX 4090": "ada",
		"RTX 4080": "ada",
		"RTX 4070": "ada",
		"RTX 3090": "ampere",
		"RTX 3080": "ampere",
		"V100":  "volta",
		"T4":    "turing",
		"L4":    "ada",
		"L40S":  "ada",
	}

	for key, family := range families {
		if contains(model, key) {
			return family
		}
	}
	return "unknown"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
