package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/riandyrn/otelchi"

	fsmadapter "github.com/neomorfeo/tenantiq/internal/adapter/fsm"
	handler "github.com/neomorfeo/tenantiq/internal/adapter/http"
	otelsetup "github.com/neomorfeo/tenantiq/internal/adapter/otel"
	riveradapter "github.com/neomorfeo/tenantiq/internal/adapter/river"
	"github.com/neomorfeo/tenantiq/internal/adapter/sqlite"
	"github.com/neomorfeo/tenantiq/internal/app"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	port := envOrDefault("PORT", "8080")
	dbPath := envOrDefault("DATABASE_PATH", "tenantiq.db")

	// --- OpenTelemetry (first, so TracerProvider is available globally) ---
	otelCfg := otelsetup.ConfigFromEnv()
	providers, err := otelsetup.Setup(context.Background(), otelCfg)
	if err != nil {
		return fmt.Errorf("otel: %w", err)
	}

	// --- Adapters (out) ---
	db, err := otelsetup.OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer db.Close()

	sqliteRepo, err := sqlite.NewFromDB(db)
	if err != nil {
		return fmt.Errorf("repository: %w", err)
	}

	// --- River (async job queue) ---
	riverClient, err := riveradapter.Setup(context.Background(), db)
	if err != nil {
		return fmt.Errorf("river: %w", err)
	}
	if err := riverClient.Start(context.Background()); err != nil {
		return fmt.Errorf("river start: %w", err)
	}

	// Wrap adapters with tracing decorators.
	repo := otelsetup.NewTracingRepository(sqliteRepo)
	publisher := otelsetup.NewTracingPublisher(riveradapter.NewPublisher(riverClient))

	// --- Application ---
	validator := fsmadapter.New()
	svc := app.NewTenantService(repo, publisher, validator)

	// --- Adapters (in) ---
	router := chi.NewMux()
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(otelchi.Middleware("tenantiq"))

	api := humachi.New(router, huma.DefaultConfig("tenantiq", "0.1.0"))
	handler.Register(api, svc)

	// --- Server ---
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown.
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("tenantiq listening", "port", port)
		slog.Info("API docs", "url", fmt.Sprintf("http://localhost:%s/docs", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()

	<-done
	slog.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown order: HTTP → River → OTel.
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("http shutdown error", "error", err)
	}

	if err := riverClient.Stop(ctx); err != nil {
		slog.Error("river shutdown error", "error", err)
	}

	if err := providers.Shutdown(ctx); err != nil {
		slog.Error("otel shutdown error", "error", err)
	}

	slog.Info("stopped")
	return nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
