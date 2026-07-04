package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/pkg/database"
	"github.com/aetherius/platform/services/storage/internal/handler"
	"github.com/aetherius/platform/services/storage/internal/repository"
	mw "github.com/aetherius/platform/pkg/middleware"
	"github.com/aetherius/platform/services/storage/internal/service"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("starting storage service")

	cfg := struct {
		dbHost      string
		dbPort      int
		dbUser      string
		dbPassword  string
		dbName      string
		accessKey   string
		refreshKey  string
		port        string
	}{
		dbHost:     getEnv("DB_HOST", "localhost"),
		dbPort:     getEnvInt("DB_PORT", 5432),
		dbUser:     getEnv("DB_USER", "aetherius"),
		dbPassword: getEnv("DB_PASSWORD", "password"),
		dbName:     getEnv("DB_NAME", "aetherius_auth"),
		accessKey:  getEnv("JWT_ACCESS_KEY", "dev-access-secret-key-change-in-production"),
		refreshKey: getEnv("JWT_REFRESH_KEY", "dev-refresh-secret-key-change-in-production"),
		port:       getEnv("PORT", "8088"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbCfg := database.PostgresConfigWithPort(cfg.dbHost, cfg.dbPort, cfg.dbUser, cfg.dbPassword, cfg.dbName)
	pool, err := database.NewPostgresPool(ctx, dbCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	if err := runMigrations(ctx, pool); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	jwtManager := auth.DefaultJWTManager(cfg.accessKey, cfg.refreshKey)

	storageRepo := repository.NewStorageRepository(pool)
	storageService, err := service.NewStorageService(storageRepo)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize storage service")
	}
	storageHandler := handler.NewStorageHandler(storageService)

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

	storageHandler.RegisterRoutes(r, jwtManager)

	srv := &http.Server{
		Addr:         ":" + cfg.port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.port).Msg("storage service listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down storage service")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
		CREATE TABLE IF NOT EXISTS storage_objects (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			deployment_id UUID,
			bucket TEXT NOT NULL,
			key TEXT NOT NULL,
			filename TEXT NOT NULL,
			content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
			size BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_storage_objects_user ON storage_objects(user_id);
		CREATE INDEX IF NOT EXISTS idx_storage_objects_bucket ON storage_objects(bucket);
	`
	if _, err := pool.Exec(ctx, query); err != nil {
		return err
	}
	log.Info().Msg("database migrations applied")
	return nil
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
