package auth

import (
	"context"
	"errors"
)

var (
	ErrEmailAlreadyExists = errors.New("email already registered")
	ErrUserNotFound       = errors.New("user not found")
)

// UserStore abstracts persistence for the UsersController.
// The controller handles password hashing/verification, the store only writes/reads hashes.
type UserStore interface {
	CreateUser(ctx context.Context, name, email, passwordHash string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
}
