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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	bcryptadapter "github.com/neomorfeo/tenantiq/internal/adapter/bcrypt"
	fsmadapter "github.com/neomorfeo/tenantiq/internal/adapter/fsm"
	handler "github.com/neomorfeo/tenantiq/internal/adapter/http"
	jwtadapter "github.com/neomorfeo/tenantiq/internal/adapter/jwt"
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
	jwtSecret := envOrDefault("TENANTIQ_JWT_SECRET", "")
	adminEmail := envOrDefault("TENANTIQ_ADMIN_EMAIL", "admin@tenantiq.local")

	if jwtSecret == "" {
		slog.Warn("TENANTIQ_JWT_SECRET not set, using insecure default (DO NOT use in production)")
		jwtSecret = "tenantiq-dev-secret-change-me-in-production!!"
	}

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

	// --- Auth adapters ---
	sqliteUserRepo := sqlite.NewUserRepository(sqliteRepo.DB())
	userRepo := otelsetup.NewTracingUserRepository(sqliteUserRepo)
	hasher := bcryptadapter.New()
	tokens := jwtadapter.New([]byte(jwtSecret), 15*time.Minute, 7*24*time.Hour)

	// --- Bootstrap (create first superadmin if no users exist) ---
	result, err := app.Bootstrap(context.Background(), userRepo, hasher, adminEmail)
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	if result.Created {
		slog.Info("initial superadmin created",
			"email", result.Email,
			"password", result.Password,
		)
		slog.Warn("save this password — it will not be shown again")
	}

	// --- Application ---
	validator := fsmadapter.New()
	tenantSvc := app.NewTenantService(repo, publisher, validator)
	authSvc := app.NewAuthService(userRepo, hasher, tokens)
	userSvc := app.NewUserService(userRepo, hasher)

	// --- Router ---
	router := chi.NewMux()
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)

	// Single Huma API: one OpenAPI spec, one /docs UI.
	humaConfig := huma.DefaultConfig("tenantiq", "0.1.0")
	humaConfig.DocsRenderer = huma.DocsRendererScalar

	// Disable docs in production (TENANTIQ_ENV=production).
	if envOrDefault("TENANTIQ_ENV", "development") == "production" {
		humaConfig.DocsPath = ""
		humaConfig.OpenAPIPath = ""
	}

	// Register Bearer auth security scheme so Scalar UI shows an Authorize button.
	humaConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
			Description:  "JWT access token obtained from POST /api/v1/auth/login",
		},
	}

	api := humachi.New(router, humaConfig)

	// Huma middleware enforces auth based on each operation's Security field.
	api.UseMiddleware(handler.NewHumaAuthMiddleware(api, tokens))

	// Register all routes on the single API.
	handler.RegisterAuth(api, authSvc)           // Public (no Security field)
	handler.Register(api, tenantSvc)             // Bearer required
	handler.RegisterUsers(api, userSvc)          // Superadmin required
	handler.RegisterPasswordUpdate(api, userSvc) // Bearer required (self-only in handler)

	// --- Server ---
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           otelhttp.NewHandler(router, "tenantiq"),
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
