package events

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// ErrAlreadyRegistered is returned when the user is already signed up for the event.
var ErrAlreadyRegistered = errors.New("already registered for this event")

// ErrNotRegistered is returned when unregister is called but there is no registration row.
var ErrNotRegistered = errors.New("not registered for this event")

// RegisterForEvent records that userID is attending eventID. The event must exist.
func RegisterForEvent(ctx context.Context, db *sql.DB, eventID, userID int64) error {
	if userID < 1 {
		return errors.New("invalid user_id")
	}
	if _, err := GetEventByID(ctx, db, eventID); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO event_registrations (event_id, user_id) VALUES (?, ?)`,
		eventID, userID,
	)
	if err == nil {
		return nil
	}
	if isSQLiteUniqueConstraint(err) {
		return ErrAlreadyRegistered
	}
	return err
}

// UnregisterFromEvent removes userID's registration for eventID. The event must exist.
func UnregisterFromEvent(ctx context.Context, db *sql.DB, eventID, userID int64) error {
	if userID < 1 {
		return errors.New("invalid user_id")
	}
	if _, err := GetEventByID(ctx, db, eventID); err != nil {
		return err
	}
	res, err := db.ExecContext(ctx,
		`DELETE FROM event_registrations WHERE event_id = ? AND user_id = ?`,
		eventID, userID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotRegistered
	}
	return nil
}

func isSQLiteUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") || strings.Contains(msg, "constraint failed")
}
