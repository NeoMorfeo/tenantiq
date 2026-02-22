package otel

import (
	"database/sql"
	"fmt"

	"github.com/XSAM/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// OpenDB opens a SQLite database with OpenTelemetry instrumentation.
// The returned *sql.DB has automatic tracing for all SQL operations
// and metrics for the connection pool.
func OpenDB(dataSourceName string) (*sql.DB, error) {
	db, err := otelsql.Open("sqlite", dataSourceName,
		otelsql.WithAttributes(semconv.DBSystemSqlite),
	)
	if err != nil {
		return nil, fmt.Errorf("opening instrumented database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	if _, err := otelsql.RegisterDBStatsMetrics(db,
		otelsql.WithAttributes(semconv.DBSystemSqlite),
	); err != nil {
		return nil, fmt.Errorf("registering db stats metrics: %w", err)
	}

	return db, nil
}
