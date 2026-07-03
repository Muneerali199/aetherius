package model

import (
	"time"

	"github.com/google/uuid"
)

type VPNPeer struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	NodeID     uuid.UUID  `json:"node_id"`
	PublicKey  string     `json:"public_key"`
	PrivateKey string     `json:"private_key,omitempty"`
	AllowedIPs []string   `json:"allowed_ips"`
	Endpoint   string     `json:"endpoint,omitempty"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
}

type NetworkConfig struct {
	WireGuardInterface string    `json:"wireguard_interface"`
	Address            string    `json:"address"`
	ListenPort         int       `json:"listen_port"`
	DNSServers         []string  `json:"dns_servers"`
	Peers              []VPNPeer `json:"peers"`
}

type FirewallRule struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Direction string    `json:"direction"`
	Protocol  string    `json:"protocol"`
	Port      int       `json:"port"`
	CIDR      string    `json:"cidr"`
	Action    string    `json:"action"`
	Priority  int       `json:"priority"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}
