package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/neomorfeo/tenantiq/internal/app"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// --- Auth Mocks ---

type mockUserRepo struct {
	users    map[string]domain.User
	emails   map[string]domain.User
	createFn func(domain.User) error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:  make(map[string]domain.User),
		emails: make(map[string]domain.User),
	}
}

func (m *mockUserRepo) Create(_ context.Context, u domain.User) error {
	if m.createFn != nil {
		return m.createFn(u)
	}
	m.users[u.ID] = u
	m.emails[u.Email] = u
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id string) (domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (domain.User, error) {
	u, ok := m.emails[email]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepo) List(_ context.Context) ([]domain.User, error) {
	out := make([]domain.User, 0, len(m.users))
	for _, u := range m.users {
		out = append(out, u)
	}
	return out, nil
}

func (m *mockUserRepo) Update(_ context.Context, u domain.User) error {
	m.users[u.ID] = u
	m.emails[u.Email] = u
	return nil
}

func (m *mockUserRepo) Count(_ context.Context) (int, error) {
	return len(m.users), nil
}

type mockHasher struct{}

func (m *mockHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (m *mockHasher) Verify(password, hash string) error {
	if hash == "hashed:"+password {
		return nil
	}
	return errors.New("mismatch")
}

type mockTokenService struct {
	accessErr  error
	refreshErr error
}

func (m *mockTokenService) GenerateAccess(u domain.User) (string, error) {
	if m.accessErr != nil {
		return "", m.accessErr
	}
	return "access:" + u.ID, nil
}

func (m *mockTokenService) GenerateRefresh(u domain.User) (string, error) {
	if m.refreshErr != nil {
		return "", m.refreshErr
	}
	return "refresh:" + u.ID, nil
}

func (m *mockTokenService) ValidateAccess(token string) (domain.TokenClaims, error) {
	if len(token) > 7 && token[:7] == "access:" {
		id := token[7:]
		return domain.TokenClaims{UserID: id, Role: domain.RoleSuperAdmin}, nil
	}
	return domain.TokenClaims{}, domain.ErrUnauthorized
}

func (m *mockTokenService) ValidateRefresh(token string) (domain.TokenClaims, error) {
	if len(token) > 8 && token[:8] == "refresh:" {
		id := token[8:]
		return domain.TokenClaims{UserID: id, Role: domain.RoleSuperAdmin}, nil
	}
	return domain.TokenClaims{}, domain.ErrUnauthorized
}

// --- Auth Tests ---

func seedUser(repo *mockUserRepo, hasher *mockHasher) domain.User {
	hash, _ := hasher.Hash("correctpassword")
	user := domain.NewUser("u-1", "admin@example.com", "Admin", hash, domain.RoleSuperAdmin)
	repo.users[user.ID] = user
	repo.emails[user.Email] = user
	return user
}

func TestLogin_Success(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	tokens := &mockTokenService{}
	seedUser(repo, hasher)

	svc := app.NewAuthService(repo, hasher, tokens)

	pair, err := svc.Login(context.Background(), "admin@example.com", "correctpassword")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if pair.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	tokens := &mockTokenService{}
	seedUser(repo, hasher)

	svc := app.NewAuthService(repo, hasher, tokens)

	_, err := svc.Login(context.Background(), "admin@example.com", "wrongpassword")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	tokens := &mockTokenService{}

	svc := app.NewAuthService(repo, hasher, tokens)

	_, err := svc.Login(context.Background(), "nonexistent@example.com", "password")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials (not ErrUserNotFound to avoid leaking), got %v", err)
	}
}

func TestRefresh_Success(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	tokens := &mockTokenService{}
	seedUser(repo, hasher)

	svc := app.NewAuthService(repo, hasher, tokens)

	pair, err := svc.Refresh(context.Background(), "refresh:u-1")
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	tokens := &mockTokenService{}

	svc := app.NewAuthService(repo, hasher, tokens)

	_, err := svc.Refresh(context.Background(), "garbage")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestRefresh_UserDeleted(t *testing.T) {
	repo := newMockUserRepo()
	hasher := &mockHasher{}
	tokens := &mockTokenService{}
	// Don't seed user — valid token but user doesn't exist.

	svc := app.NewAuthService(repo, hasher, tokens)

	_, err := svc.Refresh(context.Background(), "refresh:u-1")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}
