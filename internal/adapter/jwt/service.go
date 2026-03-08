package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/neomorfeo/tenantiq/internal/domain"
)

// Service implements domain.TokenService using JWT with HMAC-SHA256.
type Service struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	signingMethod jwt.SigningMethod
}

// New creates a JWT token service.
// The secret must be at least 32 bytes for HMAC-SHA256 security.
func New(secret []byte, accessExpiry, refreshExpiry time.Duration) *Service {
	return &Service{
		secret:        secret,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
		signingMethod: jwt.SigningMethodHS256,
	}
}

// claims extends jwt.RegisteredClaims with user-specific fields.
type claims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Role  string `json:"role"`
	Type  string `json:"type"` // "access" or "refresh"
}

func (s *Service) GenerateAccess(user domain.User) (string, error) {
	return s.generate(user, "access", s.accessExpiry)
}

func (s *Service) GenerateRefresh(user domain.User) (string, error) {
	return s.generate(user, "refresh", s.refreshExpiry)
}

func (s *Service) ValidateAccess(tokenString string) (domain.TokenClaims, error) {
	return s.validate(tokenString, "access")
}

func (s *Service) ValidateRefresh(tokenString string) (domain.TokenClaims, error) {
	return s.validate(tokenString, "refresh")
}

func (s *Service) generate(user domain.User, tokenType string, expiry time.Duration) (string, error) {
	now := time.Now().UTC()

	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
		Email: user.Email,
		Role:  string(user.Role),
		Type:  tokenType,
	}

	token := jwt.NewWithClaims(s.signingMethod, c)

	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("signing %s token: %w", tokenType, err)
	}

	return signed, nil
}

func (s *Service) validate(tokenString, expectedType string) (domain.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != s.signingMethod.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return s.secret, nil
	})
	if err != nil {
		return domain.TokenClaims{}, domain.ErrUnauthorized
	}

	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return domain.TokenClaims{}, domain.ErrUnauthorized
	}

	if c.Type != expectedType {
		return domain.TokenClaims{}, domain.ErrUnauthorized
	}

	return domain.TokenClaims{
		UserID: c.Subject,
		Email:  c.Email,
		Role:   domain.Role(c.Role),
	}, nil
}
