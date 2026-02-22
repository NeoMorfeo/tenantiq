package domain

import "time"

// Status represents the lifecycle state of a tenant.
type Status string

const (
	StatusCreating  Status = "creating"
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusDeleting  Status = "deleting"
	StatusDeleted   Status = "deleted"
)

// Event represents an action that triggers a state transition.
type Event string

const (
	EventProvisionComplete Event = "provision_complete"
	EventSuspend           Event = "suspend"
	EventReactivate        Event = "reactivate"
	EventDelete            Event = "delete"
	EventDeletionComplete  Event = "deletion_complete"
)

// Transition defines a valid state change: an event moves a tenant from Src to Dst.
type Transition struct {
	Event Event
	Src   Status
	Dst   Status
}

// Transitions defines all valid state changes in the tenant lifecycle.
// This is domain knowledge consumed by the FSM adapter.
var Transitions = []Transition{
	{Event: EventProvisionComplete, Src: StatusCreating, Dst: StatusActive},
	{Event: EventSuspend, Src: StatusActive, Dst: StatusSuspended},
	{Event: EventReactivate, Src: StatusSuspended, Dst: StatusActive},
	{Event: EventDelete, Src: StatusActive, Dst: StatusDeleting},
	{Event: EventDelete, Src: StatusSuspended, Dst: StatusDeleting},
	{Event: EventDeletionComplete, Src: StatusDeleting, Dst: StatusDeleted},
}

// Tenant is the core domain entity representing an organization using the platform.
type Tenant struct {
	ID        string
	Name      string
	Slug      string
	Status    Status
	Plan      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewTenant creates a tenant in the initial "creating" state.
func NewTenant(id, name, slug, plan string) Tenant {
	now := time.Now().UTC()
	return Tenant{
		ID:        id,
		Name:      name,
		Slug:      slug,
		Status:    StatusCreating,
		Plan:      plan,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
