package db

import (
	"database/sql"
)

func Migrate(dbConn *sql.DB) error {
	// A small set of schema needed for auth.
	_, err := dbConn.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}
