package auth

import "context"

type jwtContextKey int

const (
	jwtKeyUserID jwtContextKey = iota
	jwtKeyUserEmail
)

// WithJWTUser returns a context carrying the authenticated user's id and email (from JWT).
func WithJWTUser(ctx context.Context, userID, email string) context.Context {
	ctx = context.WithValue(ctx, jwtKeyUserID, userID)
	return context.WithValue(ctx, jwtKeyUserEmail, email)
}

// UserIDFromContext returns the JWT subject user id string when RequireJWT ran successfully.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(jwtKeyUserID).(string)
	return v, ok && v != ""
}

// UserEmailFromContext returns the JWT email claim when present.
func UserEmailFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(jwtKeyUserEmail).(string)
	return v, ok
}
