package app

import (
	"context"
	"fmt"

	"github.com/neomorfeo/tenantiq/internal/domain"
)

// TenantService orchestrates tenant lifecycle operations.
type TenantService struct {
	repo      domain.TenantRepository
	publisher domain.EventPublisher
}

// NewTenantService creates a service with the given adapters.
func NewTenantService(repo domain.TenantRepository, publisher domain.EventPublisher) *TenantService {
	return &TenantService{
		repo:      repo,
		publisher: publisher,
	}
}

// Create persists a new tenant and publishes a creation event.
func (s *TenantService) Create(ctx context.Context, name, slug, plan string) (domain.Tenant, error) {
	// Check slug uniqueness before creating.
	if _, err := s.repo.GetBySlug(ctx, slug); err == nil {
		return domain.Tenant{}, &domain.SlugConflictError{Slug: slug}
	}

	id, err := generateID()
	if err != nil {
		return domain.Tenant{}, fmt.Errorf("generating tenant id: %w", err)
	}

	tenant := domain.NewTenant(id, name, slug, plan)

	if err := s.repo.Create(ctx, tenant); err != nil {
		return domain.Tenant{}, fmt.Errorf("creating tenant: %w", err)
	}

	if err := s.publisher.Publish(ctx, domain.EventProvisionComplete, tenant); err != nil {
		return domain.Tenant{}, fmt.Errorf("publishing creation event: %w", err)
	}

	return tenant, nil
}

// GetByID returns a tenant by its unique identifier.
func (s *TenantService) GetByID(ctx context.Context, id string) (domain.Tenant, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns tenants matching the given filter.
func (s *TenantService) List(ctx context.Context, filter domain.ListFilter) ([]domain.Tenant, error) {
	return s.repo.List(ctx, filter)
}

// Transition applies a lifecycle event to a tenant, changing its state.
func (s *TenantService) Transition(ctx context.Context, id string, event domain.Event) (domain.Tenant, error) {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Tenant{}, err
	}

	dst, ok := findTransition(tenant.Status, event)
	if !ok {
		return domain.Tenant{}, &domain.TransitionError{
			Event:   event,
			Current: tenant.Status,
		}
	}

	tenant.Status = dst

	if err := s.repo.Update(ctx, tenant); err != nil {
		return domain.Tenant{}, fmt.Errorf("updating tenant: %w", err)
	}

	if err := s.publisher.Publish(ctx, event, tenant); err != nil {
		return domain.Tenant{}, fmt.Errorf("publishing event %q: %w", event, err)
	}

	return tenant, nil
}

// findTransition checks if an event is valid from the current status
// and returns the destination status.
func findTransition(current domain.Status, event domain.Event) (domain.Status, bool) {
	for _, t := range domain.Transitions {
		if t.Event == event && t.Src == current {
			return t.Dst, true
		}
	}
	return "", false
}
