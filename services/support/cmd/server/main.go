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
	"github.com/aetherius/platform/services/support/internal/handler"
	"github.com/aetherius/platform/services/support/internal/repository"
	mw "github.com/aetherius/platform/pkg/middleware"
	"github.com/aetherius/platform/services/support/internal/service"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("starting support service")

	dbHost := getEnv("DB_HOST", "localhost")
	dbPortStr := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "aetherius")
	dbPassword := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "aetherius_auth")
	port := getEnv("PORT", "8093")
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

	if err := runMigrations(ctx, pool); err != nil {
		log.Fatal().Err(err).Msg("migration failed")
	}

	jwtManager := auth.DefaultJWTManager(accessKey, refreshKey)

	supportRepo := repository.NewSupportRepository(pool)
	supportSvc := service.NewSupportService(supportRepo)
	supportHandler := handler.NewSupportHandler(supportSvc)

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

	supportHandler.RegisterRoutes(r, jwtManager)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info().Str("port", port).Msg("support service listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down support service")
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

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS support_tickets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			subject TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'open',
			priority TEXT NOT NULL DEFAULT 'medium',
			category TEXT NOT NULL DEFAULT 'general',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS support_messages (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			ticket_id UUID NOT NULL REFERENCES support_tickets(id),
			user_id UUID NOT NULL,
			content TEXT NOT NULL,
			is_staff BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_support_tickets_user ON support_tickets(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_support_tickets_status ON support_tickets(status)`,
		`CREATE INDEX IF NOT EXISTS idx_support_messages_ticket ON support_messages(ticket_id)`,
	}

	for _, m := range migrations {
		if _, err := pool.Exec(ctx, m); err != nil {
			return err
		}
	}
	log.Info().Msg("schema migrations applied")
	return nil
}
