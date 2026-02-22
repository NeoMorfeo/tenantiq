package river

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

// EventWorker processes domain event jobs from the River queue.
// For now it logs the event; future versions will dispatch to
// webhooks, provisioning logic, or notification systems.
type EventWorker struct {
	river.WorkerDefaults[EventJobArgs]
}

// Work processes a single event job.
func (w *EventWorker) Work(ctx context.Context, job *river.Job[EventJobArgs]) error {
	slog.InfoContext(ctx, "processing event",
		"event", job.Args.Event,
		"tenant_id", job.Args.TenantID,
		"tenant_slug", job.Args.Slug,
		"job_id", job.ID,
		"attempt", job.Attempt,
	)
	return nil
}
