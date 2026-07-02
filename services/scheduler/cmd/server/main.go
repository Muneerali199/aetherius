package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/queue"
	"github.com/aetherius/platform/services/scheduler/internal/scheduler"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("starting scheduler service")

	cfg := struct {
		rmqURL string
		port   string
	}{
		rmqURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		port:   getEnv("PORT", "8083"),
	}

	qClient := queue.NewClient(cfg.rmqURL)
	if err := qClient.Connect(); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to rabbitmq")
	}
	defer qClient.Close()

	if err := queue.DeclareStandardExchanges(qClient); err != nil {
		log.Fatal().Err(err).Msg("failed to declare exchanges")
	}

	sched := scheduler.NewScheduler()

	// Listen for node heartbeat events
	heartbeatQueue, err := qClient.DeclareQueue("scheduler.node.heartbeat", amqp091.Table{
		"x-dead-letter-exchange": queue.ExchangeDeadLetter,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to declare queue")
	}

	qClient.BindQueue(heartbeatQueue.Name, queue.RoutingKeyNodeHeartbeat, queue.ExchangeDomainEvents)

	heartbeatMsgs, err := qClient.Consume(heartbeatQueue.Name, "scheduler", false)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to consume heartbeats")
	}

	// Listen for offline nodes
	offlineQueue, err := qClient.DeclareQueue("scheduler.node.offline", nil)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to declare offline queue")
	}
	qClient.BindQueue(offlineQueue.Name, queue.RoutingKeyNodeOffline, queue.ExchangeDeadLetter)

	offlineMsgs, err := qClient.Consume(offlineQueue.Name, "scheduler", false)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to consume offline events")
	}

	// Process events in background
	go func() {
		for msg := range heartbeatMsgs {
			var event struct {
				NodeID string `json:"node_id"`
			}
			json.Unmarshal(msg.Body, &event)
			msg.Ack(false)
		}
	}()

	go func() {
		for msg := range offlineMsgs {
			var event struct {
				NodeID string `json:"node_id"`
			}
			json.Unmarshal(msg.Body, &event)
			if nodeID, err := uuid.Parse(event.NodeID); err == nil {
				affected := sched.HandleNodeOffline(nodeID)
				for _, result := range affected {
					// Re-enqueue affected deployments
					sched.Enqueue(&scheduler.DeploymentRequest{
						ID: result.DeploymentID,
					})
				}
			}
			msg.Ack(false)
		}
	}()

	// Periodic scheduling loop
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		for range ticker.C {
			results := sched.Schedule(context.Background())
			for _, result := range results {
				if result.Error == "" {
					log.Info().
						Str("deployment_id", result.DeploymentID.String()).
						Str("node_id", result.NodeID.String()).
						Float64("score", result.Score).
						Int64("cost_cents", result.EstimatedCostCents).
						Msg("deployment scheduled")

					// Publish deployment.scheduled event
					payload, _ := json.Marshal(result)
					qClient.Publish(context.Background(), queue.ExchangeDomainEvents,
						queue.RoutingKeyDeploymentScheduled, queue.Event{
							Type:    "deployment.scheduled",
							Source:  "scheduler-service",
							Payload: payload,
						})
				}
			}
		}
	}()

	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sched.GetStats())
	})

	srv := &http.Server{
		Addr:    ":" + cfg.port,
		Handler: r,
	}

	go func() {
		log.Info().Str("port", cfg.port).Msg("scheduler service listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down scheduler service")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
