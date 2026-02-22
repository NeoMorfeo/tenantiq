package river

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riversqlite"
	"github.com/riverqueue/river/rivermigrate"
)

// Setup creates a River client with the event worker registered and runs
// River's internal migrations. The caller must call client.Start() to begin
// processing jobs and client.Stop() for graceful shutdown.
func Setup(ctx context.Context, db *sql.DB) (*Client, error) {
	driver := riversqlite.New(db)

	// Run River's own migrations (creates river_job, river_leader, etc.).
	// These are separate from the app's goose migrations.
	migrator, err := rivermigrate.New(driver, nil)
	if err != nil {
		return nil, fmt.Errorf("creating river migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return nil, fmt.Errorf("running river migrations: %w", err)
	}

	workers := river.NewWorkers()
	river.AddWorker(workers, &EventWorker{})

	client, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 2},
		},
		Workers: workers,
	})
	if err != nil {
		return nil, fmt.Errorf("creating river client: %w", err)
	}

	return client, nil
}
