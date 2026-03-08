package bcrypt

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Cost is the bcrypt work factor. 12 gives ~250ms per hash,
// a good balance between security and responsiveness.
const Cost = 12

// Hasher implements domain.PasswordHasher using bcrypt.
type Hasher struct{}

// New creates a bcrypt-based password hasher.
func New() *Hasher {
	return &Hasher{}
}

// Hash returns the bcrypt hash of the given password.
func (h *Hasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), Cost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return string(bytes), nil
}

// Verify checks whether the given password matches the hash.
// Returns nil on success, an error otherwise.
func (h *Hasher) Verify(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
