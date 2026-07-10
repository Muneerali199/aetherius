package terminal

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Server struct {
	port     int
	server   *http.Server
}

func NewServer(port int) *Server {
	return &Server{port: port}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/terminal/", s.handleTerminal)

	addr := fmt.Sprintf(":%d", s.port)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("terminal server listen: %w", err)
	}

	go func() {
		log.Info().Str("addr", addr).Msg("terminal server listening")
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("terminal server error")
		}
	}()

	return nil
}

func (s *Server) Stop() {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *Server) handleTerminal(w http.ResponseWriter, r *http.Request) {
	containerID := strings.TrimPrefix(r.URL.Path, "/terminal/")
	if containerID == "" {
		http.Error(w, "missing container id", http.StatusBadRequest)
		return
	}

	// Attempt websocket upgrade
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Str("container", containerID).Msg("websocket upgrade failed")
		return
	}
	defer conn.Close()

	// Verify container exists
	check := exec.Command("docker", "inspect", "--format", "{{.State.Status}}", containerID)
	status, err := check.Output()
	if err != nil || strings.TrimSpace(string(status)) != "running" {
		conn.WriteJSON(map[string]string{"error": "container not running"})
		return
	}

	// Create docker exec
	cmd := exec.Command("docker", "exec", "-i", containerID, "/bin/sh", "-c", "TERM=xterm-256color /bin/bash -l")
	if err := cmd.Start(); err != nil {
		conn.WriteJSON(map[string]string{"error": "failed to start shell"})
		return
	}

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		conn.WriteJSON(map[string]string{"error": "failed to start shell"})
		return
	}

	done := make(chan struct{})

	// Container → WebSocket: stdout
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				conn.WriteMessage(websocket.BinaryMessage, buf[:n])
			}
			if err != nil {
				close(done)
				return
			}
		}
	}()

	// Container → WebSocket: stderr
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				conn.WriteMessage(websocket.BinaryMessage, buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	// WebSocket → Container: stdin
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				stdin.Close()
				return
			}
			stdin.Write(msg)
		}
	}()

	<-done
	cmd.Wait()
}

func (s *Server) URL() string {
	return fmt.Sprintf("http://localhost:%d", s.port)
}

func (s *Server) AgentURL(host string) string {
	return fmt.Sprintf("http://%s:%d", host, s.port)
}

// SendExec runs a one-shot command via HTTP (for simple use cases)
func SendExec(agentURL, containerID, command string) (string, error) {
	payload, _ := json.Marshal(map[string]string{
		"container": containerID,
		"command":   command,
	})

	resp, err := http.Post(agentURL+"/exec", "application/json", strings.NewReader(string(payload)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}