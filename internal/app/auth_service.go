package app

import (
	"context"
	"fmt"

	"github.com/neomorfeo/tenantiq/internal/domain"
)

// TokenPair holds the access and refresh tokens returned after authentication.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// AuthService handles authentication: login and token refresh.
type AuthService struct {
	users  domain.UserRepository
	hasher domain.PasswordHasher
	tokens domain.TokenService
}

// NewAuthService creates an authentication service with the given adapters.
func NewAuthService(users domain.UserRepository, hasher domain.PasswordHasher, tokens domain.TokenService) *AuthService {
	return &AuthService{
		users:  users,
		hasher: hasher,
		tokens: tokens,
	}
}

// Login validates credentials and returns a token pair.
func (s *AuthService) Login(ctx context.Context, email, password string) (TokenPair, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// Return generic error to avoid leaking whether the email exists.
		return TokenPair{}, domain.ErrInvalidCredentials
	}

	if err := s.hasher.Verify(password, user.PasswordHash); err != nil {
		return TokenPair{}, domain.ErrInvalidCredentials
	}

	return s.generateTokenPair(user)
}

// Refresh validates a refresh token and returns a new access token.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	claims, err := s.tokens.ValidateRefresh(refreshToken)
	if err != nil {
		return TokenPair{}, domain.ErrUnauthorized
	}

	// Re-fetch user to get current role (in case it changed since token was issued).
	user, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		return TokenPair{}, domain.ErrUnauthorized
	}

	return s.generateTokenPair(user)
}

func (s *AuthService) generateTokenPair(user domain.User) (TokenPair, error) {
	access, err := s.tokens.GenerateAccess(user)
	if err != nil {
		return TokenPair{}, fmt.Errorf("generating access token: %w", err)
	}

	refresh, err := s.tokens.GenerateRefresh(user)
	if err != nil {
		return TokenPair{}, fmt.Errorf("generating refresh token: %w", err)
	}

	return TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}
