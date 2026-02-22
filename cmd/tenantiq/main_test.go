package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	handler "github.com/neomorfeo/tenantiq/internal/adapter/http"
	"github.com/neomorfeo/tenantiq/internal/adapter/sqlite"
	"github.com/neomorfeo/tenantiq/internal/app"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

func TestEnvOrDefault_Fallback(t *testing.T) {
	v := envOrDefault("TENANTIQ_TEST_NONEXISTENT_KEY", "fallback")
	if v != "fallback" {
		t.Errorf("got %q, want %q", v, "fallback")
	}
}

func TestEnvOrDefault_EnvSet(t *testing.T) {
	t.Setenv("TENANTIQ_TEST_KEY", "custom")

	v := envOrDefault("TENANTIQ_TEST_KEY", "fallback")
	if v != "custom" {
		t.Errorf("got %q, want %q", v, "custom")
	}
}

func TestNoopPublisher(t *testing.T) {
	p := &noopPublisher{}
	tenant := domain.NewTenant("t-1", "Acme", "acme", "free")

	err := p.Publish(context.Background(), domain.EventProvisionComplete, tenant)
	if err != nil {
		t.Errorf("Publish returned error: %v", err)
	}
}

// TestSmoke wires the full stack like main() and verifies it responds.
func TestSmoke(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	repo, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("database: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	svc := app.NewTenantService(repo, &noopPublisher{})

	router := chi.NewMux()
	api := humachi.New(router, huma.DefaultConfig("tenantiq", "0.1.0"))
	handler.Register(api, svc)

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	// Verify the server responds to list tenants.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/api/v1/tenants", nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/v1/tenants failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var tenants []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&tenants); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(tenants) != 0 {
		t.Errorf("got %d tenants, want 0 (empty database)", len(tenants))
	}
}
