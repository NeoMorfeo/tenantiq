package i18n_test

import (
	"context"
	"testing"

	"github.com/neomorfeo/tenantiq/internal/i18n"
)

func TestTranslate_English(t *testing.T) {
	l := i18n.NewLocalizer("en")

	tests := []struct {
		id       string
		data     map[string]string
		expected string
	}{
		{"error.internal", nil, "internal server error"},
		{"error.tenant_not_found", nil, "tenant not found"},
		{"error.user_not_found", nil, "user not found"},
		{"error.invalid_credentials", nil, "invalid email or password"},
		{"error.self_delete", nil, "cannot delete own account"},
		{"error.slug_conflict", map[string]string{"Slug": "acme"}, `slug "acme" is already in use`},
		{"error.email_conflict", map[string]string{"Email": "a@b.com"}, `email "a@b.com" is already in use`},
		{"error.transition_invalid", map[string]string{"Event": "suspend", "Current": "creating"}, `event "suspend" is not valid from state "creating"`},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := i18n.T(l, tt.id, tt.data)
			if got != tt.expected {
				t.Errorf("T(%q) = %q, want %q", tt.id, got, tt.expected)
			}
		})
	}
}

func TestTranslate_Spanish(t *testing.T) {
	l := i18n.NewLocalizer("es")

	tests := []struct {
		id       string
		data     map[string]string
		expected string
	}{
		{"error.internal", nil, "error interno del servidor"},
		{"error.tenant_not_found", nil, "inquilino no encontrado"},
		{"error.user_not_found", nil, "usuario no encontrado"},
		{"error.invalid_credentials", nil, "email o contraseña incorrectos"},
		{"error.self_delete", nil, "no se puede eliminar la propia cuenta"},
		{"error.slug_conflict", map[string]string{"Slug": "acme"}, `el slug "acme" ya está en uso`},
		{"error.email_conflict", map[string]string{"Email": "a@b.com"}, `el email "a@b.com" ya está en uso`},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := i18n.T(l, tt.id, tt.data)
			if got != tt.expected {
				t.Errorf("T(%q) = %q, want %q", tt.id, got, tt.expected)
			}
		})
	}
}

func TestTranslate_FallbackToEnglish(t *testing.T) {
	// Unknown language should fall back to English.
	l := i18n.NewLocalizer("fr")
	got := i18n.T(l, "error.internal", nil)
	if got != "internal server error" {
		t.Errorf("expected English fallback, got %q", got)
	}
}

func TestTranslate_UnknownMessageID(t *testing.T) {
	l := i18n.NewLocalizer("en")
	got := i18n.T(l, "error.nonexistent", nil)
	// Should return the message ID itself as fallback.
	if got != "error.nonexistent" {
		t.Errorf("expected message ID as fallback, got %q", got)
	}
}

func TestLocalizerFromContext(t *testing.T) {
	// Without a localizer in context, should fall back to English.
	l := i18n.Localizer(context.Background())
	got := i18n.T(l, "error.internal", nil)
	if got != "internal server error" {
		t.Errorf("expected English default, got %q", got)
	}

	// With a Spanish localizer injected.
	ctx := i18n.WithLocalizer(context.Background(), i18n.NewLocalizer("es"))
	l = i18n.Localizer(ctx)
	got = i18n.T(l, "error.internal", nil)
	if got != "error interno del servidor" {
		t.Errorf("expected Spanish, got %q", got)
	}
}
