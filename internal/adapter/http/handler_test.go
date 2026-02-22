package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	adapter "github.com/neomorfeo/tenantiq/internal/adapter/http"
	"github.com/neomorfeo/tenantiq/internal/adapter/sqlite"
	"github.com/neomorfeo/tenantiq/internal/app"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// noopPublisher is a no-op EventPublisher for tests.
type noopPublisher struct{}

func (p *noopPublisher) Publish(_ context.Context, _ domain.Event, _ domain.Tenant) error {
	return nil
}

// newTestServer creates a full-stack httptest.Server with SQLite in-memory.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	repo, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("creating test repo: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	svc := app.NewTenantService(repo, &noopPublisher{})

	router := chi.NewMux()
	api := humachi.New(router, huma.DefaultConfig("tenantiq", "0.1.0"))
	adapter.Register(api, svc)

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return srv
}

// doRequest performs an HTTP request with context (avoids noctx linter).
func doRequest(t *testing.T, method, url, body string) *http.Response {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, reader)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, url, err)
	}

	return resp
}

// mustCreateTenant creates a tenant via the API and returns its response.
func mustCreateTenant(t *testing.T, srv *httptest.Server, name, slug, plan string) adapter.TenantResponse {
	t.Helper()

	body := fmt.Sprintf(`{"name":%q,"slug":%q,"plan":%q}`, name, slug, plan)
	resp := doRequest(t, http.MethodPost, srv.URL+"/api/v1/tenants", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create tenant: status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var tenant adapter.TenantResponse
	if err := json.NewDecoder(resp.Body).Decode(&tenant); err != nil {
		t.Fatalf("decode tenant: %v", err)
	}

	return tenant
}

// --- Create ---

func TestCreate(t *testing.T) {
	srv := newTestServer(t)
	tenant := mustCreateTenant(t, srv, "Acme Corp", "acme-corp", "pro")

	if tenant.ID == "" {
		t.Error("ID should not be empty")
	}
	if tenant.Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", tenant.Name, "Acme Corp")
	}
	if tenant.Slug != "acme-corp" {
		t.Errorf("Slug = %q, want %q", tenant.Slug, "acme-corp")
	}
	if tenant.Plan != "pro" {
		t.Errorf("Plan = %q, want %q", tenant.Plan, "pro")
	}
	if tenant.Status != "creating" {
		t.Errorf("Status = %q, want %q", tenant.Status, "creating")
	}
	if tenant.CreatedAt == "" {
		t.Error("CreatedAt should not be empty")
	}
}

func TestCreate_DefaultPlan(t *testing.T) {
	srv := newTestServer(t)

	resp := doRequest(t, http.MethodPost, srv.URL+"/api/v1/tenants", `{"name":"Acme","slug":"acme"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var tenant adapter.TenantResponse
	if err := json.NewDecoder(resp.Body).Decode(&tenant); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if tenant.Plan != "free" {
		t.Errorf("Plan = %q, want %q", tenant.Plan, "free")
	}
}

func TestCreate_DuplicateSlug(t *testing.T) {
	srv := newTestServer(t)
	mustCreateTenant(t, srv, "Acme", "acme", "free")

	resp := doRequest(t, http.MethodPost, srv.URL+"/api/v1/tenants", `{"name":"Acme 2","slug":"acme","plan":"pro"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestCreate_InvalidSlug(t *testing.T) {
	srv := newTestServer(t)

	resp := doRequest(t, http.MethodPost, srv.URL+"/api/v1/tenants", `{"name":"Acme","slug":"INVALID SLUG!","plan":"free"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnprocessableEntity)
	}
}

func TestCreate_MissingName(t *testing.T) {
	srv := newTestServer(t)

	resp := doRequest(t, http.MethodPost, srv.URL+"/api/v1/tenants", `{"slug":"acme","plan":"free"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnprocessableEntity)
	}
}

// --- Get ---

func TestGet(t *testing.T) {
	srv := newTestServer(t)
	created := mustCreateTenant(t, srv, "Acme", "acme", "pro")

	resp := doRequest(t, http.MethodGet, srv.URL+"/api/v1/tenants/"+created.ID, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var tenant adapter.TenantResponse
	if err := json.NewDecoder(resp.Body).Decode(&tenant); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if tenant.ID != created.ID {
		t.Errorf("ID = %q, want %q", tenant.ID, created.ID)
	}
	if tenant.Name != "Acme" {
		t.Errorf("Name = %q, want %q", tenant.Name, "Acme")
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := newTestServer(t)

	resp := doRequest(t, http.MethodGet, srv.URL+"/api/v1/tenants/nonexistent", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- List ---

func TestList(t *testing.T) {
	srv := newTestServer(t)
	mustCreateTenant(t, srv, "Acme", "acme", "free")
	mustCreateTenant(t, srv, "Globex", "globex", "pro")

	resp := doRequest(t, http.MethodGet, srv.URL+"/api/v1/tenants", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var tenants []adapter.TenantResponse
	if err := json.NewDecoder(resp.Body).Decode(&tenants); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(tenants) != 2 {
		t.Errorf("got %d tenants, want 2", len(tenants))
	}
}

func TestList_FilterByStatus(t *testing.T) {
	srv := newTestServer(t)
	created := mustCreateTenant(t, srv, "Acme", "acme", "free")
	mustCreateTenant(t, srv, "Globex", "globex", "pro")

	// Transition first tenant to active.
	resp := doRequest(t, http.MethodPost, srv.URL+"/api/v1/tenants/"+created.ID+"/events", `{"event":"provision_complete"}`)
	resp.Body.Close()

	// List only active tenants.
	resp = doRequest(t, http.MethodGet, srv.URL+"/api/v1/tenants?status=active", "")
	defer resp.Body.Close()

	var tenants []adapter.TenantResponse
	if err := json.NewDecoder(resp.Body).Decode(&tenants); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(tenants) != 1 {
		t.Fatalf("got %d tenants, want 1", len(tenants))
	}
	if tenants[0].Status != "active" {
		t.Errorf("Status = %q, want %q", tenants[0].Status, "active")
	}
}

// --- Transition ---

func TestTransition(t *testing.T) {
	srv := newTestServer(t)
	created := mustCreateTenant(t, srv, "Acme", "acme", "free")

	resp := doRequest(t, http.MethodPost, srv.URL+"/api/v1/tenants/"+created.ID+"/events", `{"event":"provision_complete"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var tenant adapter.TenantResponse
	if err := json.NewDecoder(resp.Body).Decode(&tenant); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if tenant.Status != "active" {
		t.Errorf("Status = %q, want %q", tenant.Status, "active")
	}
}

func TestTransition_InvalidEvent(t *testing.T) {
	srv := newTestServer(t)
	created := mustCreateTenant(t, srv, "Acme", "acme", "free")

	// "suspend" is not valid from "creating" state.
	resp := doRequest(t, http.MethodPost, srv.URL+"/api/v1/tenants/"+created.ID+"/events", `{"event":"suspend"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnprocessableEntity)
	}
}

func TestTransition_NotFound(t *testing.T) {
	srv := newTestServer(t)

	resp := doRequest(t, http.MethodPost, srv.URL+"/api/v1/tenants/nonexistent/events", `{"event":"provision_complete"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestTransition_InvalidEventValue(t *testing.T) {
	srv := newTestServer(t)
	created := mustCreateTenant(t, srv, "Acme", "acme", "free")

	// "bogus" is not in the enum.
	resp := doRequest(t, http.MethodPost, srv.URL+"/api/v1/tenants/"+created.ID+"/events", `{"event":"bogus"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnprocessableEntity)
	}
}
