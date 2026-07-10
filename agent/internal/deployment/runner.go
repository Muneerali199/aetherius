package deployment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type Runner struct {
	serverURL string
	apiToken  string
	nodeID    string
	interval  time.Duration
	client    *http.Client
}

type Deployment struct {
	ID             string  `json:"id"`
	UserID         string  `json:"user_id"`
	Image          string  `json:"image"`
	GPURequired    int     `json:"gpu_required"`
	VRAMRequiredGB int64   `json:"vram_required_gb"`
	RAMRequiredGB  int64   `json:"ram_required_gb"`
	DiskRequiredGB int64   `json:"disk_required_gb"`
	Ports          string  `json:"ports,omitempty"`
	Env            string  `json:"env,omitempty"`
	Status         string  `json:"status"`
	CostPerHour    float64 `json:"cost_per_hour"`
}

func NewRunner(serverURL, apiToken, nodeID string) *Runner {
	return &Runner{
		serverURL: serverURL,
		apiToken:  apiToken,
		nodeID:    nodeID,
		interval:  15 * time.Second,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (r *Runner) Start(ctx context.Context) error {
	log.Info().Str("node_id", r.nodeID).Msg("deployment runner started")

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			r.pollDeployments(ctx)
		}
	}
}

func (r *Runner) pollDeployments(ctx context.Context) {
	deployments, err := r.fetchDeployments()
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch deployments")
		return
	}

	for _, d := range deployments {
		switch d.Status {
		case "scheduling":
			r.handleScheduling(ctx, d)
		case "running":
			r.checkRunning(ctx, d)
		case "stopping":
			r.handleStopping(ctx, d)
		}
	}
}

func (r *Runner) fetchDeployments() ([]Deployment, error) {
	req, err := http.NewRequest("GET", r.serverURL+"/v1/nodes/"+r.nodeID+"/deployments", nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("fetch deployments: %d", resp.StatusCode)
	}

	var deployments []Deployment
	if err := json.NewDecoder(resp.Body).Decode(&deployments); err != nil {
		return nil, err
	}
	return deployments, nil
}

func (r *Runner) handleScheduling(ctx context.Context, d Deployment) {
	log.Info().Str("deployment", d.ID).Str("image", d.Image).Msg("starting deployment")

	r.updateStatus(d.ID, "pulling")

	containerName := fmt.Sprintf("aetherius-%s", d.ID[:12])

	if err := r.pullImage(d.Image); err != nil {
		log.Error().Err(err).Str("image", d.Image).Msg("pull failed")
		r.updateStatus(d.ID, "failed")
		return
	}

	containerID, err := r.runContainer(containerName, d)
	if err != nil {
		log.Error().Err(err).Str("image", d.Image).Msg("run failed")
		r.updateStatus(d.ID, "failed")
		return
	}

	log.Info().Str("deployment", d.ID).Str("container", containerID).Msg("deployment running")
	r.updateStatus(d.ID, "running")
}

func (r *Runner) checkRunning(ctx context.Context, d Deployment) {
	containerName := fmt.Sprintf("aetherius-%s", d.ID[:12])
	running, err := r.isContainerRunning(containerName)
	if err != nil || !running {
		log.Warn().Str("deployment", d.ID).Msg("container not running, marking stopped")
		r.updateStatus(d.ID, "stopped")
	}
}

func (r *Runner) handleStopping(ctx context.Context, d Deployment) {
	containerName := fmt.Sprintf("aetherius-%s", d.ID[:12])
	log.Info().Str("deployment", d.ID).Msg("stopping deployment")

	if err := r.stopContainer(containerName); err != nil {
		log.Error().Err(err).Msg("stop container failed")
	}

	r.updateStatus(d.ID, "stopped")
}

func (r *Runner) updateStatus(deploymentID, status string) {
	body, _ := json.Marshal(map[string]string{"status": status})
	req, err := http.NewRequest("POST",
		r.serverURL+"/v1/nodes/"+r.nodeID+"/deployments/"+deploymentID+"/status",
		bytes.NewReader(body))
	if err != nil {
		log.Error().Err(err).Msg("update status request failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("update status failed")
		return
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Error().Int("code", resp.StatusCode).Msg("update status rejected")
	}
}

func (r *Runner) pullImage(image string) error {
	log.Info().Str("image", image).Msg("pulling image")

	cmd := exec.Command("docker", "pull", image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker pull: %w\n%s", err, string(output))
	}

	log.Info().Str("image", image).Msg("image pulled")
	return nil
}

func (r *Runner) runContainer(name string, d Deployment) (string, error) {
	args := []string{"run", "-d", "--name", name, "--restart", "unless-stopped"}

	// Inject SSH authorized key if provided
	sshKey := r.fetchSSHKey()
	if sshKey != "" {
		args = append(args, "-e", fmt.Sprintf("SSH_PUBLIC_KEY=%s", sshKey))
		args = append(args, "-v", "/tmp/aetherius-ssh:/root/.ssh")
	}

	if d.GPURequired > 0 {
		if r.hasNvidiaDocker() {
			args = append(args, "--gpus", fmt.Sprintf("\"device=%d\"", 0))
		} else {
			args = append(args, "--gpus", "all")
		}
	}

	if d.Ports != "" {
		var ports map[string]int
		if err := json.Unmarshal([]byte(d.Ports), &ports); err == nil {
			for containerPort, hostPort := range ports {
				args = append(args, "-p", fmt.Sprintf("%d:%d", hostPort, containerPort))
			}
		}
	}

	var cmdOverride string
	if d.Env != "" {
		var env map[string]string
		if err := json.Unmarshal([]byte(d.Env), &env); err == nil {
			for k, v := range env {
				if k == "CMD" {
					cmdOverride = v
				} else {
					args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
				}
			}
		}
	}

	args = append(args, d.Image)

	if cmdOverride != "" {
		args = append(args, strings.Fields(cmdOverride)...)
	} else {
		// Keep container alive if no explicit command
		args = append(args, "tail", "-f", "/dev/null")
	}

	log.Info().Str("image", d.Image).Strs("args", args).Msg("running container")

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker run: %w\n%s", err, string(output))
	}

	containerID := strings.TrimSpace(string(output))
	return containerID, nil
}

func (r *Runner) isContainerRunning(name string) (bool, error) {
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", name),
		"--format", "{{.ID}}")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

func (r *Runner) stopContainer(name string) error {
	cmd := exec.Command("docker", "stop", "-t", "10", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Try force remove even if stop fails
		exec.Command("docker", "rm", "-f", name).Run()
		return fmt.Errorf("docker stop: %w\n%s", err, string(output))
	}

	exec.Command("docker", "rm", name).Run()
	log.Info().Str("container", name).Msg("container stopped and removed")
	return nil
}

func (r *Runner) fetchSSHKey() string {
	req, err := http.NewRequest("GET", r.serverURL+"/v1/ssh-keys/default", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+r.apiToken)

	resp, err := r.client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		PublicKey string `json:"public_key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}
	return result.PublicKey
}

func (r *Runner) hasNvidiaDocker() bool {
	cmd := exec.Command("docker", "info", "--format", "{{.Runtimes.nvidia.path}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

func (r *Runner) cleanupContainer(containerID string) error {
	cmd := exec.Command("docker", "rm", "-f", containerID)
	return cmd.Run()
}

func readAll(r io.Reader) string {
	data, _ := io.ReadAll(r)
	return string(data)
}
