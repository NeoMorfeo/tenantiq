package otel_test

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/codes"

	adapter "github.com/neomorfeo/tenantiq/internal/adapter/otel"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// --- Mock user repository ---

type mockUserRepo struct {
	users map[string]domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]domain.User)}
}

func (m *mockUserRepo) Create(_ context.Context, u domain.User) error {
	m.users[u.ID] = u
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
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

func (m *mockUserRepo) List(_ context.Context) ([]domain.User, error) {
	out := make([]domain.User, 0, len(m.users))
	for _, u := range m.users {
		out = append(out, u)
	}
	return out, nil
}

func (m *mockUserRepo) Update(_ context.Context, u domain.User) error {
	if _, ok := m.users[u.ID]; !ok {
		return domain.ErrUserNotFound
	}
	m.users[u.ID] = u
	return nil
}

func (m *mockUserRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.users[id]; !ok {
		return domain.ErrUserNotFound
	}
	delete(m.users, id)
	return nil
}

func (m *mockUserRepo) Count(_ context.Context) (int, error) {
	return len(m.users), nil
}

// --- Tests ---

func TestTracingUserRepository_Create_RecordsSpan(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := newMockUserRepo()
	repo := adapter.NewTracingUserRepository(inner)

	user := domain.NewUser("u-1", "test@example.com", "Test", "hashed", domain.RoleAdmin)
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if spans[0].Name != "UserRepository.Create" {
		t.Errorf("span name = %q, want %q", spans[0].Name, "UserRepository.Create")
	}
	assertAttribute(t, spans[0], "user.id", "u-1")
	assertAttribute(t, spans[0], "user.email", "test@example.com")
}

func TestTracingUserRepository_GetByID_RecordsSpan(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := newMockUserRepo()
	repo := adapter.NewTracingUserRepository(inner)

	inner.users["u-1"] = domain.NewUser("u-1", "test@example.com", "Test", "hashed", domain.RoleAdmin)

	got, err := repo.GetByID(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "u-1" {
		t.Errorf("ID = %q, want %q", got.ID, "u-1")
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	assertAttribute(t, spans[0], "user.id", "u-1")
}

func TestTracingUserRepository_GetByID_RecordsError(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := newMockUserRepo()
	repo := adapter.NewTracingUserRepository(inner)

	_, err := repo.GetByID(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if spans[0].Status.Code != codes.Error {
		t.Errorf("span status = %v, want %v", spans[0].Status.Code, codes.Error)
	}
}

func TestTracingUserRepository_GetByEmail_RecordsSpan(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := newMockUserRepo()
	repo := adapter.NewTracingUserRepository(inner)

	inner.users["u-1"] = domain.NewUser("u-1", "test@example.com", "Test", "hashed", domain.RoleAdmin)

	got, err := repo.GetByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "test@example.com")
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	assertAttribute(t, spans[0], "user.email", "test@example.com")
}

func TestTracingUserRepository_List_RecordsCount(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := newMockUserRepo()
	repo := adapter.NewTracingUserRepository(inner)

	inner.users["u-1"] = domain.NewUser("u-1", "a@example.com", "A", "h", domain.RoleAdmin)
	inner.users["u-2"] = domain.NewUser("u-2", "b@example.com", "B", "h", domain.RoleViewer)

	users, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("got %d users, want 2", len(users))
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	assertAttribute(t, spans[0], "result.count", "2")
}

func TestTracingUserRepository_Update_RecordsSpan(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := newMockUserRepo()
	repo := adapter.NewTracingUserRepository(inner)

	user := domain.NewUser("u-1", "test@example.com", "Test", "hashed", domain.RoleAdmin)
	inner.users["u-1"] = user

	user.Role = domain.RoleViewer
	if err := repo.Update(context.Background(), user); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if spans[0].Name != "UserRepository.Update" {
		t.Errorf("span name = %q, want %q", spans[0].Name, "UserRepository.Update")
	}
	assertAttribute(t, spans[0], "user.role", "viewer")
}

func TestTracingUserRepository_Delete_RecordsSpan(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := newMockUserRepo()
	repo := adapter.NewTracingUserRepository(inner)

	inner.users["u-1"] = domain.NewUser("u-1", "test@example.com", "Test", "hashed", domain.RoleAdmin)

	if err := repo.Delete(context.Background(), "u-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if spans[0].Name != "UserRepository.Delete" {
		t.Errorf("span name = %q, want %q", spans[0].Name, "UserRepository.Delete")
	}
	assertAttribute(t, spans[0], "user.id", "u-1")
}

func TestTracingUserRepository_Delete_RecordsError(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := newMockUserRepo()
	repo := adapter.NewTracingUserRepository(inner)

	err := repo.Delete(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if spans[0].Status.Code != codes.Error {
		t.Errorf("span status = %v, want %v", spans[0].Status.Code, codes.Error)
	}
}

func TestTracingUserRepository_Count_RecordsSpan(t *testing.T) {
	exporter := setupTestTracer(t)
	inner := newMockUserRepo()
	repo := adapter.NewTracingUserRepository(inner)

	inner.users["u-1"] = domain.NewUser("u-1", "a@example.com", "A", "h", domain.RoleAdmin)

	count, err := repo.Count(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	assertAttribute(t, spans[0], "result.count", "1")
}
