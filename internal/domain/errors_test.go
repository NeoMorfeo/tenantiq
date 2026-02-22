package domain_test

import (
	"testing"

	"github.com/neomorfeo/tenantiq/internal/domain"
)

func TestSlugConflictError_Error(t *testing.T) {
	err := &domain.SlugConflictError{Slug: "acme"}
	want := `slug "acme" is already in use`
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestTransitionError_Error(t *testing.T) {
	err := &domain.TransitionError{
		Event:   domain.EventSuspend,
		Current: domain.StatusCreating,
	}
	want := `event "suspend" is not valid from state "creating"`
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
