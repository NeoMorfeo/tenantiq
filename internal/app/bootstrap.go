package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/neomorfeo/tenantiq/internal/domain"
)

// BootstrapResult holds the outcome of the bootstrap process.
type BootstrapResult struct {
	Created  bool
	Email    string
	Password string
}

// Bootstrap creates the initial superadmin user if no users exist.
// Returns the generated password so it can be displayed once at startup.
func Bootstrap(ctx context.Context, users domain.UserRepository, hasher domain.PasswordHasher, adminEmail string) (BootstrapResult, error) {
	count, err := users.Count(ctx)
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("checking user count: %w", err)
	}

	if count > 0 {
		return BootstrapResult{Created: false}, nil
	}

	password, err := generatePassword(32)
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("generating password: %w", err)
	}

	id, err := generateID()
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("generating user id: %w", err)
	}

	hash, err := hasher.Hash(password)
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("hashing password: %w", err)
	}

	user := domain.NewUser(id, adminEmail, "Admin", hash, domain.RoleSuperAdmin)

	if err := users.Create(ctx, user); err != nil {
		return BootstrapResult{}, fmt.Errorf("creating superadmin: %w", err)
	}

	return BootstrapResult{
		Created:  true,
		Email:    adminEmail,
		Password: password,
	}, nil
}

// generatePassword creates a cryptographically random password of n bytes,
// encoded as URL-safe base64 (no padding).
func generatePassword(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
