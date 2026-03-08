package app_test

import (
	"context"
	"testing"

	"github.com/neomorfeo/tenantiq/internal/app"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

func TestBootstrap_CreatesFirstUser(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}

	result, err := app.Bootstrap(context.Background(), repo, hasher, "admin@tenantiq.local")
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	if !result.Created {
		t.Error("Created should be true")
	}
	if result.Email != "admin@tenantiq.local" {
		t.Errorf("Email = %q, want %q", result.Email, "admin@tenantiq.local")
	}
	if result.Password == "" {
		t.Error("Password should not be empty")
	}

	// Verify user was persisted.
	count, _ := repo.Count(context.Background())
	if count != 1 {
		t.Errorf("user count = %d, want 1", count)
	}

	// Verify user is superadmin.
	users, _ := repo.List(context.Background())
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].Role != domain.RoleSuperAdmin {
		t.Errorf("Role = %q, want %q", users[0].Role, domain.RoleSuperAdmin)
	}
}

func TestBootstrap_SkipsIfUsersExist(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}

	// Seed an existing user.
	hash, _ := hasher.Hash("pass")
	user := domain.NewUser("u-1", "existing@example.com", "Existing", hash, domain.RoleSuperAdmin)
	repo.users[user.ID] = user
	repo.emails[user.Email] = user

	result, err := app.Bootstrap(context.Background(), repo, hasher, "admin@tenantiq.local")
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	if result.Created {
		t.Error("Created should be false when users already exist")
	}
	if result.Password != "" {
		t.Error("Password should be empty when not created")
	}
}

func TestBootstrap_GeneratesUniquePasswords(t *testing.T) {
	passwords := make(map[string]bool)
	for range 10 {
		repo := newMockUserRepo()
		hasher := &mockHasher{}

		result, err := app.Bootstrap(context.Background(), repo, hasher, "admin@tenantiq.local")
		if err != nil {
			t.Fatalf("Bootstrap() error = %v", err)
		}
		if passwords[result.Password] {
			t.Fatalf("duplicate password generated: %q", result.Password)
		}
		passwords[result.Password] = true
	}
}
