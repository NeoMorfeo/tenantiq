package app

import (
	"context"
	"fmt"

	"github.com/neomorfeo/tenantiq/internal/domain"
)

// UserService handles user management operations.
type UserService struct {
	users  domain.UserRepository
	hasher domain.PasswordHasher
}

// NewUserService creates a user management service with the given adapters.
func NewUserService(users domain.UserRepository, hasher domain.PasswordHasher) *UserService {
	return &UserService{
		users:  users,
		hasher: hasher,
	}
}

// CreateUser creates a new user with the given attributes.
func (s *UserService) CreateUser(ctx context.Context, email, name, password string, role domain.Role) (domain.User, error) {
	if !domain.IsValidRole(role) {
		return domain.User{}, fmt.Errorf("invalid role %q", role)
	}

	// Check email uniqueness before creating.
	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return domain.User{}, &domain.EmailConflictError{Email: email}
	}

	id, err := generateID()
	if err != nil {
		return domain.User{}, fmt.Errorf("generating user id: %w", err)
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return domain.User{}, fmt.Errorf("hashing password: %w", err)
	}

	user := domain.NewUser(id, email, name, hash, role)

	if err := s.users.Create(ctx, user); err != nil {
		return domain.User{}, fmt.Errorf("creating user: %w", err)
	}

	return user, nil
}

// GetUser returns a user by ID.
func (s *UserService) GetUser(ctx context.Context, id string) (domain.User, error) {
	return s.users.GetByID(ctx, id)
}

// ListUsers returns all users.
func (s *UserService) ListUsers(ctx context.Context) ([]domain.User, error) {
	return s.users.List(ctx)
}

// UpdatePassword changes a user's password after verifying the current one.
func (s *UserService) UpdatePassword(ctx context.Context, id, currentPassword, newPassword string) error {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.hasher.Verify(currentPassword, user.PasswordHash); err != nil {
		return domain.ErrInvalidCredentials
	}

	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	user.PasswordHash = hash

	return s.users.Update(ctx, user)
}
