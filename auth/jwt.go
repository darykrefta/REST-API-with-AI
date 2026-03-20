package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrJWTEmptySecret = errors.New("jwt secret is required")
	ErrJWTInvalid     = errors.New("invalid token")
	ErrJWTExpired     = errors.New("token expired")
)

// UserClaims is the JWT payload for authenticated users.
type UserClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateUserJWT signs an HS256 JWT with user_id and email claims.
func GenerateUserJWT(userID, email string, secret []byte, ttl time.Duration) (string, error) {
	if len(secret) == 0 {
		return "", ErrJWTEmptySecret
	}
	if ttl <= 0 {
		return "", errors.New("jwt ttl must be positive")
	}

	now := time.Now()
	claims := UserClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// VerifyUserJWT parses and validates an HS256 JWT and returns user_id and email from claims.
func VerifyUserJWT(tokenString string, secret []byte) (userID, email string, err error) {
	if len(secret) == 0 {
		return "", "", ErrJWTEmptySecret
	}

	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrJWTInvalid
		}
		return secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", "", ErrJWTExpired
		}
		return "", "", ErrJWTInvalid
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return "", "", ErrJWTInvalid
	}

	return claims.UserID, claims.Email, nil
}
