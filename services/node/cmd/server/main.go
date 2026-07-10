package main

import (
	"context"
	_ "embed"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/pkg/database"
	"github.com/aetherius/platform/pkg/queue"
	"github.com/aetherius/platform/services/node/internal/handler"
	"github.com/aetherius/platform/services/node/internal/repository"
	mw "github.com/aetherius/platform/pkg/middleware"
	"github.com/aetherius/platform/services/node/internal/service"
)

//go:embed install.sh
var installScript []byte

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("starting node service")

	cfg := struct {
		dbHost     string
		dbPort     int
		dbUser     string
		dbPassword string
		dbName     string
		rmqURL     string
		accessKey  string
		refreshKey string
		port       string
	}{
		dbHost:     getEnv("DB_HOST", "localhost"),
		dbPort:     getEnvInt("DB_PORT", 5432),
		dbUser:     getEnv("DB_USER", "aetherius"),
		dbPassword: getEnv("DB_PASSWORD", "password"),
		dbName:     getEnv("DB_NAME", "aetherius_node"),
		rmqURL:     getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		accessKey:  getEnv("JWT_ACCESS_KEY", "dev-access-secret-key-change-in-production"),
		refreshKey: getEnv("JWT_REFRESH_KEY", "dev-refresh-secret-key-change-in-production"),
		port:       getEnv("PORT", "8082"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbCfg := database.PostgresConfigWithPort(cfg.dbHost, cfg.dbPort, cfg.dbUser, cfg.dbPassword, cfg.dbName)
	pool, err := database.NewPostgresPool(ctx, dbCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
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

	jwtManager := auth.NewJWTManager(cfg.accessKey, cfg.refreshKey, 15*time.Minute, 7*24*time.Hour, "aetherius")
	authMiddleware := auth.HTTPMiddleware(jwtManager)

	nodeRepo := repository.NewNodeRepository(pool)
	nodeService := service.NewNodeService(nodeRepo, qClient)
	nodeHandler := handler.NewNodeHandler(nodeService, jwtManager)

	r := chi.NewRouter()
	r.Use(mw.CORSMiddleware)
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/install.sh", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/x-shellscript")
		w.WriteHeader(http.StatusOK)
		w.Write(installScript)
	})

	// Public routes (no auth)
	r.Get("/v1/nodes/available", nodeHandler.ListAvailableNodes)

	// Agent-facing routes (no auth — node uses its own token)
	r.Post("/v1/nodes/heartbeat", nodeHandler.Heartbeat)
	r.Post("/v1/nodes/{id}/heartbeat", nodeHandler.SimpleHeartbeat)

	// User-facing routes (authenticated with JWT)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/v1/nodes/register", nodeHandler.RegisterNode)
		r.Get("/v1/nodes", nodeHandler.ListNodes)
		r.Get("/v1/nodes/{id}", nodeHandler.GetNode)
		r.Post("/v1/nodes/{id}/pause", nodeHandler.PauseNode)
		r.Post("/v1/nodes/{id}/resume", nodeHandler.ResumeNode)
		r.Get("/v1/ssh-keys", nodeHandler.ListSSHKeys)
		r.Get("/v1/ssh-keys/default", nodeHandler.GetDefaultSSHKey)
		r.Post("/v1/ssh-keys", nodeHandler.AddSSHKey)
		r.Delete("/v1/ssh-keys/{id}", nodeHandler.DeleteSSHKey)
	})

	// WebSocket terminal (auth via query param)
	r.Get("/v1/workspace/{id}/terminal", nodeHandler.WorkspaceTerminal)

	// Agent-facing deployment routes (no auth — used by running nodes)
	r.Get("/v1/nodes/{id}/deployments", nodeHandler.ListNodeDeployments)
	r.Post("/v1/nodes/{id}/deployments/{deploymentId}/status", nodeHandler.UpdateDeploymentStatus)

	srv := &http.Server{
		Addr:    ":" + cfg.port,
		Handler: r,
	}

	go func() {
		log.Info().Str("port", cfg.port).Msg("node service listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down node service")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
