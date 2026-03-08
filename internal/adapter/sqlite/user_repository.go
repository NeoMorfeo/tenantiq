package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/neomorfeo/tenantiq/internal/domain"
)

// UserRepository implements domain.UserRepository using SQLite.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository wraps an existing database connection.
// Migrations must have been run already (by TenantRepository or runMigrations).
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u domain.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, name, password_hash, role, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Email, u.Name, u.PasswordHash, string(u.Role),
		u.CreatedAt.Format(timeFormat),
		u.UpdatedAt.Format(timeFormat),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return &domain.EmailConflictError{Email: u.Email}
		}
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx,
		`SELECT id, email, name, password_hash, role, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	))
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx,
		`SELECT id, email, name, password_hash, role, created_at, updated_at
		 FROM users WHERE email = ?`, email,
	))
}

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, email, name, password_hash, role, created_at, updated_at
		 FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		u, err := r.scanUserFromRows(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, rows.Err()
}

func (r *UserRepository) Update(ctx context.Context, u domain.User) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET email = ?, name = ?, password_hash = ?, role = ?, updated_at = ?
		 WHERE id = ?`,
		u.Email, u.Name, u.PasswordHash, string(u.Role),
		time.Now().UTC().Format(timeFormat), u.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return &domain.EmailConflictError{Email: u.Email}
		}
		return fmt.Errorf("updating user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting users: %w", err)
	}
	return count, nil
}

func (r *UserRepository) scanUser(row *sql.Row) (domain.User, error) {
	var u domain.User
	var role, createdAt, updatedAt string

	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &role, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("scanning user: %w", err)
	}

	u.Role = domain.Role(role)
	u.CreatedAt, _ = time.Parse(timeFormat, createdAt)
	u.UpdatedAt, _ = time.Parse(timeFormat, updatedAt)

	return u, nil
}

func (r *UserRepository) scanUserFromRows(rows *sql.Rows) (domain.User, error) {
	var u domain.User
	var role, createdAt, updatedAt string

	err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &role, &createdAt, &updatedAt)
	if err != nil {
		return domain.User{}, fmt.Errorf("scanning user row: %w", err)
	}

	u.Role = domain.Role(role)
	u.CreatedAt, _ = time.Parse(timeFormat, createdAt)
	u.UpdatedAt, _ = time.Parse(timeFormat, updatedAt)

	return u, nil
}
