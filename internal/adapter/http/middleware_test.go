package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	handler "github.com/neomorfeo/tenantiq/internal/adapter/http"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// mockTokenService for middleware tests.
type mockTokenService struct{}

func (m *mockTokenService) GenerateAccess(u domain.User) (string, error) {
	return "access:" + u.ID, nil
}

func (m *mockTokenService) GenerateRefresh(u domain.User) (string, error) {
	return "refresh:" + u.ID, nil
}

func (m *mockTokenService) ValidateAccess(token string) (domain.TokenClaims, error) {
	if token == "valid-token" {
		return domain.TokenClaims{UserID: "u-1", Email: "admin@example.com", Role: domain.RoleSuperAdmin}, nil
	}
	if token == "viewer-token" {
		return domain.TokenClaims{UserID: "u-2", Email: "viewer@example.com", Role: domain.RoleViewer}, nil
	}
	return domain.TokenClaims{}, domain.ErrUnauthorized
}

func (m *mockTokenService) ValidateRefresh(token string) (domain.TokenClaims, error) {
	return domain.TokenClaims{}, domain.ErrUnauthorized
}

// newTestAPI creates a Huma API with the auth middleware and a test endpoint.
func newTestAPI(t *testing.T, security []map[string][]string) *httptest.Server {
	t.Helper()
	router := chi.NewMux()
	api := humachi.New(router, huma.DefaultConfig("test", "1.0.0"))
	tokens := &mockTokenService{}

	api.UseMiddleware(handler.NewHumaAuthMiddleware(api, tokens))

	huma.Register(api, huma.Operation{
		OperationID: "test-op",
		Method:      http.MethodGet,
		Path:        "/test",
		Security:    security,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body string }, error) {
		claims, ok := handler.ClaimsFromContext(ctx)
		if ok {
			return &struct{ Body string }{Body: claims.UserID}, nil
		}
		return &struct{ Body string }{Body: "public"}, nil
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv
}

func doGet(t *testing.T, url, authHeader string) *http.Response {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func TestHumaAuth_PublicEndpoint(t *testing.T) {
	srv := newTestAPI(t, nil) // no Security = public

	resp := doGet(t, srv.URL+"/test", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHumaAuth_ProtectedEndpoint_ValidToken(t *testing.T) {
	srv := newTestAPI(t, handler.BearerSecurity)

	resp := doGet(t, srv.URL+"/test", "Bearer valid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHumaAuth_ProtectedEndpoint_MissingHeader(t *testing.T) {
	srv := newTestAPI(t, handler.BearerSecurity)

	resp := doGet(t, srv.URL+"/test", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestHumaAuth_ProtectedEndpoint_InvalidToken(t *testing.T) {
	srv := newTestAPI(t, handler.BearerSecurity)

	resp := doGet(t, srv.URL+"/test", "Bearer invalid-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestHumaAuth_RoleRequired_Allowed(t *testing.T) {
	srv := newTestAPI(t, handler.RoleSecurity(domain.RoleSuperAdmin))

	resp := doGet(t, srv.URL+"/test", "Bearer valid-token") // superadmin
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHumaAuth_RoleRequired_Forbidden(t *testing.T) {
	srv := newTestAPI(t, handler.RoleSecurity(domain.RoleSuperAdmin))

	resp := doGet(t, srv.URL+"/test", "Bearer viewer-token") // viewer
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}
