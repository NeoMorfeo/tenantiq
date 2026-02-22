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

// TransitionValidator checks if a state transition is valid and returns
// the destination status. Implementations may use an FSM library or
// any other mechanism to enforce the rules defined in Transitions.
type TransitionValidator interface {
	Apply(ctx context.Context, current Status, event Event) (Status, error)
}
