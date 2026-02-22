package otel_test

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	adapter "github.com/neomorfeo/tenantiq/internal/adapter/otel"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// --- Test tracer setup ---

func setupTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	return exporter
}

// --- Mock repository ---

type mockRepo struct {
	tenants map[string]domain.Tenant
	slugs   map[string]domain.Tenant
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		tenants: make(map[string]domain.Tenant),
		slugs:   make(map[string]domain.Tenant),
	}
}

func (m *mockRepo) Create(_ context.Context, t domain.Tenant) error {
	m.tenants[t.ID] = t
	m.slugs[t.Slug] = t
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id string) (domain.Tenant, error) {
	t, ok := m.tenants[id]
	if !ok {
		return domain.Tenant{}, domain.ErrTenantNotFound
	}
	return t, nil
}

func (m *mockRepo) GetBySlug(_ context.Context, slug string) (domain.Tenant, error) {
	t, ok := m.slugs[slug]
	if !ok {
		return domain.Tenant{}, domain.ErrTenantNotFound
	}
	return t, nil
}

func (m *mockRepo) List(_ context.Context, _ domain.ListFilter) ([]domain.Tenant, error) {
	out := make([]domain.Tenant, 0, len(m.tenants))
	for _, t := range m.tenants {
		out = append(out, t)
	}
	return out, nil
}

func (m *mockRepo) Update(_ context.Context, t domain.Tenant) error {
	if _, ok := m.tenants[t.ID]; !ok {
		return domain.ErrTenantNotFound
	}
	m.tenants[t.ID] = t
	m.slugs[t.Slug] = t
	return nil
}

// --- Tests ---

func TestTracingRepository_Create_RecordsSpan(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := newMockRepo()
	repo := adapter.NewTracingRepository(inner)

	tenant := domain.NewTenant("t-1", "Acme", "acme", "free")
	if err := repo.Create(context.Background(), tenant); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if spans[0].Name != "TenantRepository.Create" {
		t.Errorf("span name = %q, want %q", spans[0].Name, "TenantRepository.Create")
	}

	assertAttribute(t, spans[0], "tenant.id", "t-1")
	assertAttribute(t, spans[0], "tenant.slug", "acme")
}

func TestTracingRepository_GetByID_RecordsSpan(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := newMockRepo()
	repo := adapter.NewTracingRepository(inner)

	tenant := domain.NewTenant("t-1", "Acme", "acme", "free")
	inner.tenants["t-1"] = tenant

	got, err := repo.GetByID(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "t-1" {
		t.Errorf("ID = %q, want %q", got.ID, "t-1")
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if spans[0].Name != "TenantRepository.GetByID" {
		t.Errorf("span name = %q, want %q", spans[0].Name, "TenantRepository.GetByID")
	}
}

func TestTracingRepository_GetByID_RecordsError(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := newMockRepo()
	repo := adapter.NewTracingRepository(inner)

	_, err := repo.GetByID(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrTenantNotFound) {
		t.Fatalf("expected ErrTenantNotFound, got %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}

	if spans[0].Status.Code != codes.Error {
		t.Errorf("span status = %v, want %v", spans[0].Status.Code, codes.Error)
	}

	if len(spans[0].Events) == 0 {
		t.Error("expected error event on span")
	}
}

func TestTracingRepository_List_RecordsResultCount(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := newMockRepo()
	repo := adapter.NewTracingRepository(inner)

	inner.tenants["t-1"] = domain.NewTenant("t-1", "A", "a", "free")
	inner.tenants["t-2"] = domain.NewTenant("t-2", "B", "b", "pro")

	tenants, err := repo.List(context.Background(), domain.ListFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tenants) != 2 {
		t.Errorf("got %d tenants, want 2", len(tenants))
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}

	assertAttribute(t, spans[0], "result.count", "2")
}

func TestTracingRepository_Update_RecordsSpan(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := newMockRepo()
	repo := adapter.NewTracingRepository(inner)

	tenant := domain.NewTenant("t-1", "Acme", "acme", "free")
	inner.tenants["t-1"] = tenant

	tenant.Status = domain.StatusActive
	if err := repo.Update(context.Background(), tenant); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if spans[0].Name != "TenantRepository.Update" {
		t.Errorf("span name = %q, want %q", spans[0].Name, "TenantRepository.Update")
	}

	assertAttribute(t, spans[0], "tenant.status", "active")
}

func TestTracingRepository_GetBySlug_RecordsSpan(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := newMockRepo()
	repo := adapter.NewTracingRepository(inner)

	tenant := domain.NewTenant("t-1", "Acme", "acme", "free")
	inner.slugs["acme"] = tenant

	got, err := repo.GetBySlug(context.Background(), "acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "t-1" {
		t.Errorf("ID = %q, want %q", got.ID, "t-1")
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}

	assertAttribute(t, spans[0], "tenant.slug", "acme")
}

// assertAttribute checks that a span has an attribute with the given key and string value.
func assertAttribute(t *testing.T, span tracetest.SpanStub, key, want string) {
	t.Helper()
	for _, attr := range span.Attributes {
		if string(attr.Key) == key {
			got := attr.Value.Emit()
			if got != want {
				t.Errorf("attribute %q = %q, want %q", key, got, want)
			}
			return
		}
	}
	t.Errorf("attribute %q not found on span %q", key, span.Name)
}
