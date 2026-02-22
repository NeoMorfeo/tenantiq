package fsm_test

import (
	"context"
	"errors"
	"testing"

	adapter "github.com/neomorfeo/tenantiq/internal/adapter/fsm"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

func TestValidator_AllTransitions(t *testing.T) {
	v := adapter.New()
	ctx := context.Background()

	for _, tr := range domain.Transitions {
		dst, err := v.Apply(ctx, tr.Src, tr.Event)
		if err != nil {
			t.Errorf("Apply(%q, %q) unexpected error: %v", tr.Src, tr.Event, err)
			continue
		}
		if dst != tr.Dst {
			t.Errorf("Apply(%q, %q) = %q, want %q", tr.Src, tr.Event, dst, tr.Dst)
		}
	}
}

func TestValidator_InvalidTransition(t *testing.T) {
	v := adapter.New()
	ctx := context.Background()

	// Can't suspend from "creating" state.
	_, err := v.Apply(ctx, domain.StatusCreating, domain.EventSuspend)
	var trErr *domain.TransitionError
	if !errors.As(err, &trErr) {
		t.Fatalf("expected TransitionError, got %v", err)
	}
	if trErr.Event != domain.EventSuspend {
		t.Errorf("event = %q, want %q", trErr.Event, domain.EventSuspend)
	}
	if trErr.Current != domain.StatusCreating {
		t.Errorf("current = %q, want %q", trErr.Current, domain.StatusCreating)
	}
}

func TestValidator_FullLifecycle(t *testing.T) {
	v := adapter.New()
	ctx := context.Background()

	steps := []struct {
		from  domain.Status
		event domain.Event
		want  domain.Status
	}{
		{domain.StatusCreating, domain.EventProvisionComplete, domain.StatusActive},
		{domain.StatusActive, domain.EventSuspend, domain.StatusSuspended},
		{domain.StatusSuspended, domain.EventReactivate, domain.StatusActive},
		{domain.StatusActive, domain.EventDelete, domain.StatusDeleting},
		{domain.StatusDeleting, domain.EventDeletionComplete, domain.StatusDeleted},
	}

	for _, step := range steps {
		got, err := v.Apply(ctx, step.from, step.event)
		if err != nil {
			t.Fatalf("Apply(%q, %q) error: %v", step.from, step.event, err)
		}
		if got != step.want {
			t.Errorf("Apply(%q, %q) = %q, want %q", step.from, step.event, got, step.want)
		}
	}
}

func TestValidator_DeleteFromSuspended(t *testing.T) {
	v := adapter.New()
	ctx := context.Background()

	// Delete is valid from both "active" and "suspended".
	got, err := v.Apply(ctx, domain.StatusSuspended, domain.EventDelete)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != domain.StatusDeleting {
		t.Errorf("got %q, want %q", got, domain.StatusDeleting)
	}
}
