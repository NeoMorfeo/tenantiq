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

	handler "github.com/neomorfeo/tenantiq/internal/adapter/http"
	otelsetup "github.com/neomorfeo/tenantiq/internal/adapter/otel"
	"github.com/neomorfeo/tenantiq/internal/adapter/sqlite"
	"github.com/neomorfeo/tenantiq/internal/app"
	"github.com/neomorfeo/tenantiq/internal/domain"
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

	// Wrap adapters with tracing decorators.
	repo := otelsetup.NewTracingRepository(sqliteRepo)
	publisher := otelsetup.NewTracingPublisher(&noopPublisher{})

	// --- Application ---
	svc := app.NewTenantService(repo, publisher)

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

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("http shutdown error", "error", err)
	}

	// Flush OTel providers (ensure all spans/metrics are exported).
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

// noopPublisher is a temporary EventPublisher that does nothing.
// Will be replaced by River when we implement the async task queue.
type noopPublisher struct{}

func (p *noopPublisher) Publish(ctx context.Context, event domain.Event, tenant domain.Tenant) error {
	slog.InfoContext(ctx, "event published",
		"event", string(event),
		"tenant_id", tenant.ID,
		"tenant_slug", tenant.Slug,
	)
	return nil
}
