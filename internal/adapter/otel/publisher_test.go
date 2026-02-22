package otel_test

import (
	"context"
	"fmt"
	"testing"

	"go.opentelemetry.io/otel/codes"

	adapter "github.com/neomorfeo/tenantiq/internal/adapter/otel"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// --- Mock publisher ---

type mockPublisher struct {
	events []publishedEvent
}

type publishedEvent struct {
	event  domain.Event
	tenant domain.Tenant
}

func (m *mockPublisher) Publish(_ context.Context, e domain.Event, t domain.Tenant) error {
	m.events = append(m.events, publishedEvent{event: e, tenant: t})
	return nil
}

type failingPublisher struct{}

func (p *failingPublisher) Publish(_ context.Context, _ domain.Event, _ domain.Tenant) error {
	return fmt.Errorf("publish failed")
}

// --- Tests ---

func TestTracingPublisher_Publish_RecordsSpan(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := &mockPublisher{}
	pub := adapter.NewTracingPublisher(inner)

	tenant := domain.NewTenant("t-1", "Acme", "acme", "free")
	if err := pub.Publish(context.Background(), domain.EventProvisionComplete, tenant); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if spans[0].Name != "EventPublisher.Publish" {
		t.Errorf("span name = %q, want %q", spans[0].Name, "EventPublisher.Publish")
	}

	assertAttribute(t, spans[0], "event.type", "provision_complete")
	assertAttribute(t, spans[0], "tenant.id", "t-1")
	assertAttribute(t, spans[0], "tenant.slug", "acme")

	if len(inner.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(inner.events))
	}
}

func TestTracingPublisher_Publish_RecordsError(t *testing.T) {
	exporter := setupTestTracer(t)
	pub := adapter.NewTracingPublisher(&failingPublisher{})

	tenant := domain.NewTenant("t-1", "Acme", "acme", "free")
	err := pub.Publish(context.Background(), domain.EventProvisionComplete, tenant)
	if err == nil {
		t.Fatal("expected error")
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}

	if spans[0].Status.Code != codes.Error {
		t.Errorf("span status = %v, want %v", spans[0].Status.Code, codes.Error)
	}
}
