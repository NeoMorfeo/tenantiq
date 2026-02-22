package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/neomorfeo/tenantiq/internal/app"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// --- Mocks ---

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
	m.tenants[t.ID] = t
	m.slugs[t.Slug] = t
	return nil
}

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

// --- Tests ---

func TestCreate_Success(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := app.NewTenantService(repo, pub)

	tenant, err := svc.Create(context.Background(), "Acme", "acme", "free")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tenant.Name != "Acme" {
		t.Errorf("Name = %q, want %q", tenant.Name, "Acme")
	}
	if tenant.Status != domain.StatusCreating {
		t.Errorf("Status = %q, want %q", tenant.Status, domain.StatusCreating)
	}
	if len(tenant.ID) == 0 {
		t.Error("ID should not be empty")
	}

	// Verify it was persisted.
	stored, err := repo.GetByID(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("tenant not found in repo: %v", err)
	}
	if stored.Slug != "acme" {
		t.Errorf("stored Slug = %q, want %q", stored.Slug, "acme")
	}

	// Verify event was published.
	if len(pub.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(pub.events))
	}
	if pub.events[0].event != domain.EventProvisionComplete {
		t.Errorf("event = %q, want %q", pub.events[0].event, domain.EventProvisionComplete)
	}
}

func TestCreate_DuplicateSlug(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := app.NewTenantService(repo, pub)

	if _, err := svc.Create(context.Background(), "Acme", "acme", "free"); err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err := svc.Create(context.Background(), "Acme 2", "acme", "pro")
	var slugErr *domain.SlugConflictError
	if !errors.As(err, &slugErr) {
		t.Fatalf("expected SlugConflictError, got %v", err)
	}
	if slugErr.Slug != "acme" {
		t.Errorf("slug = %q, want %q", slugErr.Slug, "acme")
	}
}

func TestTransition_HappyPath(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := app.NewTenantService(repo, pub)

	tenant, _ := svc.Create(context.Background(), "Acme", "acme", "free")

	// creating → active
	tenant, err := svc.Transition(context.Background(), tenant.ID, domain.EventProvisionComplete)
	if err != nil {
		t.Fatalf("provision_complete failed: %v", err)
	}
	if tenant.Status != domain.StatusActive {
		t.Errorf("Status = %q, want %q", tenant.Status, domain.StatusActive)
	}

	// active → suspended
	tenant, err = svc.Transition(context.Background(), tenant.ID, domain.EventSuspend)
	if err != nil {
		t.Fatalf("suspend failed: %v", err)
	}
	if tenant.Status != domain.StatusSuspended {
		t.Errorf("Status = %q, want %q", tenant.Status, domain.StatusSuspended)
	}

	// suspended → active
	tenant, err = svc.Transition(context.Background(), tenant.ID, domain.EventReactivate)
	if err != nil {
		t.Fatalf("reactivate failed: %v", err)
	}
	if tenant.Status != domain.StatusActive {
		t.Errorf("Status = %q, want %q", tenant.Status, domain.StatusActive)
	}
}

func TestTransition_InvalidEvent(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := app.NewTenantService(repo, pub)

	tenant, _ := svc.Create(context.Background(), "Acme", "acme", "free")

	// Can't suspend from creating.
	_, err := svc.Transition(context.Background(), tenant.ID, domain.EventSuspend)
	var trErr *domain.TransitionError
	if !errors.As(err, &trErr) {
		t.Fatalf("expected TransitionError, got %v", err)
	}
	if trErr.Event != domain.EventSuspend {
		t.Errorf("event = %q, want %q", trErr.Event, domain.EventSuspend)
	}
	if trErr.Current != domain.StatusCreating {
		t.Errorf("current = %q, want %q", trErr.Current, domain.StatusCreating)
	}
}

func TestTransition_NotFound(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := app.NewTenantService(repo, pub)

	_, err := svc.Transition(context.Background(), "nonexistent", domain.EventSuspend)
	if !errors.Is(err, domain.ErrTenantNotFound) {
		t.Errorf("expected ErrTenantNotFound, got %v", err)
	}
}
