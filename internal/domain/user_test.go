package domain_test

import (
	"testing"
	"time"

	"github.com/neomorfeo/tenantiq/internal/domain"
)

func TestNewUser(t *testing.T) {
	before := time.Now().UTC()
	user := domain.NewUser("id-1", "admin@example.com", "Admin", "hashed", domain.RoleSuperAdmin)
	after := time.Now().UTC()

	if user.ID != "id-1" {
		t.Errorf("ID = %q, want %q", user.ID, "id-1")
	}
	if user.Email != "admin@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "admin@example.com")
	}
	if user.Name != "Admin" {
		t.Errorf("Name = %q, want %q", user.Name, "Admin")
	}
	if user.PasswordHash != "hashed" {
		t.Errorf("PasswordHash = %q, want %q", user.PasswordHash, "hashed")
	}
	if user.Role != domain.RoleSuperAdmin {
		t.Errorf("Role = %q, want %q", user.Role, domain.RoleSuperAdmin)
	}
	if user.CreatedAt.Before(before) || user.CreatedAt.After(after) {
		t.Errorf("CreatedAt = %v, want between %v and %v", user.CreatedAt, before, after)
	}
	if user.UpdatedAt != user.CreatedAt {
		t.Errorf("UpdatedAt should equal CreatedAt on new user")
	}
}

func TestIsValidRole(t *testing.T) {
	valid := []domain.Role{
		domain.RoleSuperAdmin,
		domain.RoleAdmin,
		domain.RoleViewer,
	}
	for _, r := range valid {
		if !domain.IsValidRole(r) {
			t.Errorf("IsValidRole(%q) = false, want true", r)
		}
	}

	invalid := []domain.Role{"root", "guest", "", "ADMIN"}
	for _, r := range invalid {
		if domain.IsValidRole(r) {
			t.Errorf("IsValidRole(%q) = true, want false", r)
		}
	}
}

func TestIsValidTheme(t *testing.T) {
	valid := []domain.Theme{
		domain.ThemeLight,
		domain.ThemeDark,
		domain.ThemeSystem,
	}
	for _, th := range valid {
		if !domain.IsValidTheme(th) {
			t.Errorf("IsValidTheme(%q) = false, want true", th)
		}
	}

	invalid := []domain.Theme{"blue", "auto", "", "DARK"}
	for _, th := range invalid {
		if domain.IsValidTheme(th) {
			t.Errorf("IsValidTheme(%q) = true, want false", th)
		}
	}
}

func TestIsValidLanguage(t *testing.T) {
	valid := []domain.Language{
		domain.LangEN,
		domain.LangES,
	}
	for _, l := range valid {
		if !domain.IsValidLanguage(l) {
			t.Errorf("IsValidLanguage(%q) = false, want true", l)
		}
	}

	invalid := []domain.Language{"fr", "de", "", "EN"}
	for _, l := range invalid {
		if domain.IsValidLanguage(l) {
			t.Errorf("IsValidLanguage(%q) = true, want false", l)
		}
	}
}

func TestTokenClaims_Fields(t *testing.T) {
	claims := domain.TokenClaims{
		UserID: "user-1",
		Email:  "admin@example.com",
		Role:   domain.RoleSuperAdmin,
	}

	if claims.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-1")
	}
	if claims.Email != "admin@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "admin@example.com")
	}
	if claims.Role != domain.RoleSuperAdmin {
		t.Errorf("Role = %q, want %q", claims.Role, domain.RoleSuperAdmin)
	}
}
