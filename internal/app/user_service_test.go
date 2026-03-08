package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/neomorfeo/tenantiq/internal/app"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

func TestCreateUser_Success(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	svc := app.NewUserService(repo, hasher)

	user, err := svc.CreateUser(context.Background(), "user@example.com", "User", "password123", domain.RoleAdmin)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	if user.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "user@example.com")
	}
	if user.Role != domain.RoleAdmin {
		t.Errorf("Role = %q, want %q", user.Role, domain.RoleAdmin)
	}
	if user.ID == "" {
		t.Error("ID should not be empty")
	}
	// Password should be hashed, not stored in plain.
	if user.PasswordHash == "password123" {
		t.Error("PasswordHash should not be the plain password")
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	svc := app.NewUserService(repo, hasher)

	if _, err := svc.CreateUser(context.Background(), "user@example.com", "User", "pass", domain.RoleAdmin); err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err := svc.CreateUser(context.Background(), "user@example.com", "User 2", "pass", domain.RoleViewer)
	var emailErr *domain.EmailConflictError
	if !errors.As(err, &emailErr) {
		t.Fatalf("expected EmailConflictError, got %v", err)
	}
}

func TestCreateUser_InvalidRole(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	svc := app.NewUserService(repo, hasher)

	_, err := svc.CreateUser(context.Background(), "user@example.com", "User", "pass", domain.Role("root"))
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestListUsers(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	svc := app.NewUserService(repo, hasher)

	if _, err := svc.CreateUser(context.Background(), "a@example.com", "A", "pass", domain.RoleAdmin); err != nil {
		t.Fatalf("create A failed: %v", err)
	}
	if _, err := svc.CreateUser(context.Background(), "b@example.com", "B", "pass", domain.RoleViewer); err != nil {
		t.Fatalf("create B failed: %v", err)
	}

	users, err := svc.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 2 {
		t.Errorf("got %d users, want 2", len(users))
	}
}

func TestGetUser_Success(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	svc := app.NewUserService(repo, hasher)

	created, _ := svc.CreateUser(context.Background(), "user@example.com", "User", "pass", domain.RoleAdmin)

	got, err := svc.GetUser(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	svc := app.NewUserService(repo, hasher)

	_, err := svc.GetUser(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

// --- UpdateUser ---

func TestUpdateUser_Name(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	svc := app.NewUserService(repo, hasher)

	created, _ := svc.CreateUser(context.Background(), "user@example.com", "User", "pass", domain.RoleAdmin)

	name := "Updated Name"
	updated, err := svc.UpdateUser(context.Background(), created.ID, &name, nil)
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "Updated Name")
	}
	if updated.Role != domain.RoleAdmin {
		t.Errorf("Role = %q, want %q (should be unchanged)", updated.Role, domain.RoleAdmin)
	}
}

func TestUpdateUser_Role(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	svc := app.NewUserService(repo, hasher)

	created, _ := svc.CreateUser(context.Background(), "user@example.com", "User", "pass", domain.RoleAdmin)

	role := domain.RoleViewer
	updated, err := svc.UpdateUser(context.Background(), created.ID, nil, &role)
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if updated.Role != domain.RoleViewer {
		t.Errorf("Role = %q, want %q", updated.Role, domain.RoleViewer)
	}
	if updated.Name != "User" {
		t.Errorf("Name = %q, want %q (should be unchanged)", updated.Name, "User")
	}
}

func TestUpdateUser_InvalidRole(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	svc := app.NewUserService(repo, hasher)

	created, _ := svc.CreateUser(context.Background(), "user@example.com", "User", "pass", domain.RoleAdmin)

	role := domain.Role("root")
	_, err := svc.UpdateUser(context.Background(), created.ID, nil, &role)
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestUpdateUser_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	svc := app.NewUserService(repo, hasher)

	name := "X"
	_, err := svc.UpdateUser(context.Background(), "nonexistent", &name, nil)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

// --- DeleteUser ---

func TestDeleteUser_Success(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	svc := app.NewUserService(repo, hasher)

	created, _ := svc.CreateUser(context.Background(), "user@example.com", "User", "pass", domain.RoleAdmin)

	err := svc.DeleteUser(context.Background(), created.ID, "other-user-id")
	if err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}

	_, err = svc.GetUser(context.Background(), created.ID)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound after delete, got %v", err)
	}
}

func TestDeleteUser_SelfDelete(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	svc := app.NewUserService(repo, hasher)

	created, _ := svc.CreateUser(context.Background(), "user@example.com", "User", "pass", domain.RoleAdmin)

	err := svc.DeleteUser(context.Background(), created.ID, created.ID)
	if !errors.Is(err, domain.ErrSelfDelete) {
		t.Errorf("expected ErrSelfDelete, got %v", err)
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	svc := app.NewUserService(repo, hasher)

	err := svc.DeleteUser(context.Background(), "nonexistent", "caller")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

// --- UpdatePassword ---

func TestUpdatePassword_Success(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	svc := app.NewUserService(repo, hasher)

	user, _ := svc.CreateUser(context.Background(), "user@example.com", "User", "oldpass", domain.RoleAdmin)

	err := svc.UpdatePassword(context.Background(), user.ID, "oldpass", "newpass")
	if err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}

	// Verify new password works (hash updated in repo).
	updated, _ := repo.GetByID(context.Background(), user.ID)
	if err := hasher.Verify("newpass", updated.PasswordHash); err != nil {
		t.Error("new password should verify successfully after update")
	}
}

func TestUpdatePassword_WrongCurrent(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	svc := app.NewUserService(repo, hasher)

	user, _ := svc.CreateUser(context.Background(), "user@example.com", "User", "oldpass", domain.RoleAdmin)

	err := svc.UpdatePassword(context.Background(), user.ID, "wrongpass", "newpass")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}
