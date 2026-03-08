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

// UserRepository defines the persistence contract for users.
type UserRepository interface {
	Create(ctx context.Context, user User) error
	GetByID(ctx context.Context, id string) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	List(ctx context.Context) ([]User, error)
	Update(ctx context.Context, user User) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
}

// PasswordHasher defines the contract for password hashing.
// Isolated as a port so the algorithm (bcrypt, argon2, etc.)
// can be swapped without touching business logic.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) error
}

// TokenService defines the contract for authentication token operations.
// The implementation chooses the token format (JWT, PASETO, etc.).
type TokenService interface {
	GenerateAccess(user User) (string, error)
	GenerateRefresh(user User) (string, error)
	ValidateAccess(token string) (TokenClaims, error)
	ValidateRefresh(token string) (TokenClaims, error)
}
