package river

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/riverqueue/river"

	"github.com/neomorfeo/tenantiq/internal/domain"
)

// Compile-time check: Publisher implements domain.EventPublisher.
var _ domain.EventPublisher = (*Publisher)(nil)

// EventJobArgs carries the data needed to process a domain event asynchronously.
// River serializes this as JSON into its job queue table. It includes a snapshot
// of the tenant at the time the event was published, so the worker never needs
// to query the database.
type EventJobArgs struct {
	Event    string `json:"event"`
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Status   string `json:"status"`
	Plan     string `json:"plan"`
}

// Kind returns the unique job type identifier used by River's job routing.
func (EventJobArgs) Kind() string { return "event.published" }

// Client is the River client type parameterized for SQLite (*sql.Tx).
type Client = river.Client[*sql.Tx]

// Publisher implements domain.EventPublisher by enqueuing River jobs.
type Publisher struct {
	client *Client
}

// NewPublisher creates a publisher backed by the given River client.
func NewPublisher(client *Client) *Publisher {
	return &Publisher{client: client}
}

// Publish enqueues a domain event as an async job in River.
func (p *Publisher) Publish(ctx context.Context, event domain.Event, tenant domain.Tenant) error {
	_, err := p.client.Insert(ctx, EventJobArgs{
		Event:    string(event),
		TenantID: tenant.ID,
		Name:     tenant.Name,
		Slug:     tenant.Slug,
		Status:   string(tenant.Status),
		Plan:     tenant.Plan,
	}, nil)
	if err != nil {
		return fmt.Errorf("enqueuing event job: %w", err)
	}
	return nil
}
