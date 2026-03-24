package events

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ErrNotFound is returned when no event exists for the given id.
var ErrNotFound = errors.New("event not found")

// ErrNotOwner is returned when the authenticated user is not the event creator (wrong user_id on the token).
var ErrNotOwner = errors.New("not the event owner")

// Event is a persisted event row.
type Event struct {
	ID          int64
	Title       string
	Description string
	Address     string
	Date        time.Time
	UserID      int64
	ImageURL    string
}

// CreateEventInput holds fields for creating an event.
type CreateEventInput struct {
	Title       string
	Description string
	Address     string
	Date        time.Time
	UserID      int64
	ImageURL    string
}

// UpdateEventInput holds fields for updating an event (full replace).
type UpdateEventInput struct {
	Title       string
	Description string
	Address     string
	Date        time.Time
	ImageURL    string
}

// CreateEvent inserts a new event and returns it with ID set.
func CreateEvent(ctx context.Context, db *sql.DB, in CreateEventInput) (Event, error) {
	if in.UserID < 1 {
		return Event{}, errors.New("invalid user_id")
	}
	res, err := db.ExecContext(ctx,
		`INSERT INTO events (title, description, address, date, user_id, image_url) VALUES (?, ?, ?, ?, ?, ?)`,
		in.Title, in.Description, in.Address, in.Date.Format(time.RFC3339), in.UserID, nullIfEmpty(in.ImageURL),
	)
	if err != nil {
		return Event{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Event{}, err
	}
	return GetEventByID(ctx, db, id)
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// UpdateEvent replaces fields only if the row belongs to ownerUserID.
func UpdateEvent(ctx context.Context, db *sql.DB, id int64, ownerUserID int64, in UpdateEventInput) (Event, error) {
	if ownerUserID < 1 {
		return Event{}, errors.New("invalid user_id")
	}
	res, err := db.ExecContext(ctx,
		`UPDATE events SET title = ?, description = ?, address = ?, date = ?, image_url = ? WHERE id = ? AND user_id = ?`,
		in.Title, in.Description, in.Address, in.Date.Format(time.RFC3339), nullIfEmpty(in.ImageURL), id, ownerUserID,
	)
	if err != nil {
		return Event{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Event{}, err
	}
	if n > 0 {
		return GetEventByID(ctx, db, id)
	}
	_, err = GetEventByID(ctx, db, id)
	if errors.Is(err, ErrNotFound) {
		return Event{}, ErrNotFound
	}
	if err != nil {
		return Event{}, err
	}
	return Event{}, ErrNotOwner
}

// DeleteEvent removes the row only if it belongs to ownerUserID.
func DeleteEvent(ctx context.Context, db *sql.DB, id int64, ownerUserID int64) error {
	if ownerUserID < 1 {
		return errors.New("invalid user_id")
	}
	res, err := db.ExecContext(ctx, `DELETE FROM events WHERE id = ? AND user_id = ?`, id, ownerUserID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = GetEventByID(ctx, db, id)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return ErrNotOwner
}

// ListEvents returns all events ordered by date ascending, then id.
func ListEvents(ctx context.Context, db *sql.DB) ([]Event, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, title, description, address, date, user_id, image_url FROM events ORDER BY date ASC, id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		ev, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

// GetEventByID returns the event with the given id.
func GetEventByID(ctx context.Context, db *sql.DB, id int64) (Event, error) {
	row := db.QueryRowContext(ctx,
		`SELECT id, title, description, address, date, user_id, image_url FROM events WHERE id = ?`,
		id,
	)
	ev, err := scanEventRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Event{}, ErrNotFound
		}
		return Event{}, err
	}
	return ev, nil
}

func scanEventRow(row *sql.Row) (Event, error) {
	var ev Event
	var dateStr string
	var uid sql.NullInt64
	var img sql.NullString
	if err := row.Scan(&ev.ID, &ev.Title, &ev.Description, &ev.Address, &dateStr, &uid, &img); err != nil {
		return Event{}, err
	}
	if uid.Valid {
		ev.UserID = uid.Int64
	}
	if img.Valid {
		ev.ImageURL = img.String
	}
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return Event{}, err
	}
	ev.Date = t
	return ev, nil
}

func scanEvent(rows *sql.Rows) (Event, error) {
	var ev Event
	var dateStr string
	var uid sql.NullInt64
	var img sql.NullString
	if err := rows.Scan(&ev.ID, &ev.Title, &ev.Description, &ev.Address, &dateStr, &uid, &img); err != nil {
		return Event{}, err
	}
	if uid.Valid {
		ev.UserID = uid.Int64
	}
	if img.Valid {
		ev.ImageURL = img.String
	}
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return Event{}, err
	}
	ev.Date = t
	return ev, nil
}
