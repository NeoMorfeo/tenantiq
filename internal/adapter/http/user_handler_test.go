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

// --- Password Update ---

// newFullTestServer creates a test server with auth middleware, user endpoints,
// password update, and auth endpoints (login, refresh, me).
func newFullTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	repo, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("creating test repo: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	userRepo := sqlite.NewUserRepository(repo.DB())
	hasher := &testHasher{}
	tokens := &mockTokenService{}
	userSvc := app.NewUserService(userRepo, hasher)
	authSvc := app.NewAuthService(userRepo, hasher, tokens)

	router := chi.NewMux()
	router.Use(adapter.InjectHTTP)
	api := humachi.New(router, huma.DefaultConfig("test", "1.0.0"))
	api.UseMiddleware(adapter.NewHumaAuthMiddleware(api, tokens))
	adapter.RegisterUsers(api, userSvc)
	adapter.RegisterPasswordUpdate(api, userSvc)
	adapter.RegisterAuth(api, authSvc, tokens, adapter.CookieConfig{})

	admin, err := userSvc.CreateUser(context.Background(), "admin@example.com", "Admin", "pass", domain.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("seeding admin: %v", err)
	}

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return srv, admin.ID
}

func TestUserHandler_UpdatePassword_Forbidden(t *testing.T) {
	srv, adminID := newFullTestServer(t)

	// "valid-token" gives UserID "u-1" which differs from adminID → 403.
	body := `{"current_password":"pass","new_password":"newpass123"}`
	resp := doAuthRequest(t, http.MethodPut, srv.URL+"/api/v1/users/"+adminID+"/password", body, "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestUserHandler_UpdatePassword_SelfOnly(t *testing.T) {
	srv, _ := newFullTestServer(t)

	// "valid-token" gives UserID "u-1", trying to change "u-2" password → 403.
	body := `{"current_password":"pass","new_password":"newpass123"}`
	resp := doAuthRequest(t, http.MethodPut, srv.URL+"/api/v1/users/u-2/password", body, "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestUserHandler_UpdatePassword_Unauthenticated(t *testing.T) {
	srv, adminID := newFullTestServer(t)

	body := `{"current_password":"pass","new_password":"newpass123"}`
	resp := doAuthRequest(t, http.MethodPut, srv.URL+"/api/v1/users/"+adminID+"/password", body, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestUserHandler_UpdatePassword_WrongCurrentPassword(t *testing.T) {
	srv, _ := newFullTestServer(t)

	// "valid-token" gives UserID "u-1"; update password for "u-1".
	// User "u-1" doesn't exist in DB → will hit toUserError (user not found).
	body := `{"current_password":"wrongpass","new_password":"newpass123"}`
	resp := doAuthRequest(t, http.MethodPut, srv.URL+"/api/v1/users/u-1/password", body, "valid-token")
	defer resp.Body.Close()

	// User "u-1" doesn't exist in DB, so we get 404 (user not found) from toUserError.
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- Auth Handler (Login) ---

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	srv, _ := newFullTestServer(t)

	body := `{"email":"admin@example.com","password":"wrongpassword"}`
	resp := doAuthRequest(t, http.MethodPost, srv.URL+"/api/v1/auth/login", body, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestAuthHandler_Login_NonexistentUser(t *testing.T) {
	srv, _ := newFullTestServer(t)

	body := `{"email":"nobody@example.com","password":"somepassword"}`
	resp := doAuthRequest(t, http.MethodPost, srv.URL+"/api/v1/auth/login", body, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	srv, _ := newFullTestServer(t)

	body := `{"email":"admin@example.com","password":"pass"}`
	resp := doAuthRequest(t, http.MethodPost, srv.URL+"/api/v1/auth/login", body, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var pair app.TokenPair
	if err := json.NewDecoder(resp.Body).Decode(&pair); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if pair.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
}

// --- Auth Handler (Refresh) ---

func TestAuthHandler_Refresh_MissingToken(t *testing.T) {
	srv, _ := newFullTestServer(t)

	// Empty body, no cookie → missing refresh token.
	body := `{}`
	resp := doAuthRequest(t, http.MethodPost, srv.URL+"/api/v1/auth/refresh", body, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	srv, _ := newFullTestServer(t)

	body := `{"refresh_token":"invalid-refresh-token"}`
	resp := doAuthRequest(t, http.MethodPost, srv.URL+"/api/v1/auth/refresh", body, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

// --- Auth Handler (Me) ---

func TestAuthHandler_Me_Authenticated(t *testing.T) {
	srv, _ := newFullTestServer(t)

	resp := doAuthRequest(t, http.MethodGet, srv.URL+"/api/v1/auth/me", "", "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var out struct {
		UserID string      `json:"user_id"`
		Email  string      `json:"email"`
		Role   domain.Role `json:"role"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.UserID != "u-1" {
		t.Errorf("UserID = %q, want %q", out.UserID, "u-1")
	}
	if out.Email != "admin@example.com" {
		t.Errorf("Email = %q, want %q", out.Email, "admin@example.com")
	}
}

func TestAuthHandler_Me_Unauthenticated(t *testing.T) {
	srv, _ := newFullTestServer(t)

	resp := doAuthRequest(t, http.MethodGet, srv.URL+"/api/v1/auth/me", "", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

// --- Auth Handler (Logout) ---

func TestAuthHandler_Logout(t *testing.T) {
	srv, _ := newFullTestServer(t)

	resp := doAuthRequest(t, http.MethodPost, srv.URL+"/api/v1/auth/logout", "", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

// --- Profile Endpoints ---

// newProfileTestServer creates a test server with auth middleware, user endpoints,
// and profile endpoints. It seeds a superadmin user whose ID matches the mock
// token's UserID ("u-1") so that profile self-service endpoints work.
func newProfileTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	repo, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("creating test repo: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	userRepo := sqlite.NewUserRepository(repo.DB())
	hasher := &testHasher{}
	userSvc := app.NewUserService(userRepo, hasher)

	// Seed user with ID "u-1" to match mockTokenService's "valid-token" claims.
	u := domain.NewUser("u-1", "admin@example.com", "Admin", "hashed:pass", domain.RoleSuperAdmin)
	if err := userRepo.Create(context.Background(), u); err != nil {
		t.Fatalf("seeding user: %v", err)
	}

	router := chi.NewMux()
	router.Use(adapter.InjectHTTP)
	api := humachi.New(router, huma.DefaultConfig("test", "1.0.0"))
	tokens := &mockTokenService{}
	api.UseMiddleware(adapter.NewHumaAuthMiddleware(api, tokens))
	adapter.RegisterProfile(api, userSvc)
	adapter.RegisterUsers(api, userSvc)

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return srv
}

func TestProfileHandler_GetProfile(t *testing.T) {
	srv := newProfileTestServer(t)

	resp := doAuthRequest(t, http.MethodGet, srv.URL+"/api/v1/users/me", "", "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var profile adapter.ProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if profile.ID != "u-1" {
		t.Errorf("ID = %q, want %q", profile.ID, "u-1")
	}
	if profile.Theme != "system" {
		t.Errorf("Theme = %q, want %q", profile.Theme, "system")
	}
	if profile.Language != "en" {
		t.Errorf("Language = %q, want %q", profile.Language, "en")
	}
}

func TestProfileHandler_GetProfile_Unauthenticated(t *testing.T) {
	srv := newProfileTestServer(t)

	resp := doAuthRequest(t, http.MethodGet, srv.URL+"/api/v1/users/me", "", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestProfileHandler_UpdateProfile_Name(t *testing.T) {
	srv := newProfileTestServer(t)

	body := `{"name":"Updated Admin"}`
	resp := doAuthRequest(t, http.MethodPatch, srv.URL+"/api/v1/users/me", body, "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var profile adapter.ProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if profile.Name != "Updated Admin" {
		t.Errorf("Name = %q, want %q", profile.Name, "Updated Admin")
	}
}

func TestProfileHandler_UpdateProfile_Email(t *testing.T) {
	srv := newProfileTestServer(t)

	body := `{"email":"newemail@example.com"}`
	resp := doAuthRequest(t, http.MethodPatch, srv.URL+"/api/v1/users/me", body, "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var profile adapter.ProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if profile.Email != "newemail@example.com" {
		t.Errorf("Email = %q, want %q", profile.Email, "newemail@example.com")
	}
}

func TestProfileHandler_UpdateProfile_DuplicateEmail(t *testing.T) {
	srv := newProfileTestServer(t)

	// Create another user first.
	createBody := `{"email":"other@example.com","name":"Other","password":"password123","role":"viewer"}`
	createResp := doAuthRequest(t, http.MethodPost, srv.URL+"/api/v1/users", createBody, "valid-token")
	createResp.Body.Close()

	// Try to update profile email to the other user's email.
	body := `{"email":"other@example.com"}`
	resp := doAuthRequest(t, http.MethodPatch, srv.URL+"/api/v1/users/me", body, "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestProfileHandler_UpdateProfile_Unauthenticated(t *testing.T) {
	srv := newProfileTestServer(t)

	body := `{"name":"X"}`
	resp := doAuthRequest(t, http.MethodPatch, srv.URL+"/api/v1/users/me", body, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestProfileHandler_UpdatePreferences(t *testing.T) {
	srv := newProfileTestServer(t)

	body := `{"theme":"dark","language":"es"}`
	resp := doAuthRequest(t, http.MethodPatch, srv.URL+"/api/v1/users/me/preferences", body, "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var out struct {
		Theme    string `json:"theme"`
		Language string `json:"language"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if out.Theme != "dark" {
		t.Errorf("Theme = %q, want %q", out.Theme, "dark")
	}
	if out.Language != "es" {
		t.Errorf("Language = %q, want %q", out.Language, "es")
	}
}

func TestProfileHandler_UpdatePreferences_ThemeOnly(t *testing.T) {
	srv := newProfileTestServer(t)

	body := `{"theme":"light"}`
	resp := doAuthRequest(t, http.MethodPatch, srv.URL+"/api/v1/users/me/preferences", body, "valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var out struct {
		Theme    string `json:"theme"`
		Language string `json:"language"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if out.Theme != "light" {
		t.Errorf("Theme = %q, want %q", out.Theme, "light")
	}
	if out.Language != "en" {
		t.Errorf("Language = %q, want %q (should be unchanged)", out.Language, "en")
	}
}

func TestProfileHandler_UpdatePreferences_Unauthenticated(t *testing.T) {
	srv := newProfileTestServer(t)

	body := `{"theme":"dark"}`
	resp := doAuthRequest(t, http.MethodPatch, srv.URL+"/api/v1/users/me/preferences", body, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}
