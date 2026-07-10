package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/pkg/database"
	"github.com/aetherius/platform/pkg/queue"
	"github.com/aetherius/platform/services/scheduler/internal/handler"
	"github.com/aetherius/platform/services/scheduler/internal/model"
	"github.com/aetherius/platform/services/scheduler/internal/repository"
	mw "github.com/aetherius/platform/pkg/middleware"
	"github.com/aetherius/platform/services/scheduler/internal/scheduler"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("starting scheduler service")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := struct {
		rmqURL    string
		port      string
		dbHost    string
		dbPort    int
		dbUser    string
		dbPass    string
		dbName    string
		jwtSecret string
	}{
		rmqURL:    getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		port:      getEnv("PORT", "8083"),
		dbHost:    getEnv("DB_HOST", "localhost"),
		dbPort:    getEnvInt("DB_PORT", 5432),
		dbUser:    getEnv("DB_USER", "postgres"),
		dbPass:    getEnv("DB_PASSWORD", "postgres"),
		dbName:    getEnv("DB_NAME", "aetherius"),
		jwtSecret: getEnv("JWT_SECRET", "dev-secret-change-in-production"),
	}

	pgCfg := database.PostgresConfigWithPort(cfg.dbHost, cfg.dbPort, cfg.dbUser, cfg.dbPass, cfg.dbName)
	pool, err := database.NewPostgresPool(ctx, pgCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}
	defer pool.Close()

	qClient := queue.NewClient(cfg.rmqURL)
	if err := qClient.Connect(); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to rabbitmq")
	}
	defer qClient.Close()

	if err := queue.DeclareStandardExchanges(qClient); err != nil {
		log.Fatal().Err(err).Msg("failed to declare exchanges")
	}

	repo := repository.NewSchedulerRepository(pool)
	sched := scheduler.NewScheduler()
	jwtManager := auth.DefaultJWTManager(cfg.jwtSecret, cfg.jwtSecret)
	h := handler.NewSchedulerHandler(repo, sched)

	heartbeatQueue, err := qClient.DeclareQueue("scheduler.node.heartbeat", amqp091.Table{
		"x-dead-letter-exchange": queue.ExchangeDeadLetter,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to declare queue")
	}

	qClient.BindQueue(heartbeatQueue.Name, queue.RoutingKeyNodeHeartbeat, queue.ExchangeDomainEvents)

	heartbeatTag := "scheduler-hb-" + randomID(6)
	heartbeatMsgs, err := qClient.Consume(heartbeatQueue.Name, heartbeatTag, false)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to consume heartbeats")
	}

	offlineQueue, err := qClient.DeclareQueue("scheduler.node.offline", nil)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to declare offline queue")
	}
	qClient.BindQueue(offlineQueue.Name, queue.RoutingKeyNodeOffline, queue.ExchangeDeadLetter)

	offlineTag := "scheduler-off-" + randomID(6)
	offlineMsgs, err := qClient.Consume(offlineQueue.Name, offlineTag, false)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to consume offline events")
	}

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
					sched.Enqueue(&scheduler.DeploymentRequest{
						ID: result.DeploymentID,
					})
				}
			}
			msg.Ack(false)
		}
	}()

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		for range ticker.C {

			nodes, err := repo.ListActiveNodes(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to list active nodes")
			} else {
				sched.UpdateNodes(nodes)
			}

			pending, err := repo.ListPendingDeployments(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to list pending deployments")
				continue
			}

			for _, d := range pending {
				sched.Enqueue(&scheduler.DeploymentRequest{
					ID:     d.ID,
					UserID: d.UserID,
					Resources: scheduler.ResourceRequest{
						GPUCount:    d.GPURequired,
						VRAMBytes:   d.VRAMRequiredGB * 1024 * 1024 * 1024,
						RAMBytes:    d.RAMRequiredGB * 1024 * 1024 * 1024,
						DiskBytes:   d.DiskRequiredGB * 1024 * 1024 * 1024,
					},
					Placement: scheduler.PlacementPreference{
						Region: d.Region,
					},
					CreatedAt: d.CreatedAt,
				})
			}

			results := sched.Schedule(ctx)
			for _, result := range results {
				if result.Error == "" {
					if err := repo.AssignNode(ctx, result.DeploymentID, result.NodeID); err != nil {
						log.Error().Err(err).Msg("failed to assign node")
						continue
					}

					log.Info().
						Str("deployment_id", result.DeploymentID.String()).
						Str("node_id", result.NodeID.String()).
						Float64("score", result.Score).
						Int64("cost_cents", result.EstimatedCostCents).
						Msg("deployment scheduled")

					payload, _ := json.Marshal(result)
					qClient.Publish(ctx, queue.ExchangeDomainEvents,
						queue.RoutingKeyDeploymentScheduled, queue.Event{
							Type:    "deployment.scheduled",
							Source:  "scheduler-service",
							Payload: payload,
						})
				} else {
					log.Warn().
						Str("deployment_id", result.DeploymentID.String()).
						Str("error", result.Error).
						Msg("deployment scheduling failed")
					repo.UpdateStatus(ctx, result.DeploymentID, model.DeployStatusFailed)
				}

				sched.Complete(result.DeploymentID, result)
			}
		}
	}()

	r := chi.NewRouter()
	r.Use(mw.CORSMiddleware)
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sched.GetStats())
	})

	h.RegisterRoutes(r, jwtManager)

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

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return fallback
}

func randomID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
