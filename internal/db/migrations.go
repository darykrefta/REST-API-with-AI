package db

import (
	"database/sql"
	"strings"
)

func Migrate(dbConn *sql.DB) error {
	_, err := dbConn.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}

	_, err = dbConn.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			description TEXT NOT NULL,
			address TEXT NOT NULL,
			date TEXT NOT NULL,
			user_id INTEGER NOT NULL REFERENCES users(id),
			image_url TEXT
		);
	`)
	if err != nil {
		return err
	}

	// Existing databases may have been created without user_id on events.
	_, alterErr := dbConn.Exec(`ALTER TABLE events ADD COLUMN user_id INTEGER REFERENCES users(id)`)
	if alterErr != nil && !strings.Contains(strings.ToLower(alterErr.Error()), "duplicate column") {
		return alterErr
	}

	_, err = dbConn.Exec(`
		CREATE TABLE IF NOT EXISTS event_registrations (
			event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id),
			registered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (event_id, user_id)
		);
	`)
	if err != nil {
		return err
	}

	_, alterImg := dbConn.Exec(`ALTER TABLE events ADD COLUMN image_url TEXT`)
	if alterImg != nil && !strings.Contains(strings.ToLower(alterImg.Error()), "duplicate column") {
		return alterImg
	}

	return nil
}
