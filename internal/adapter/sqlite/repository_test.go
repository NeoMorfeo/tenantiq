package sqlite_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/neomorfeo/tenantiq/internal/adapter/sqlite"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// newTestRepo creates an in-memory SQLite repository for testing.
func newTestRepo(t *testing.T) *sqlite.TenantRepository {
	t.Helper()
	repo, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("creating test repo: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

func mustCreate(t *testing.T, repo *sqlite.TenantRepository, tenant domain.Tenant) {
	t.Helper()
	if err := repo.Create(context.Background(), tenant); err != nil {
		t.Fatalf("mustCreate failed: %v", err)
	}
}

func mustUpdate(t *testing.T, repo *sqlite.TenantRepository, tenant domain.Tenant) {
	t.Helper()
	if err := repo.Update(context.Background(), tenant); err != nil {
		t.Fatalf("mustUpdate failed: %v", err)
	}
}

func TestCreate_And_GetByID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	tenant := domain.NewTenant("t-1", "Acme Corp", "acme-corp", "pro")

	if err := repo.Create(ctx, tenant); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.GetByID(ctx, "t-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ID != "t-1" {
		t.Errorf("ID = %q, want %q", got.ID, "t-1")
	}
	if got.Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", got.Name, "Acme Corp")
	}
	if got.Slug != "acme-corp" {
		t.Errorf("Slug = %q, want %q", got.Slug, "acme-corp")
	}
	if got.Status != domain.StatusCreating {
		t.Errorf("Status = %q, want %q", got.Status, domain.StatusCreating)
	}
	if got.Plan != "pro" {
		t.Errorf("Plan = %q, want %q", got.Plan, "pro")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.GetByID(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrTenantNotFound) {
		t.Errorf("expected ErrTenantNotFound, got %v", err)
	}
}

func TestGetBySlug(t *testing.T) {
	repo := newTestRepo(t)

	tenant := domain.NewTenant("t-1", "Acme", "acme", "free")
	mustCreate(t, repo, tenant)

	got, err := repo.GetBySlug(context.Background(), "acme")
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if got.ID != "t-1" {
		t.Errorf("ID = %q, want %q", got.ID, "t-1")
	}
}

func TestGetBySlug_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.GetBySlug(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrTenantNotFound) {
		t.Errorf("expected ErrTenantNotFound, got %v", err)
	}
}

func TestCreate_DuplicateSlug(t *testing.T) {
	repo := newTestRepo(t)

	t1 := domain.NewTenant("t-1", "Acme", "acme", "free")
	t2 := domain.NewTenant("t-2", "Acme 2", "acme", "pro")

	mustCreate(t, repo, t1)
	err := repo.Create(context.Background(), t2)

	var slugErr *domain.SlugConflictError
	if !errors.As(err, &slugErr) {
		t.Fatalf("expected SlugConflictError, got %v", err)
	}
	if slugErr.Slug != "acme" {
		t.Errorf("slug = %q, want %q", slugErr.Slug, "acme")
	}
}

func TestUpdate(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	tenant := domain.NewTenant("t-1", "Acme", "acme", "free")
	mustCreate(t, repo, tenant)

	tenant.Status = domain.StatusActive
	tenant.Name = "Acme Updated"

	if err := repo.Update(ctx, tenant); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := repo.GetByID(ctx, "t-1")
	if got.Status != domain.StatusActive {
		t.Errorf("Status = %q, want %q", got.Status, domain.StatusActive)
	}
	if got.Name != "Acme Updated" {
		t.Errorf("Name = %q, want %q", got.Name, "Acme Updated")
	}
	if got.UpdatedAt.Before(got.CreatedAt) {
		t.Error("UpdatedAt should not be before CreatedAt")
	}
}

func TestUpdate_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	tenant := domain.NewTenant("nonexistent", "X", "x", "free")
	err := repo.Update(context.Background(), tenant)
	if !errors.Is(err, domain.ErrTenantNotFound) {
		t.Errorf("expected ErrTenantNotFound, got %v", err)
	}
}

func TestList_All(t *testing.T) {
	repo := newTestRepo(t)

	mustCreate(t, repo, domain.NewTenant("t-1", "A", "a", "free"))
	mustCreate(t, repo, domain.NewTenant("t-2", "B", "b", "pro"))

	tenants, err := repo.List(context.Background(), domain.ListFilter{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tenants) != 2 {
		t.Errorf("got %d tenants, want 2", len(tenants))
	}
}

func TestList_FilterByStatus(t *testing.T) {
	repo := newTestRepo(t)

	t1 := domain.NewTenant("t-1", "A", "a", "free")
	mustCreate(t, repo, t1)

	t2 := domain.NewTenant("t-2", "B", "b", "pro")
	mustCreate(t, repo, t2)

	t2.Status = domain.StatusActive
	mustUpdate(t, repo, t2)

	status := domain.StatusActive
	tenants, err := repo.List(context.Background(), domain.ListFilter{Status: &status})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tenants) != 1 {
		t.Fatalf("got %d tenants, want 1", len(tenants))
	}
	if tenants[0].ID != "t-2" {
		t.Errorf("ID = %q, want %q", tenants[0].ID, "t-2")
	}
}

func TestList_Pagination(t *testing.T) {
	repo := newTestRepo(t)

	for i := range 5 {
		id := fmt.Sprintf("t-%d", i)
		slug := fmt.Sprintf("s-%d", i)
		mustCreate(t, repo, domain.NewTenant(id, "T", slug, "free"))
	}

	tenants, err := repo.List(context.Background(), domain.ListFilter{Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tenants) != 2 {
		t.Errorf("got %d tenants, want 2", len(tenants))
	}
}
