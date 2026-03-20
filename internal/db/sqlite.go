package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func OpenSQLite(dsn string) (*sql.DB, error) {
	// modernc.org/sqlite is a pure-Go driver, avoiding CGO.
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Ensure the connection is actually usable.
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
