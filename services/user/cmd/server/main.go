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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/pkg/database"
	"github.com/aetherius/platform/services/user/internal/handler"
	"github.com/aetherius/platform/services/user/internal/repository"
	mw "github.com/aetherius/platform/pkg/middleware"
	"github.com/aetherius/platform/services/user/internal/service"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("starting user service")

	dbHost := getEnv("DB_HOST", "localhost")
	dbPortStr := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "aetherius")
	dbPassword := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "aetherius_auth")
	port := getEnv("PORT", "8084")
	accessKey := getEnv("JWT_ACCESS_KEY", "")
	refreshKey := getEnv("JWT_REFRESH_KEY", "")

	dbPort, err := strconv.Atoi(dbPortStr)
	if err != nil {
		log.Fatal().Err(err).Str("db_port", dbPortStr).Msg("invalid DB_PORT")
	}

	ctx := context.Background()

	dbCfg := database.PostgresConfigWithPort(dbHost, dbPort, dbUser, dbPassword, dbName)
	pool, err := database.NewPostgresPool(ctx, dbCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("could not connect to postgres")
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS user_profiles (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL UNIQUE,
			bio TEXT NOT NULL DEFAULT '',
			website TEXT NOT NULL DEFAULT '',
			github_user TEXT NOT NULL DEFAULT '',
			timezone TEXT NOT NULL DEFAULT 'UTC',
			notify_email BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS api_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			name TEXT NOT NULL,
			key_prefix TEXT NOT NULL DEFAULT '',
			key_hash TEXT NOT NULL DEFAULT '',
			last_used TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);
		CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
	`); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}
	log.Info().Msg("database migrations applied")

	jwtManager := auth.DefaultJWTManager(accessKey, refreshKey)

	userRepo := repository.NewUserRepository(pool)
	userSvc := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userSvc)

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

	userHandler.RegisterRoutes(r, jwtManager)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info().Str("port", port).Msg("user service listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down user service")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
