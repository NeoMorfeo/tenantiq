package domain

import "time"

// Role represents the access level of a user.
type Role string

const (
	RoleSuperAdmin Role = "superadmin"
	RoleAdmin      Role = "admin"
	RoleViewer     Role = "viewer"
)

// User represents an authenticated user in the system.
type User struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewUser creates a user with the given attributes.
func NewUser(id, email, name, passwordHash string, role Role) User {
	now := time.Now().UTC()
	return User{
		ID:           id,
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// IsValidRole reports whether r is a recognized role.
func IsValidRole(r Role) bool {
	switch r {
	case RoleSuperAdmin, RoleAdmin, RoleViewer:
		return true
	}
	return false
}

// TokenClaims represents the identity extracted from an authentication token.
// This is domain knowledge — it defines *who* is making a request,
// independent of the token mechanism (JWT, PASETO, etc.).
type TokenClaims struct {
	UserID string
	Email  string
	Role   Role
}
