package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aetherius/platform/services/node/internal/model"
)

var (
	ErrNodeNotFound = errors.New("node not found")
)

type NodeRepository struct {
	pool *pgxpool.Pool
}

func NewNodeRepository(pool *pgxpool.Pool) *NodeRepository {
	return &NodeRepository{pool: pool}
}

func (r *NodeRepository) Create(ctx context.Context, node *model.Node) error {
	query := `
		INSERT INTO nodes (
			id, provider_id, status, hardware_fingerprint,
			total_gpu, available_gpu, total_vram_gb, available_vram_gb,
			total_ram_gb, available_ram_gb, total_disk_gb, available_disk_gb,
			cpu_model, cpu_cores, gpu_models, network_speed_mbps,
			public_ip, region, country, city, latitude, longitude,
			cuda_version, docker_version, os_name, agent_version, node_token, agent_url
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22,
			$23, $24, $25, $26, $27, $28
		) RETURNING first_seen, last_heartbeat, created_at`

	err := r.pool.QueryRow(ctx, query,
		node.ID, node.ProviderID, node.Status, node.HardwareFingerprint,
		node.TotalGPU, node.AvailableGPU, node.TotalVRAMGB, node.AvailableVRAMGB,
		node.TotalRAMGB, node.AvailableRAMGB, node.TotalDiskGB, node.AvailableDiskGB,
		node.CPUModel, node.CPUCores, node.GPUModels, node.NetworkSpeedMbps,
		node.PublicIP, node.Region, node.Country, node.City, node.Latitude, node.Longitude,
		node.CUDAVersion, node.DockerVersion, node.OSName, node.AgentVersion, node.NodeToken, node.AgentURL,
	).Scan(&node.FirstSeen, &node.LastHeartbeat, &node.CreatedAt)

	return err
}

func (r *NodeRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Node, error) {
	query := `SELECT * FROM nodes WHERE id = $1`
	node := &model.Node{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&node.ID, &node.ProviderID, &node.Status, &node.HardwareFingerprint,
		&node.BenchmarkScore, &node.ReputationScore,
		&node.TotalGPU, &node.AvailableGPU, &node.TotalVRAMGB, &node.AvailableVRAMGB,
		&node.TotalRAMGB, &node.AvailableRAMGB, &node.TotalDiskGB, &node.AvailableDiskGB,
		&node.CPUModel, &node.CPUCores, &node.GPUModels, &node.NetworkSpeedMbps,
		&node.PublicIP, &node.Region, &node.Country, &node.City, &node.Latitude, &node.Longitude,
		&node.CUDAVersion, &node.DockerVersion, &node.OSName, &node.AgentVersion,
			&node.NodeToken, &node.FirstSeen, &node.LastHeartbeat, &node.CreatedAt, &node.UpdatedAt, &node.AgentURL,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNodeNotFound
	}
	return node, err
}

func (r *NodeRepository) UpdateHeartbeat(ctx context.Context, nodeID uuid.UUID, status model.NodeStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE nodes SET status = $1, last_heartbeat = NOW() WHERE id = $2`,
		status, nodeID,
	)
	return err
}

func (r *NodeRepository) UpdateResources(ctx context.Context, nodeID uuid.UUID, availGPU int, availVRAM, availRAM, availDisk int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE nodes SET available_gpu = $1, available_vram_gb = $2,
		available_ram_gb = $3, available_disk_gb = $4, last_heartbeat = NOW()
		WHERE id = $5`,
		availGPU, availVRAM, availRAM, availDisk, nodeID,
	)
	return err
}

func (r *NodeRepository) GetAvailable(ctx context.Context, reqGPU int, reqVRAM, reqRAM, reqDisk int64, region string, limit int) ([]*model.Node, error) {
	query := `
		SELECT * FROM nodes
		WHERE status = 'active'
		AND available_gpu >= $1
		AND available_vram_gb >= $2
		AND available_ram_gb >= $3
		AND available_disk_gb >= $4
		AND (region = $5 OR $5 = '')
		AND last_heartbeat > NOW() - INTERVAL '30 seconds'
		ORDER BY reputation_score DESC, benchmark_score DESC
		LIMIT $6`

	rows, err := r.pool.Query(ctx, query, reqGPU, reqVRAM, reqRAM, reqDisk, region, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*model.Node
	for rows.Next() {
		node := &model.Node{}
		err := rows.Scan(
			&node.ID, &node.ProviderID, &node.Status, &node.HardwareFingerprint,
			&node.BenchmarkScore, &node.ReputationScore,
			&node.TotalGPU, &node.AvailableGPU, &node.TotalVRAMGB, &node.AvailableVRAMGB,
			&node.TotalRAMGB, &node.AvailableRAMGB, &node.TotalDiskGB, &node.AvailableDiskGB,
			&node.CPUModel, &node.CPUCores, &node.GPUModels, &node.NetworkSpeedMbps,
			&node.PublicIP, &node.Region, &node.Country, &node.City, &node.Latitude, &node.Longitude,
			&node.CUDAVersion, &node.DockerVersion, &node.OSName, &node.AgentVersion,
			&node.NodeToken, &node.FirstSeen, &node.LastHeartbeat, &node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (r *NodeRepository) ListByProviderID(ctx context.Context, providerID uuid.UUID) ([]*model.Node, error) {
	query := `SELECT * FROM nodes WHERE provider_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*model.Node
	for rows.Next() {
		node := &model.Node{}
		rows.Scan(
			&node.ID, &node.ProviderID, &node.Status, &node.HardwareFingerprint,
			&node.BenchmarkScore, &node.ReputationScore,
			&node.TotalGPU, &node.AvailableGPU, &node.TotalVRAMGB, &node.AvailableVRAMGB,
			&node.TotalRAMGB, &node.AvailableRAMGB, &node.TotalDiskGB, &node.AvailableDiskGB,
			&node.CPUModel, &node.CPUCores, &node.GPUModels, &node.NetworkSpeedMbps,
			&node.PublicIP, &node.Region, &node.Country, &node.City, &node.Latitude, &node.Longitude,
			&node.CUDAVersion, &node.DockerVersion, &node.OSName, &node.AgentVersion,
			&node.NodeToken, &node.FirstSeen, &node.LastHeartbeat, &node.CreatedAt, &node.UpdatedAt,
		)
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (r *NodeRepository) ListByStatus(ctx context.Context, status model.NodeStatus) ([]*model.Node, error) {
	query := `SELECT * FROM nodes WHERE status = $1 ORDER BY last_heartbeat DESC`
	rows, err := r.pool.Query(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*model.Node
	for rows.Next() {
		node := &model.Node{}
		rows.Scan(
			&node.ID, &node.ProviderID, &node.Status, &node.HardwareFingerprint,
			&node.BenchmarkScore, &node.ReputationScore,
			&node.TotalGPU, &node.AvailableGPU, &node.TotalVRAMGB, &node.AvailableVRAMGB,
			&node.TotalRAMGB, &node.AvailableRAMGB, &node.TotalDiskGB, &node.AvailableDiskGB,
			&node.CPUModel, &node.CPUCores, &node.GPUModels, &node.NetworkSpeedMbps,
			&node.PublicIP, &node.Region, &node.Country, &node.City, &node.Latitude, &node.Longitude,
			&node.CUDAVersion, &node.DockerVersion, &node.OSName, &node.AgentVersion,
			&node.NodeToken, &node.FirstSeen, &node.LastHeartbeat, &node.CreatedAt, &node.UpdatedAt,
		)
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (r *NodeRepository) ListActive(ctx context.Context) ([]*model.Node, error) {
	query := `SELECT * FROM nodes WHERE status = 'active' ORDER BY total_gpu DESC, total_vram_gb DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*model.Node
	for rows.Next() {
		node := &model.Node{}
		if err := rows.Scan(
			&node.ID, &node.ProviderID, &node.Status, &node.HardwareFingerprint,
			&node.BenchmarkScore, &node.ReputationScore,
			&node.TotalGPU, &node.AvailableGPU, &node.TotalVRAMGB, &node.AvailableVRAMGB,
			&node.TotalRAMGB, &node.AvailableRAMGB, &node.TotalDiskGB, &node.AvailableDiskGB,
			&node.CPUModel, &node.CPUCores, &node.GPUModels, &node.NetworkSpeedMbps,
			&node.PublicIP, &node.Region, &node.Country, &node.City, &node.Latitude, &node.Longitude,
			&node.CUDAVersion, &node.DockerVersion, &node.OSName, &node.AgentVersion,
			&node.NodeToken, &node.FirstSeen, &node.LastHeartbeat, &node.CreatedAt, &node.UpdatedAt,
			&node.AgentURL,
		); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

func (r *NodeRepository) GetByFingerprint(ctx context.Context, fingerprint string) (*model.Node, error) {
	query := `SELECT * FROM nodes WHERE hardware_fingerprint = $1`
	node := &model.Node{}
	err := r.pool.QueryRow(ctx, query, fingerprint).Scan(
		&node.ID, &node.ProviderID, &node.Status, &node.HardwareFingerprint,
		&node.BenchmarkScore, &node.ReputationScore,
		&node.TotalGPU, &node.AvailableGPU, &node.TotalVRAMGB, &node.AvailableVRAMGB,
		&node.TotalRAMGB, &node.AvailableRAMGB, &node.TotalDiskGB, &node.AvailableDiskGB,
		&node.CPUModel, &node.CPUCores, &node.GPUModels, &node.NetworkSpeedMbps,
		&node.PublicIP, &node.Region, &node.Country, &node.City, &node.Latitude, &node.Longitude,
		&node.CUDAVersion, &node.DockerVersion, &node.OSName, &node.AgentVersion,
			&node.NodeToken, &node.FirstSeen, &node.LastHeartbeat, &node.CreatedAt, &node.UpdatedAt, &node.AgentURL,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return node, err
}

func (r *NodeRepository) UpdateNode(ctx context.Context, node *model.Node) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE nodes SET
			status = $1, node_token = $2, available_gpu = $3, available_vram_gb = $4,
			available_ram_gb = $5, available_disk_gb = $6, agent_version = $7,
			agent_url = $8, updated_at = NOW(), last_heartbeat = NOW()
		WHERE id = $9`,
		node.Status, node.NodeToken, node.AvailableGPU, node.AvailableVRAMGB,
		node.AvailableRAMGB, node.AvailableDiskGB, node.AgentVersion, node.AgentURL, node.ID,
	)
	return err
}

func (r *NodeRepository) InsertHeartbeat(ctx context.Context, hb *model.Heartbeat) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO node_heartbeats (
			id, node_id, status, gpu_utilization, gpu_temp,
			vram_used, cpu_utilization, ram_used_gb, disk_used_gb,
			network_rx_bytes, network_tx_bytes, load_average,
			uptime_seconds, running_containers, reported_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		hb.ID, hb.NodeID, hb.Status, hb.GPUUtilization, hb.GPUTemp,
		hb.VRAMUsed, hb.CPUUtilization, hb.RAMUsedGB, hb.DiskUsedGB,
		hb.NetworkRXBytes, hb.NetworkTXBytes, hb.LoadAverage,
		hb.UptimeSeconds, hb.RunningContainer, hb.ReportedAt,
	)
	return err
}

func (r *NodeRepository) UpdateAgentURL(ctx context.Context, nodeID uuid.UUID, agentURL string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE nodes SET agent_url = $1, updated_at = NOW() WHERE id = $2`,
		agentURL, nodeID,
	)
	return err
}

func (r *NodeRepository) UpdateStatus(ctx context.Context, nodeID uuid.UUID, status model.NodeStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE nodes SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, nodeID,
	)
	return err
}

func (r *NodeRepository) GetDeploymentByID(ctx context.Context, id uuid.UUID) (*model.Deployment, error) {
	var d model.Deployment
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, node_id, image, gpu_required, vram_required_gb,
			ram_required_gb, disk_required_gb, ports, env, status,
			cost_per_hour, region, assigned_at, created_at, updated_at
		FROM deployments WHERE id = $1`, id).Scan(
		&d.ID, &d.UserID, &d.NodeID, &d.Image,
		&d.GPURequired, &d.VRAMRequiredGB, &d.RAMRequiredGB,
		&d.DiskRequiredGB, &d.Ports, &d.Env, &d.Status,
		&d.CostPerHour, &d.Region, &d.AssignedAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNodeNotFound
	}
	return &d, err
}

func (r *NodeRepository) ListDeploymentsByNodeID(ctx context.Context, nodeID uuid.UUID) ([]model.Deployment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, node_id, image, gpu_required, vram_required_gb,
			ram_required_gb, disk_required_gb, ports, env, status,
			cost_per_hour, region, assigned_at, created_at, updated_at
		FROM deployments WHERE node_id = $1 AND status IN ('scheduling', 'running')
		ORDER BY created_at DESC`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deployments []model.Deployment
	for rows.Next() {
		var d model.Deployment
		err := rows.Scan(&d.ID, &d.UserID, &d.NodeID, &d.Image,
			&d.GPURequired, &d.VRAMRequiredGB, &d.RAMRequiredGB,
			&d.DiskRequiredGB, &d.Ports, &d.Env, &d.Status,
			&d.CostPerHour, &d.Region, &d.AssignedAt, &d.CreatedAt, &d.UpdatedAt)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, d)
	}
	return deployments, nil
}

func (r *NodeRepository) UpdateDeploymentStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE deployments SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id)
	return err
}

func (r *NodeRepository) ListSSHKeys(ctx context.Context, userID uuid.UUID) ([]*model.SSHKey, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, user_id, name, public_key, fingerprint, is_default, created_at FROM ssh_keys WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []*model.SSHKey
	for rows.Next() {
		k := &model.SSHKey{}
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.PublicKey, &k.Fingerprint, &k.IsDefault, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *NodeRepository) CreateSSHKey(ctx context.Context, key *model.SSHKey) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO ssh_keys (id, user_id, name, public_key, fingerprint, is_default) VALUES ($1,$2,$3,$4,$5,$6) RETURNING created_at`,
		key.ID, key.UserID, key.Name, key.PublicKey, key.Fingerprint, key.IsDefault,
	).Scan(&key.CreatedAt)
}

func (r *NodeRepository) DeleteSSHKey(ctx context.Context, id, userID uuid.UUID) error {
	res, err := r.pool.Exec(ctx, `DELETE FROM ssh_keys WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New("SSH key not found")
	}
	return nil
}

func (r *NodeRepository) GetDefaultSSHKey(ctx context.Context, userID uuid.UUID) (*model.SSHKey, error) {
	k := &model.SSHKey{}
	err := r.pool.QueryRow(ctx, `SELECT id, user_id, name, public_key, fingerprint, is_default, created_at FROM ssh_keys WHERE user_id = $1 AND is_default = true LIMIT 1`, userID).
		Scan(&k.ID, &k.UserID, &k.Name, &k.PublicKey, &k.Fingerprint, &k.IsDefault, &k.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNodeNotFound
	}
	return k, err
}

func (r *NodeRepository) GetNodeByToken(ctx context.Context, token string) (*model.Node, error) {
	query := `SELECT * FROM nodes WHERE node_token = $1`
	node := &model.Node{}
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&node.ID, &node.ProviderID, &node.Status, &node.HardwareFingerprint,
		&node.BenchmarkScore, &node.ReputationScore,
		&node.TotalGPU, &node.AvailableGPU, &node.TotalVRAMGB, &node.AvailableVRAMGB,
		&node.TotalRAMGB, &node.AvailableRAMGB, &node.TotalDiskGB, &node.AvailableDiskGB,
		&node.CPUModel, &node.CPUCores, &node.GPUModels, &node.NetworkSpeedMbps,
		&node.PublicIP, &node.Region, &node.Country, &node.City, &node.Latitude, &node.Longitude,
		&node.CUDAVersion, &node.DockerVersion, &node.OSName, &node.AgentVersion,
			&node.NodeToken, &node.FirstSeen, &node.LastHeartbeat, &node.CreatedAt, &node.UpdatedAt, &node.AgentURL,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNodeNotFound
	}
	return node, err
}
