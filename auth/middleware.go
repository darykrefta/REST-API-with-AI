package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// BearerToken returns the token from Authorization: Bearer <token>.
func BearerToken(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", errors.New("missing authorization header")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return "", errors.New("invalid authorization scheme")
	}
	tok := strings.TrimSpace(strings.TrimPrefix(h, prefix))
	if tok == "" {
		return "", errors.New("missing token")
	}
	return tok, nil
}

// RequireJWT wraps a handler: valid Bearer JWT required. GET-style public routes should not use this.
func RequireJWT(secret []byte, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(secret) == 0 {
			writeAuthJSON(w, http.StatusServiceUnavailable, "authentication is not configured")
			return
		}
		raw, err := BearerToken(r)
		if err != nil {
			writeAuthJSON(w, http.StatusUnauthorized, "missing or invalid authorization")
			return
		}
		userID, email, err := VerifyUserJWT(raw, secret)
		if err != nil {
			if errors.Is(err, ErrJWTExpired) {
				writeAuthJSON(w, http.StatusUnauthorized, "token expired")
				return
			}
			writeAuthJSON(w, http.StatusUnauthorized, "missing or invalid authorization")
			return
		}
		ctx := WithJWTUser(r.Context(), userID, email)
		next(w, r.WithContext(ctx))
	}
}

func writeAuthJSON(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
