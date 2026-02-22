package river_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	goriver "github.com/riverqueue/river"

	_ "modernc.org/sqlite"

	riveradapter "github.com/neomorfeo/tenantiq/internal/adapter/river"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := t.TempDir() + "/river_test.db"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		t.Fatalf("setting WAL: %v", err)
	}

	return db
}

func setupClient(t *testing.T, db *sql.DB) *riveradapter.Client {
	t.Helper()

	client, err := riveradapter.Setup(context.Background(), db)
	if err != nil {
		t.Fatalf("river setup: %v", err)
	}

	return client
}

func TestPublisher_Publish_EnqueuesJob(t *testing.T) {
	db := setupTestDB(t)
	client := setupClient(t, db)
	ctx := context.Background()

	// Subscribe to job completions before starting so we don't miss events.
	subscribeChan, subscribeCancel := client.Subscribe(goriver.EventKindJobCompleted)
	defer subscribeCancel()

	if err := client.Start(ctx); err != nil {
		t.Fatalf("river start: %v", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Stop(stopCtx); err != nil {
			t.Errorf("river stop: %v", err)
		}
	})

	pub := riveradapter.NewPublisher(client)
	tenant := domain.NewTenant("t-1", "Acme", "acme", "free")

	if err := pub.Publish(ctx, domain.EventProvisionComplete, tenant); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait for the worker to process the job.
	select {
	case event := <-subscribeChan:
		if event.Job.Kind != "event.published" {
			t.Errorf("job kind = %q, want %q", event.Job.Kind, "event.published")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for job completion")
	}
}

func TestPublisher_Publish_PreservesEventData(t *testing.T) {
	db := setupTestDB(t)
	client := setupClient(t, db)
	ctx := context.Background()

	subscribeChan, subscribeCancel := client.Subscribe(goriver.EventKindJobCompleted)
	defer subscribeCancel()

	if err := client.Start(ctx); err != nil {
		t.Fatalf("river start: %v", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Stop(stopCtx); err != nil {
			t.Errorf("river stop: %v", err)
		}
	})

	pub := riveradapter.NewPublisher(client)
	tenant := domain.NewTenant("t-42", "Test Corp", "test-corp", "pro")

	if err := pub.Publish(ctx, domain.EventSuspend, tenant); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	select {
	case event := <-subscribeChan:
		// Verify the job carried the right args by checking the encoded JSON.
		args := event.Job.EncodedArgs
		if args == nil {
			t.Fatal("expected encoded args, got nil")
		}
		// The args are stored as JSON; verify key fields are present.
		argsStr := string(args)
		for _, want := range []string{`"event":"suspend"`, `"tenant_id":"t-42"`, `"slug":"test-corp"`, `"plan":"pro"`} {
			if !strings.Contains(argsStr, want) {
				t.Errorf("encoded args missing %s, got: %s", want, argsStr)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for job completion")
	}
}
