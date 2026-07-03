package service

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/services/monitoring/internal/model"
)

type MonitoringService struct {
	mu            sync.RWMutex
	metrics       *model.MetricsSnapshot
	alerts        []*model.Alert
	alertIDCounter int
	startTime     time.Time
}

func NewMonitoringService() *MonitoringService {
	return &MonitoringService{
		startTime: time.Now(),
	}
}

func (s *MonitoringService) RecordMetrics(snapshot *model.MetricsSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics = snapshot
}

func (s *MonitoringService) GetMetrics() *model.MetricsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.metrics == nil {
		return nil
	}
	cp := *s.metrics
	return &cp
}

func (s *MonitoringService) RecordAlert(alert *model.Alert) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertIDCounter++
	alert.ID = fmt.Sprintf("alert-%d", s.alertIDCounter)
	alert.Timestamp = time.Now()
	s.alerts = append(s.alerts, alert)
}

func (s *MonitoringService) GetAlerts() []*model.Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*model.Alert, len(s.alerts))
	for i, a := range s.alerts {
		cp := *a
		result[i] = &cp
	}
	return result
}

func (s *MonitoringService) AcknowledgeAlert(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range s.alerts {
		if a.ID == id {
			a.Acknowledged = true
			return nil
		}
	}
	return errors.New("alert not found")
}

var serviceEndpoints = []struct {
	Name string
	URL  string
}{
	{"auth", "http://localhost:8081/health"},
	{"node", "http://localhost:8082/health"},
	{"scheduler", "http://localhost:8083/health"},
	{"user", "http://localhost:8084/health"},
	{"deployment", "http://localhost:8085/health"},
	{"marketplace", "http://localhost:8086/health"},
	{"billing", "http://localhost:8087/health"},
	{"storage", "http://localhost:8088/health"},
	{"notification", "http://localhost:8089/health"},
	{"networking", "http://localhost:8090/health"},
	{"ai", "http://localhost:8091/health"},
}

func (s *MonitoringService) CheckServices() *model.SystemHealth {
	var services []model.ServiceStatus
	overallStatus := "healthy"

	client := &http.Client{Timeout: 2 * time.Second}

	for _, ep := range serviceEndpoints {
		start := time.Now()
		resp, err := client.Get(ep.URL)
		latency := time.Since(start).Milliseconds()

		svc := model.ServiceStatus{
			Name:      ep.Name,
			Latency:   latency,
			LastCheck: time.Now(),
		}

		if err != nil {
			svc.Status = "down"
			log.Warn().Str("service", ep.Name).Err(err).Msg("health check failed")
		} else {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				svc.Status = "up"
			} else {
				svc.Status = "degraded"
				log.Warn().Str("service", ep.Name).Int("status", resp.StatusCode).Msg("service returned non-200")
			}
		}

		services = append(services, svc)
	}

	for _, svc := range services {
		if svc.Status == "down" {
			overallStatus = "down"
			break
		}
		if svc.Status == "degraded" {
			overallStatus = "degraded"
		}
	}

	return &model.SystemHealth{
		Status:    overallStatus,
		Uptime:    time.Since(s.startTime).Round(time.Second).String(),
		Version:   "1.0.0",
		Services:  services,
		Timestamp: time.Now(),
	}
}
