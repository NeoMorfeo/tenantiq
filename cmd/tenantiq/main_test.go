package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"

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

// testPublisher is a local EventPublisher for the smoke test.
// The smoke test verifies HTTP wiring, not River.
type testPublisher struct{}

func (p *testPublisher) Publish(_ context.Context, _ domain.Event, _ domain.Tenant) error {
	return nil
}

// testValidator is a local TransitionValidator for the smoke test.
type testValidator struct{}

func (v *testValidator) Apply(_ context.Context, current domain.Status, event domain.Event) (domain.Status, error) {
	for _, t := range domain.Transitions {
		if t.Event == event && t.Src == current {
			return t.Dst, nil
		}
	}
	return "", &domain.TransitionError{Event: event, Current: current}
}

// TestSmoke wires the full stack like main() and verifies it responds.
func TestSmoke(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	repo, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("database: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	svc := app.NewTenantService(repo, &testPublisher{}, &testValidator{})

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

// TestRun exercises the real run() function end-to-end: OTel, River, HTTP
// server, and graceful shutdown. It uses stdout OTel exporter and a temp
// database to avoid external dependencies.
func TestRun(t *testing.T) {
	t.Setenv("DATABASE_PATH", t.TempDir()+"/test-run.db")
	t.Setenv("PORT", "19876")
	t.Setenv("OTEL_EXPORTER", "stdout")
	t.Setenv("OTEL_ENVIRONMENT", "test")

	// Discard OTel stdout exporter output during the test.
	origStdout := os.Stdout
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("opening /dev/null: %v", err)
	}
	os.Stdout = devNull
	t.Cleanup(func() {
		os.Stdout = origStdout
		devNull.Close()
	})

	errCh := make(chan error, 1)
	go func() { errCh <- run() }()

	// Wait for the HTTP server to become ready.
	serverURL := "http://localhost:19876"
	ready := false
	for i := 0; i < 50; i++ {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, serverURL+"/api/v1/tenants", nil)
		resp, reqErr := http.DefaultClient.Do(req)
		if reqErr == nil {
			resp.Body.Close()
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		t.Fatal("server did not start within 5 seconds")
	}

	// Verify the API responds correctly.
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, serverURL+"/api/v1/tenants", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/v1/tenants failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Send SIGINT to trigger graceful shutdown.
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("finding process: %v", err)
	}
	if err := proc.Signal(syscall.SIGINT); err != nil {
		t.Fatalf("sending SIGINT: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("run() returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run() did not exit within 10 seconds")
	}
}

// TestRun_InvalidDB verifies run() returns an error for an invalid database path.
func TestRun_InvalidDB(t *testing.T) {
	t.Setenv("DATABASE_PATH", "/nonexistent/path/db.sqlite")
	t.Setenv("PORT", "19877")
	t.Setenv("OTEL_EXPORTER", "stdout")
	t.Setenv("OTEL_ENVIRONMENT", "test")

	// Discard OTel stdout output.
	origStdout := os.Stdout
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("opening /dev/null: %v", err)
	}
	os.Stdout = devNull
	t.Cleanup(func() {
		os.Stdout = origStdout
		devNull.Close()
	})

	if err := run(); err == nil {
		t.Fatal("expected error for invalid database path, got nil")
	}
}
