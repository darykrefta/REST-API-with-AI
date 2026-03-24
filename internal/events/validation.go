package events

import (
	"errors"
	"strings"
	"time"
)

// validateEventInput trims title, description, and address and ensures they are not
// empty or whitespace-only. It also requires a non-zero parsed date (valid instant in time).
func validateEventInput(in *CreateEventInput) error {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.Address = strings.TrimSpace(in.Address)

	if in.Title == "" {
		return errors.New("title is required")
	}
	if in.Description == "" {
		return errors.New("description is required")
	}
	if in.Address == "" {
		return errors.New("address is required")
	}
	if in.Date.IsZero() {
		return errors.New("date is required and must be a valid RFC3339 timestamp")
	}
	// Reject obviously invalid instants (e.g. year 0 from bad data).
	if in.Date.Year() < 1 {
		return errors.New("date must be a valid calendar date")
	}
	return nil
}

// parseEventDate parses a non-empty RFC3339 date string.
func parseEventDate(raw string) (time.Time, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, errors.New("date is required")
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, errors.New("date must be RFC3339 (e.g. 2026-04-01T18:00:00Z)")
	}
	return t, nil
}
