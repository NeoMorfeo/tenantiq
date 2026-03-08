package jwt_test

import (
	"testing"
	"time"

	adapter "github.com/neomorfeo/tenantiq/internal/adapter/jwt"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// Verify Service satisfies the domain port at compile time.
var _ domain.TokenService = (*adapter.Service)(nil)

var testSecret = []byte("test-secret-key-at-least-32-bytes!")

func testUser() domain.User {
	return domain.NewUser("u-1", "admin@example.com", "Admin", "hashed", domain.RoleSuperAdmin)
}

func TestGenerateAccess_And_ValidateAccess(t *testing.T) {
	svc := adapter.New(testSecret, 15*time.Minute, 7*24*time.Hour)
	user := testUser()

	token, err := svc.GenerateAccess(user)
	if err != nil {
		t.Fatalf("GenerateAccess() error = %v", err)
	}
	if token == "" {
		t.Fatal("GenerateAccess() returned empty token")
	}

	claims, err := svc.ValidateAccess(token)
	if err != nil {
		t.Fatalf("ValidateAccess() error = %v", err)
	}

	if claims.UserID != "u-1" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "u-1")
	}
	if claims.Email != "admin@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "admin@example.com")
	}
	if claims.Role != domain.RoleSuperAdmin {
		t.Errorf("Role = %q, want %q", claims.Role, domain.RoleSuperAdmin)
	}
}

func TestGenerateRefresh_And_ValidateRefresh(t *testing.T) {
	svc := adapter.New(testSecret, 15*time.Minute, 7*24*time.Hour)
	user := testUser()

	token, err := svc.GenerateRefresh(user)
	if err != nil {
		t.Fatalf("GenerateRefresh() error = %v", err)
	}

	claims, err := svc.ValidateRefresh(token)
	if err != nil {
		t.Fatalf("ValidateRefresh() error = %v", err)
	}

	if claims.UserID != "u-1" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "u-1")
	}
}

func TestValidateAccess_RejectsRefreshToken(t *testing.T) {
	svc := adapter.New(testSecret, 15*time.Minute, 7*24*time.Hour)

	token, _ := svc.GenerateRefresh(testUser())
	_, err := svc.ValidateAccess(token)
	if err == nil {
		t.Error("ValidateAccess() should reject a refresh token")
	}
}

func TestValidateRefresh_RejectsAccessToken(t *testing.T) {
	svc := adapter.New(testSecret, 15*time.Minute, 7*24*time.Hour)

	token, _ := svc.GenerateAccess(testUser())
	_, err := svc.ValidateRefresh(token)
	if err == nil {
		t.Error("ValidateRefresh() should reject an access token")
	}
}

func TestValidateAccess_RejectsExpiredToken(t *testing.T) {
	svc := adapter.New(testSecret, -1*time.Second, 7*24*time.Hour)

	token, _ := svc.GenerateAccess(testUser())
	_, err := svc.ValidateAccess(token)
	if err == nil {
		t.Error("ValidateAccess() should reject an expired token")
	}
}

func TestValidateAccess_RejectsWrongSecret(t *testing.T) {
	svc1 := adapter.New(testSecret, 15*time.Minute, 7*24*time.Hour)
	svc2 := adapter.New([]byte("different-secret-key-32-bytes!!!"), 15*time.Minute, 7*24*time.Hour)

	token, _ := svc1.GenerateAccess(testUser())
	_, err := svc2.ValidateAccess(token)
	if err == nil {
		t.Error("ValidateAccess() should reject a token signed with a different secret")
	}
}

func TestValidateAccess_RejectsGarbage(t *testing.T) {
	svc := adapter.New(testSecret, 15*time.Minute, 7*24*time.Hour)

	_, err := svc.ValidateAccess("not-a-valid-token")
	if err == nil {
		t.Error("ValidateAccess() should reject garbage input")
	}
}
