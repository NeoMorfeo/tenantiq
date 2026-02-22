package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/neomorfeo/tenantiq/internal/adapter/sqlite"
	"github.com/neomorfeo/tenantiq/internal/app"
	"github.com/neomorfeo/tenantiq/internal/domain"

	handler "github.com/neomorfeo/tenantiq/internal/adapter/http"
)

func main() {
	port := envOrDefault("PORT", "8080")
	dbPath := envOrDefault("DATABASE_PATH", "tenantiq.db")

	// --- Adapters (out) ---
	repo, err := sqlite.New(dbPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer repo.Close()

	publisher := &noopPublisher{}

	// --- Application ---
	svc := app.NewTenantService(repo, publisher)

	// --- Adapters (in) ---
	router := chi.NewMux()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)

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
		log.Printf("tenantiq listening on :%s", port)
		log.Printf("API docs: http://localhost:%s/docs", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-done
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	cancel()

	log.Println("stopped")
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

func (p *noopPublisher) Publish(_ context.Context, event domain.Event, tenant domain.Tenant) error {
	log.Printf("event: %s tenant=%s (%s)", event, tenant.ID, tenant.Slug)
	return nil
}
