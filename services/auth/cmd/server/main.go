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
	"github.com/aetherius/platform/services/auth/internal/handler"
	"github.com/aetherius/platform/services/auth/internal/repository"
	"github.com/aetherius/platform/services/auth/internal/service"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("starting auth service")

	cfg := struct {
		dbHost      string
		dbPort      int
		dbUser      string
		dbPassword  string
		dbName      string
		accessKey   string
		refreshKey  string
		port        string
		issuer      string
	}{
		dbHost:     getEnv("DB_HOST", "localhost"),
		dbPort:     getEnvInt("DB_PORT", 5432),
		dbUser:     getEnv("DB_USER", "aetherius"),
		dbPassword: getEnv("DB_PASSWORD", "password"),
		dbName:     getEnv("DB_NAME", "aetherius_auth"),
		accessKey:  getEnv("JWT_ACCESS_KEY", "dev-access-secret-key-change-in-production"),
		refreshKey: getEnv("JWT_REFRESH_KEY", "dev-refresh-secret-key-change-in-production"),
		port:       getEnv("PORT", "8081"),
		issuer:     getEnv("JWT_ISSUER", "aetherius"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbCfg := database.PostgresConfigWithPort(cfg.dbHost, cfg.dbPort, cfg.dbUser, cfg.dbPassword, cfg.dbName)
	pool, err := database.NewPostgresPool(ctx, dbCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	jwtManager := auth.NewJWTManager(
		cfg.accessKey, cfg.refreshKey,
		15*time.Minute, 7*24*time.Hour, cfg.issuer,
	)

	userRepo := repository.NewUserRepository(pool)
	authService := service.NewAuthService(userRepo, jwtManager)
	authHandler := handler.NewAuthHandler(authService)

	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(30 * time.Second))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Public routes (no auth required)
	authHandler.RegisterRoutes(r, jwtManager)

	srv := &http.Server{
		Addr:         ":" + cfg.port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.port).Msg("auth service listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down auth service")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}
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
