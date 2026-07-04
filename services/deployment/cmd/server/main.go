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
	"github.com/aetherius/platform/services/deployment/internal/handler"
	"github.com/aetherius/platform/services/deployment/internal/repository"
	mw "github.com/aetherius/platform/pkg/middleware"
	"github.com/aetherius/platform/services/deployment/internal/service"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("starting deployment service")

	port := getEnv("PORT", "8085")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPortStr := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "deployments")
	accessKey := getEnv("JWT_ACCESS_KEY", "")
	refreshKey := getEnv("JWT_REFRESH_KEY", "")

	dbPort, err := strconv.Atoi(dbPortStr)
	if err != nil {
		log.Fatal().Err(err).Str("DB_PORT", dbPortStr).Msg("invalid DB_PORT")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pgCfg := database.PostgresConfigWithPort(dbHost, dbPort, dbUser, dbPassword, dbName)
	pool, err := database.NewPostgresPool(ctx, pgCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}
	defer pool.Close()

	jwtManager := auth.DefaultJWTManager(accessKey, refreshKey)

	deploymentRepo := repository.NewDeploymentRepo(pool)
	deploymentSvc := service.NewDeploymentService(deploymentRepo)
	deploymentHandler := handler.NewDeploymentHandler(deploymentSvc)

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

	r.Group(func(r chi.Router) {
		r.Use(auth.HTTPMiddleware(jwtManager))
		deploymentHandler.RegisterRoutes(r)
	})

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info().Str("port", port).Msg("deployment service listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down deployment service")
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
