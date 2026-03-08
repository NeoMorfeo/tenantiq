package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors for simple conditions without extra context.
var (
	ErrTenantNotFound     = errors.New("tenant not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
)

// SlugConflictError is returned when a tenant slug is already in use.
type SlugConflictError struct {
	Slug string
}

func (e *SlugConflictError) Error() string {
	return fmt.Sprintf("slug %q is already in use", e.Slug)
}

// TransitionError is returned when a state transition is not allowed.
type TransitionError struct {
	Event   Event
	Current Status
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("event %q is not valid from state %q", e.Event, e.Current)
}

// EmailConflictError is returned when a user email is already in use.
type EmailConflictError struct {
	Email string
}

func (e *EmailConflictError) Error() string {
	return fmt.Sprintf("email %q is already in use", e.Email)
}
