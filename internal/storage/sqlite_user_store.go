package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"rest.api/auth"
)

type SQLiteUserStore struct {
	db *sql.DB
}

func NewSQLiteUserStore(dbConn *sql.DB) *SQLiteUserStore {
	return &SQLiteUserStore{db: dbConn}
}

func (s *SQLiteUserStore) CreateUser(ctx context.Context, name, email, passwordHash string) (auth.User, error) {
	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO users (name, email, password_hash) VALUES (?, ?, ?)`,
		name,
		email,
		passwordHash,
	)
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "unique") && strings.Contains(lower, "users.email") {
			return auth.User{}, auth.ErrEmailAlreadyExists
		}
		if strings.Contains(lower, "unique") && strings.Contains(lower, "email") {
			return auth.User{}, auth.ErrEmailAlreadyExists
		}
		return auth.User{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return auth.User{}, err
	}

	return auth.User{
		ID:    itoa64(id),
		Name:  name,
		Email: email,
		// PasswordHash intentionally omitted from JSON via struct tag.
		// (It's still present for controller login comparisons.)
		PasswordHash: passwordHash,
	}, nil
}

func (s *SQLiteUserStore) GetUserByEmail(ctx context.Context, email string) (auth.User, error) {
	var (
		id           int64
		name         string
		passwordHash string
	)

	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, email, password_hash FROM users WHERE email = ?`,
		email,
	).Scan(&id, &name, new(string), &passwordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.User{}, auth.ErrUserNotFound
		}
		return auth.User{}, err
	}

	return auth.User{
		ID:           itoa64(id),
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
	}, nil
}

func (s *SQLiteUserStore) ListUsers(ctx context.Context) ([]auth.User, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, email FROM users ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []auth.User
	for rows.Next() {
		var id int64
		var name, email string
		if err := rows.Scan(&id, &name, &email); err != nil {
			return nil, err
		}
		out = append(out, auth.User{
			ID:    itoa64(id),
			Name:  name,
			Email: email,
		})
	}
	return out, rows.Err()
}

func itoa64(v int64) string {
	// Small helper to avoid pulling in fmt for a single conversion.
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := make([]byte, 0, 20)
	for v > 0 {
		d := v % 10
		buf = append([]byte{byte('0' + d)}, buf...)
		v /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
