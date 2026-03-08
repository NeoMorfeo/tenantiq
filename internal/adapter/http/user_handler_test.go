package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	adapter "github.com/neomorfeo/tenantiq/internal/adapter/http"
	"github.com/neomorfeo/tenantiq/internal/adapter/sqlite"
	"github.com/neomorfeo/tenantiq/internal/app"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// newUserTestServer creates a test server with auth middleware and user endpoints.
// It seeds a superadmin user and returns the server + user ID.
func newUserTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	repo, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("creating test repo: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	userRepo := sqlite.NewUserRepository(repo.DB())
	hasher := &testHasher{}
	userSvc := app.NewUserService(userRepo, hasher)

	router := chi.NewMux()
	router.Use(adapter.InjectHTTP)
	api := humachi.New(router, huma.DefaultConfig("test", "1.0.0"))
	tokens := &mockTokenService{}
	api.UseMiddleware(adapter.NewHumaAuthMiddleware(api, tokens))
	adapter.RegisterUsers(api, userSvc)

	admin, err := userSvc.CreateUser(context.Background(), "admin@example.com", "Admin", "pass", domain.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("seeding admin: %v", err)
	}

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return srv, admin.ID
}

// testHasher is a deterministic hasher for tests.
type testHasher struct{}

func (h *testHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (h *testHasher) Verify(password, hash string) error {
	if hash == "hashed:"+password {
		return nil
	}
	return fmt.Errorf("mismatch")
}

// doAuthRequest performs an HTTP request with an optional Bearer token.
func doAuthRequest(t *testing.T, method, url, body, token string) *http.Response {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, reader)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, url, err)
	}
	return resp
}

// --- List Users ---

func TestUserHandler_ListUsers(t *testing.T) {
	srv, _ := newUserTestServer(t)

	resp := doAuthRequest(t, http.MethodGet, srv.URL+"/api/v1/users", "", "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var users []adapter.UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("got %d users, want 1 (the seeded admin)", len(users))
	}
}

func TestUserHandler_ListUsers_Unauthorized(t *testing.T) {
	srv, _ := newUserTestServer(t)

	resp := doAuthRequest(t, http.MethodGet, srv.URL+"/api/v1/users", "", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestUserHandler_ListUsers_Forbidden(t *testing.T) {
	srv, _ := newUserTestServer(t)

	resp := doAuthRequest(t, http.MethodGet, srv.URL+"/api/v1/users", "", "viewer-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

// --- Create User ---

func TestUserHandler_CreateUser(t *testing.T) {
	srv, _ := newUserTestServer(t)

	body := `{"email":"new@example.com","name":"New User","password":"password123","role":"viewer"}`
	resp := doAuthRequest(t, http.MethodPost, srv.URL+"/api/v1/users", body, "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var user adapter.UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if user.Email != "new@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "new@example.com")
	}
	if user.Role != "viewer" {
		t.Errorf("Role = %q, want %q", user.Role, "viewer")
	}
}

func TestUserHandler_CreateUser_DuplicateEmail(t *testing.T) {
	srv, _ := newUserTestServer(t)

	body := `{"email":"admin@example.com","name":"Dup","password":"password123","role":"viewer"}`
	resp := doAuthRequest(t, http.MethodPost, srv.URL+"/api/v1/users", body, "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

// --- Update User (PATCH) ---

func TestUserHandler_UpdateUser(t *testing.T) {
	srv, adminID := newUserTestServer(t)

	body := `{"name":"Updated Admin"}`
	resp := doAuthRequest(t, http.MethodPatch, srv.URL+"/api/v1/users/"+adminID, body, "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var user adapter.UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if user.Name != "Updated Admin" {
		t.Errorf("Name = %q, want %q", user.Name, "Updated Admin")
	}
}

func TestUserHandler_UpdateUser_Role(t *testing.T) {
	srv, adminID := newUserTestServer(t)

	body := `{"role":"viewer"}`
	resp := doAuthRequest(t, http.MethodPatch, srv.URL+"/api/v1/users/"+adminID, body, "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var user adapter.UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if user.Role != "viewer" {
		t.Errorf("Role = %q, want %q", user.Role, "viewer")
	}
}

func TestUserHandler_UpdateUser_NotFound(t *testing.T) {
	srv, _ := newUserTestServer(t)

	body := `{"name":"X"}`
	resp := doAuthRequest(t, http.MethodPatch, srv.URL+"/api/v1/users/nonexistent", body, "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- Delete User ---

func TestUserHandler_DeleteUser(t *testing.T) {
	srv, _ := newUserTestServer(t)

	// Create a user to delete.
	createBody := `{"email":"to-delete@example.com","name":"Delete Me","password":"password123","role":"viewer"}`
	createResp := doAuthRequest(t, http.MethodPost, srv.URL+"/api/v1/users", createBody, "valid-token")
	var created adapter.UserResponse
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created user: %v", err)
	}
	createResp.Body.Close()

	resp := doAuthRequest(t, http.MethodDelete, srv.URL+"/api/v1/users/"+created.ID, "", "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	// Verify user is gone.
	getResp := doAuthRequest(t, http.MethodGet, srv.URL+"/api/v1/users/"+created.ID, "", "valid-token")
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("get after delete: status = %d, want %d", getResp.StatusCode, http.StatusNotFound)
	}
}

func TestUserHandler_DeleteUser_Self(t *testing.T) {
	srv, _ := newUserTestServer(t)

	// mockTokenService returns UserID "u-1" for "valid-token".
	// Deleting "u-1" triggers self-deletion guard → 409.
	resp := doAuthRequest(t, http.MethodDelete, srv.URL+"/api/v1/users/u-1", "", "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestUserHandler_DeleteUser_NotFound(t *testing.T) {
	srv, _ := newUserTestServer(t)

	resp := doAuthRequest(t, http.MethodDelete, srv.URL+"/api/v1/users/nonexistent", "", "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}
