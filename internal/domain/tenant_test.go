package domain_test

import (
	"testing"
	"time"

	"github.com/neomorfeo/tenantiq/internal/domain"
)

func TestNewTenant(t *testing.T) {
	before := time.Now().UTC()
	tenant := domain.NewTenant("id-1", "Acme Corp", "acme-corp", "pro")
	after := time.Now().UTC()

	if tenant.ID != "id-1" {
		t.Errorf("ID = %q, want %q", tenant.ID, "id-1")
	}
	if tenant.Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", tenant.Name, "Acme Corp")
	}
	if tenant.Slug != "acme-corp" {
		t.Errorf("Slug = %q, want %q", tenant.Slug, "acme-corp")
	}
	if tenant.Status != domain.StatusCreating {
		t.Errorf("Status = %q, want %q", tenant.Status, domain.StatusCreating)
	}
	if tenant.Plan != "pro" {
		t.Errorf("Plan = %q, want %q", tenant.Plan, "pro")
	}
	if tenant.CreatedAt.Before(before) || tenant.CreatedAt.After(after) {
		t.Errorf("CreatedAt = %v, want between %v and %v", tenant.CreatedAt, before, after)
	}
	if tenant.UpdatedAt != tenant.CreatedAt {
		t.Errorf("UpdatedAt should equal CreatedAt on new tenant")
	}
}

func TestTransitions_AllEventsHaveEntries(t *testing.T) {
	events := []domain.Event{
		domain.EventProvisionComplete,
		domain.EventSuspend,
		domain.EventReactivate,
		domain.EventDelete,
		domain.EventDeletionComplete,
	}

	for _, event := range events {
		found := false
		for _, tr := range domain.Transitions {
			if tr.Event == event {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("event %q has no transition defined", event)
		}
	}
}

func TestTransitions_ValidPaths(t *testing.T) {
	// Walk the full happy path: creating → active → suspended → active → deleting → deleted
	cases := []struct {
		event domain.Event
		src   domain.Status
		dst   domain.Status
	}{
		{domain.EventProvisionComplete, domain.StatusCreating, domain.StatusActive},
		{domain.EventSuspend, domain.StatusActive, domain.StatusSuspended},
		{domain.EventReactivate, domain.StatusSuspended, domain.StatusActive},
		{domain.EventDelete, domain.StatusActive, domain.StatusDeleting},
		{domain.EventDeletionComplete, domain.StatusDeleting, domain.StatusDeleted},
		// Also: delete from suspended
		{domain.EventDelete, domain.StatusSuspended, domain.StatusDeleting},
	}

	for _, tc := range cases {
		found := false
		for _, tr := range domain.Transitions {
			if tr.Event == tc.event && tr.Src == tc.src && tr.Dst == tc.dst {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing transition: %q from %q → %q", tc.event, tc.src, tc.dst)
		}
	}
}

func TestTransitions_InvalidPaths(t *testing.T) {
	// These transitions must NOT exist.
	invalid := []struct {
		event domain.Event
		src   domain.Status
	}{
		{domain.EventSuspend, domain.StatusCreating},
		{domain.EventReactivate, domain.StatusCreating},
		{domain.EventReactivate, domain.StatusActive},
		{domain.EventProvisionComplete, domain.StatusActive},
		{domain.EventDelete, domain.StatusCreating},
		{domain.EventDelete, domain.StatusDeleted},
	}

	for _, tc := range invalid {
		for _, tr := range domain.Transitions {
			if tr.Event == tc.event && tr.Src == tc.src {
				t.Errorf("unexpected transition: %q from %q should not exist", tc.event, tc.src)
			}
		}
	}
}
