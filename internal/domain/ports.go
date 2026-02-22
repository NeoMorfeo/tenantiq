package domain

import "context"

// TenantRepository defines the persistence contract for tenants.
type TenantRepository interface {
	Create(ctx context.Context, tenant Tenant) error
	GetByID(ctx context.Context, id string) (Tenant, error)
	GetBySlug(ctx context.Context, slug string) (Tenant, error)
	List(ctx context.Context, filter ListFilter) ([]Tenant, error)
	Update(ctx context.Context, tenant Tenant) error
}

// ListFilter holds optional criteria for listing tenants.
type ListFilter struct {
	Status *Status
	Limit  int
	Offset int
}

// EventPublisher defines the contract for emitting domain events.
type EventPublisher interface {
	Publish(ctx context.Context, event Event, tenant Tenant) error
}
