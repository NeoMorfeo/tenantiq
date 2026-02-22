package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/neomorfeo/tenantiq/internal/domain"
	"github.com/pressly/goose/v3"

	_ "modernc.org/sqlite" // Register SQLite driver.
)

//go:embed migrations/*.sql
var migrations embed.FS

// TenantRepository implements domain.TenantRepository using SQLite.
type TenantRepository struct {
	db *sql.DB
}

// New opens a SQLite database, runs migrations, and returns a ready repository.
func New(dataSourceName string) (*TenantRepository, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	// Enable foreign keys (off by default in SQLite).
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	return NewFromDB(db)
}

// NewFromDB wraps an existing database connection, runs migrations, and returns a ready repository.
// Use this when the *sql.DB has been pre-configured (e.g., with otelsql instrumentation).
func NewFromDB(db *sql.DB) (*TenantRepository, error) {
	if err := runMigrations(db); err != nil {
		return nil, err
	}

	return &TenantRepository{db: db}, nil
}

// Close closes the underlying database connection.
func (r *TenantRepository) Close() error {
	return r.db.Close()
}

// DB returns the underlying database connection for use by other adapters (e.g., river).
func (r *TenantRepository) DB() *sql.DB {
	return r.db
}

func runMigrations(db *sql.DB) error {
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}

const timeFormat = "2006-01-02T15:04:05Z"

func (r *TenantRepository) Create(ctx context.Context, t domain.Tenant) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tenants (id, name, slug, status, plan, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.Slug, string(t.Status), t.Plan,
		t.CreatedAt.Format(timeFormat),
		t.UpdatedAt.Format(timeFormat),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return &domain.SlugConflictError{Slug: t.Slug}
		}
		return fmt.Errorf("inserting tenant: %w", err)
	}
	return nil
}

func (r *TenantRepository) GetByID(ctx context.Context, id string) (domain.Tenant, error) {
	return r.scanTenant(r.db.QueryRowContext(ctx,
		`SELECT id, name, slug, status, plan, created_at, updated_at
		 FROM tenants WHERE id = ?`, id,
	))
}

func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (domain.Tenant, error) {
	return r.scanTenant(r.db.QueryRowContext(ctx,
		`SELECT id, name, slug, status, plan, created_at, updated_at
		 FROM tenants WHERE slug = ?`, slug,
	))
}

func (r *TenantRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.Tenant, error) {
	query := `SELECT id, name, slug, status, plan, created_at, updated_at FROM tenants`
	var args []any

	if filter.Status != nil {
		query += ` WHERE status = ?`
		args = append(args, string(*filter.Status))
	}

	query += ` ORDER BY created_at DESC`

	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing tenants: %w", err)
	}
	defer rows.Close()

	var tenants []domain.Tenant
	for rows.Next() {
		t, err := r.scanTenantFromRows(rows)
		if err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}

	return tenants, rows.Err()
}

func (r *TenantRepository) Update(ctx context.Context, t domain.Tenant) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE tenants SET name = ?, slug = ?, status = ?, plan = ?, updated_at = ?
		 WHERE id = ?`,
		t.Name, t.Slug, string(t.Status), t.Plan,
		time.Now().UTC().Format(timeFormat), t.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return &domain.SlugConflictError{Slug: t.Slug}
		}
		return fmt.Errorf("updating tenant: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrTenantNotFound
	}

	return nil
}

// scanTenant scans a single row from QueryRow into a domain.Tenant.
func (r *TenantRepository) scanTenant(row *sql.Row) (domain.Tenant, error) {
	var t domain.Tenant
	var status, createdAt, updatedAt string

	err := row.Scan(&t.ID, &t.Name, &t.Slug, &status, &t.Plan, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Tenant{}, domain.ErrTenantNotFound
		}
		return domain.Tenant{}, fmt.Errorf("scanning tenant: %w", err)
	}

	t.Status = domain.Status(status)
	t.CreatedAt, _ = time.Parse(timeFormat, createdAt)
	t.UpdatedAt, _ = time.Parse(timeFormat, updatedAt)

	return t, nil
}

// scanTenantFromRows scans a single row from Rows (used in List).
func (r *TenantRepository) scanTenantFromRows(rows *sql.Rows) (domain.Tenant, error) {
	var t domain.Tenant
	var status, createdAt, updatedAt string

	err := rows.Scan(&t.ID, &t.Name, &t.Slug, &status, &t.Plan, &createdAt, &updatedAt)
	if err != nil {
		return domain.Tenant{}, fmt.Errorf("scanning tenant row: %w", err)
	}

	t.Status = domain.Status(status)
	t.CreatedAt, _ = time.Parse(timeFormat, createdAt)
	t.UpdatedAt, _ = time.Parse(timeFormat, updatedAt)

	return t, nil
}

// isUniqueViolation checks if a SQLite error is a UNIQUE constraint violation.
func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
