package bcrypt_test

import (
	"testing"

	adapter "github.com/neomorfeo/tenantiq/internal/adapter/bcrypt"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// Verify Hasher satisfies the domain port at compile time.
var _ domain.PasswordHasher = (*adapter.Hasher)(nil)

func TestHash_ProducesDifferentHashes(t *testing.T) {
	h := adapter.New()

	hash1, err := h.Hash("password123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	hash2, err := h.Hash("password123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if hash1 == hash2 {
		t.Error("same password should produce different hashes (unique salt)")
	}
}

func TestVerify_CorrectPassword(t *testing.T) {
	h := adapter.New()

	hash, err := h.Hash("secret")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if err := h.Verify("secret", hash); err != nil {
		t.Errorf("Verify() should succeed for correct password, got %v", err)
	}
}

func TestVerify_WrongPassword(t *testing.T) {
	h := adapter.New()

	hash, err := h.Hash("secret")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if err := h.Verify("wrong", hash); err == nil {
		t.Error("Verify() should fail for wrong password")
	}
}
