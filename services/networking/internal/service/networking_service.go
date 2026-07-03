package service

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/aetherius/platform/services/networking/internal/model"
)

type NetworkingService struct {
	mu      sync.RWMutex
	peers   map[uuid.UUID]*model.VPNPeer
	rules   map[uuid.UUID]*model.FirewallRule
	ipIndex int
}

func NewNetworkingService() *NetworkingService {
	return &NetworkingService{
		peers: make(map[uuid.UUID]*model.VPNPeer),
		rules: make(map[uuid.UUID]*model.FirewallRule),
	}
}

func generateDummyKeyPair() (string, string) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789+/"
	b := make([]byte, 44)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	pub := make([]byte, 44)
	for i := range pub {
		pub[i] = letters[rand.Intn(len(letters))]
	}
	return string(pub), string(b)
}

func (s *NetworkingService) nextIP() string {
	s.ipIndex++
	ip := s.ipIndex
	return fmt.Sprintf("10.0.%d.%d", (ip>>8)&0xFF, ip&0xFF)
}

func (s *NetworkingService) CreateVPNSession(userID, nodeID uuid.UUID) *model.VPNPeer {
	pubKey, privKey := generateDummyKeyPair()

	peer := &model.VPNPeer{
		ID:         uuid.New(),
		UserID:     userID,
		NodeID:     nodeID,
		PublicKey:  pubKey,
		PrivateKey: privKey,
		AllowedIPs: []string{s.nextIP() + "/32"},
		Endpoint:   "",
		Status:     "active",
		CreatedAt:  time.Now().UTC(),
	}

	s.mu.Lock()
	s.peers[peer.ID] = peer
	s.mu.Unlock()

	return peer
}

func (s *NetworkingService) GetVPNSessions(userID uuid.UUID) []*model.VPNPeer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.VPNPeer
	for _, p := range s.peers {
		if p.UserID == userID {
			cp := *p
			cp.PrivateKey = ""
			result = append(result, &cp)
		}
	}
	return result
}

func (s *NetworkingService) DeleteVPNSession(peerID, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.peers[peerID]
	if !ok {
		return fmt.Errorf("peer not found")
	}
	if p.UserID != userID {
		return fmt.Errorf("peer not found")
	}
	delete(s.peers, peerID)
	return nil
}

func (s *NetworkingService) GetNetworkConfig(userID uuid.UUID) *model.NetworkConfig {
	s.mu.RLock()
	peers := make([]model.VPNPeer, 0)
	for _, p := range s.peers {
		if p.UserID == userID {
			cp := *p
			cp.PrivateKey = ""
			peers = append(peers, cp)
		}
	}
	s.mu.RUnlock()

	return &model.NetworkConfig{
		WireGuardInterface: "wg0",
		Address:            "10.0.0.1/16",
		ListenPort:         51820,
		DNSServers:         []string{"1.1.1.1", "8.8.8.8"},
		Peers:              peers,
	}
}

func (s *NetworkingService) AddFirewallRule(userID uuid.UUID, rule *model.FirewallRule) *model.FirewallRule {
	rule.ID = uuid.New()
	rule.UserID = userID
	rule.CreatedAt = time.Now().UTC()
	if rule.Priority == 0 {
		rule.Priority = 100
	}
	rule.Enabled = true
	if rule.Protocol == "" {
		rule.Protocol = "any"
	}
	if rule.Direction == "" {
		rule.Direction = "inbound"
	}
	if rule.Action == "" {
		rule.Action = "allow"
	}

	s.mu.Lock()
	s.rules[rule.ID] = rule
	s.mu.Unlock()

	return rule
}

// Status field isn't on FirewallRule - use Enabled field
// Already handled above by setting Enabled = true

func (s *NetworkingService) ListFirewallRules(userID uuid.UUID) []*model.FirewallRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.FirewallRule
	for _, r := range s.rules {
		if r.UserID == userID {
			cp := *r
			result = append(result, &cp)
		}
	}
	return result
}

func (s *NetworkingService) DeleteFirewallRule(ruleID, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.rules[ruleID]
	if !ok {
		return fmt.Errorf("rule not found")
	}
	if r.UserID != userID {
		return fmt.Errorf("rule not found")
	}
	delete(s.rules, ruleID)
	return nil
}
