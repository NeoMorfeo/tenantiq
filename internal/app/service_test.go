package app_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/neomorfeo/tenantiq/internal/app"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// --- Mocks ---

type mockRepo struct {
	tenants   map[string]domain.Tenant
	slugs     map[string]domain.Tenant
	createErr error
	updateErr error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		tenants: make(map[string]domain.Tenant),
		slugs:   make(map[string]domain.Tenant),
	}
}

func (m *mockRepo) Create(_ context.Context, t domain.Tenant) error {
	if m.createErr != nil {
		return m.createErr
	}
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
	if m.updateErr != nil {
		return m.updateErr
	}
	m.tenants[t.ID] = t
	m.slugs[t.Slug] = t
	return nil
}

type mockPublisher struct {
	events     []publishedEvent
	publishErr error
}

type publishedEvent struct {
	event  domain.Event
	tenant domain.Tenant
}

func (m *mockPublisher) Publish(_ context.Context, e domain.Event, t domain.Tenant) error {
	if m.publishErr != nil {
		return m.publishErr
	}
	m.events = append(m.events, publishedEvent{event: e, tenant: t})
	return nil
}

// mockValidator implements domain.TransitionValidator using the same
// domain.Transitions table. This keeps service tests independent of looplab/fsm.
type mockValidator struct{}

func (m *mockValidator) Apply(_ context.Context, current domain.Status, event domain.Event) (domain.Status, error) {
	for _, t := range domain.Transitions {
		if t.Event == event && t.Src == current {
			return t.Dst, nil
		}
	}
	return "", &domain.TransitionError{Event: event, Current: current}
}

// --- Tests ---

func TestCreate_Success(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := app.NewTenantService(repo, pub, &mockValidator{})

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
	svc := app.NewTenantService(repo, pub, &mockValidator{})

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
	svc := app.NewTenantService(repo, pub, &mockValidator{})

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
	svc := app.NewTenantService(repo, pub, &mockValidator{})

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
	svc := app.NewTenantService(repo, pub, &mockValidator{})

	_, err := svc.Transition(context.Background(), "nonexistent", domain.EventSuspend)
	if !errors.Is(err, domain.ErrTenantNotFound) {
		t.Errorf("expected ErrTenantNotFound, got %v", err)
	}
}

// --- GetByID ---

func TestGetByID_Success(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := app.NewTenantService(repo, pub, &mockValidator{})

	created, _ := svc.Create(context.Background(), "Acme", "acme", "free")

	got, err := svc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := app.NewTenantService(repo, pub, &mockValidator{})

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrTenantNotFound) {
		t.Errorf("expected ErrTenantNotFound, got %v", err)
	}
}

// --- List ---

func TestList_Success(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := app.NewTenantService(repo, pub, &mockValidator{})

	if _, err := svc.Create(context.Background(), "Acme", "acme", "free"); err != nil {
		t.Fatalf("create Acme: %v", err)
	}
	if _, err := svc.Create(context.Background(), "Globex", "globex", "pro"); err != nil {
		t.Fatalf("create Globex: %v", err)
	}

	tenants, err := svc.List(context.Background(), domain.ListFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tenants) != 2 {
		t.Errorf("got %d tenants, want 2", len(tenants))
	}
}

// --- Error paths ---

func TestCreate_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.createErr = fmt.Errorf("disk full")
	pub := &mockPublisher{}
	svc := app.NewTenantService(repo, pub, &mockValidator{})

	_, err := svc.Create(context.Background(), "Acme", "acme", "free")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "creating tenant") {
		t.Errorf("error = %q, want it to contain 'creating tenant'", err)
	}
}

func TestCreate_PublishError(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := app.NewTenantService(repo, pub, &mockValidator{})

	// Set publish error after service is wired, before calling Create.
	pub.publishErr = fmt.Errorf("queue down")

	_, err := svc.Create(context.Background(), "Acme", "acme", "free")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "publishing creation event") {
		t.Errorf("error = %q, want it to contain 'publishing creation event'", err)
	}
}

func TestTransition_UpdateError(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := app.NewTenantService(repo, pub, &mockValidator{})

	tenant, _ := svc.Create(context.Background(), "Acme", "acme", "free")

	repo.updateErr = fmt.Errorf("db locked")

	_, err := svc.Transition(context.Background(), tenant.ID, domain.EventProvisionComplete)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "updating tenant") {
		t.Errorf("error = %q, want it to contain 'updating tenant'", err)
	}
}

func TestTransition_PublishError(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := app.NewTenantService(repo, pub, &mockValidator{})

	tenant, _ := svc.Create(context.Background(), "Acme", "acme", "free")

	// Set publish error after Create succeeds.
	pub.publishErr = fmt.Errorf("queue down")

	_, err := svc.Transition(context.Background(), tenant.ID, domain.EventProvisionComplete)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "publishing event") {
		t.Errorf("error = %q, want it to contain 'publishing event'", err)
	}
}
