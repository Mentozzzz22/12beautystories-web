// Package apperr defines errors shared by services and mapped to HTTP responses.
package apperr

import (
	"errors"
	"fmt"
)

// Sentinel errors mapped to 404 and 409.
var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// Field error codes understood by the web app.
const (
	Required = "required"
	Invalid  = "invalid"
	TooLong  = "too_long"
	Taken    = "taken"
)

// ValidationError lists invalid input fields with machine-readable codes.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid fields: %d", len(e.Fields))
}

// Fields collects field errors; Err returns nil when nothing was added.
type Fields map[string]string

// Add records a field error unless one is already set for that field.
func (f Fields) Add(field, code string) {
	if _, ok := f[field]; !ok {
		f[field] = code
	}
}

// Err converts the collected errors into a *ValidationError or nil.
func (f Fields) Err() error {
	if len(f) == 0 {
		return nil
	}
	return &ValidationError{Fields: f}
}

// Field returns a ValidationError for a single field.
func Field(field, code string) error {
	return &ValidationError{Fields: map[string]string{field: code}}
}
