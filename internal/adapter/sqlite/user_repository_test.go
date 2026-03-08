package sqlite_test

import (
	"context"
	"errors"
	"testing"

	"github.com/neomorfeo/tenantiq/internal/adapter/sqlite"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// newTestUserRepo creates an in-memory SQLite user repository for testing.
// It reuses the tenant repo to run migrations, then wraps the same DB.
func newTestUserRepo(t *testing.T) *sqlite.UserRepository {
	t.Helper()
	tenantRepo := newTestRepo(t) // runs all migrations
	return sqlite.NewUserRepository(tenantRepo.DB())
}

func TestUserCreate_And_GetByID(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	user := domain.NewUser("u-1", "admin@example.com", "Admin", "hashed", domain.RoleSuperAdmin)

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.GetByID(ctx, "u-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ID != "u-1" {
		t.Errorf("ID = %q, want %q", got.ID, "u-1")
	}
	if got.Email != "admin@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "admin@example.com")
	}
	if got.Name != "Admin" {
		t.Errorf("Name = %q, want %q", got.Name, "Admin")
	}
	if got.PasswordHash != "hashed" {
		t.Errorf("PasswordHash = %q, want %q", got.PasswordHash, "hashed")
	}
	if got.Role != domain.RoleSuperAdmin {
		t.Errorf("Role = %q, want %q", got.Role, domain.RoleSuperAdmin)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestUserGetByID_NotFound(t *testing.T) {
	repo := newTestUserRepo(t)

	_, err := repo.GetByID(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserGetByEmail(t *testing.T) {
	repo := newTestUserRepo(t)

	user := domain.NewUser("u-1", "admin@example.com", "Admin", "hashed", domain.RoleSuperAdmin)
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.GetByEmail(context.Background(), "admin@example.com")
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}
	if got.ID != "u-1" {
		t.Errorf("ID = %q, want %q", got.ID, "u-1")
	}
}

func TestUserGetByEmail_NotFound(t *testing.T) {
	repo := newTestUserRepo(t)

	_, err := repo.GetByEmail(context.Background(), "nonexistent@example.com")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserCreate_DuplicateEmail(t *testing.T) {
	repo := newTestUserRepo(t)

	u1 := domain.NewUser("u-1", "admin@example.com", "Admin", "hashed", domain.RoleSuperAdmin)
	u2 := domain.NewUser("u-2", "admin@example.com", "Admin 2", "hashed2", domain.RoleAdmin)

	if err := repo.Create(context.Background(), u1); err != nil {
		t.Fatalf("Create u1 failed: %v", err)
	}

	err := repo.Create(context.Background(), u2)
	var emailErr *domain.EmailConflictError
	if !errors.As(err, &emailErr) {
		t.Fatalf("expected EmailConflictError, got %v", err)
	}
	if emailErr.Email != "admin@example.com" {
		t.Errorf("email = %q, want %q", emailErr.Email, "admin@example.com")
	}
}

func TestUserUpdate(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	user := domain.NewUser("u-1", "admin@example.com", "Admin", "hashed", domain.RoleSuperAdmin)
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	user.Name = "Super Admin"
	user.Role = domain.RoleAdmin

	if err := repo.Update(ctx, user); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := repo.GetByID(ctx, "u-1")
	if got.Name != "Super Admin" {
		t.Errorf("Name = %q, want %q", got.Name, "Super Admin")
	}
	if got.Role != domain.RoleAdmin {
		t.Errorf("Role = %q, want %q", got.Role, domain.RoleAdmin)
	}
	if got.UpdatedAt.Before(got.CreatedAt) {
		t.Error("UpdatedAt should not be before CreatedAt")
	}
}

func TestUserUpdate_NotFound(t *testing.T) {
	repo := newTestUserRepo(t)

	user := domain.NewUser("nonexistent", "x@example.com", "X", "hashed", domain.RoleViewer)
	err := repo.Update(context.Background(), user)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserList(t *testing.T) {
	repo := newTestUserRepo(t)

	u1 := domain.NewUser("u-1", "a@example.com", "A", "h1", domain.RoleSuperAdmin)
	u2 := domain.NewUser("u-2", "b@example.com", "B", "h2", domain.RoleAdmin)

	if err := repo.Create(context.Background(), u1); err != nil {
		t.Fatalf("Create u1 failed: %v", err)
	}
	if err := repo.Create(context.Background(), u2); err != nil {
		t.Fatalf("Create u2 failed: %v", err)
	}

	users, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("got %d users, want 2", len(users))
	}
}

func TestUserDelete(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	user := domain.NewUser("u-1", "admin@example.com", "Admin", "hashed", domain.RoleSuperAdmin)
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := repo.Delete(ctx, "u-1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := repo.GetByID(ctx, "u-1")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound after delete, got %v", err)
	}
}

func TestUserDelete_NotFound(t *testing.T) {
	repo := newTestUserRepo(t)

	err := repo.Delete(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserCount(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	u := domain.NewUser("u-1", "a@example.com", "A", "h", domain.RoleSuperAdmin)
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}
