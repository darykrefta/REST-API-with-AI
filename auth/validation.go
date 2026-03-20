package auth

import (
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"
)

var (
	ErrValidationEmailRequired    = errors.New("email is required")
	ErrValidationEmailInvalid     = errors.New("invalid email address")
	ErrValidationPasswordRequired = errors.New("password is required")
	ErrValidationPasswordTooShort = errors.New("password must be at least 6 characters")
)

// validateEmail trims, lowercases, and parses the address with net/mail.
// Returns the canonical mailbox from addr.Address (lowercased, trimmed).
func validateEmail(raw string) (string, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return "", ErrValidationEmailRequired
	}

	addr, err := mail.ParseAddress(s)
	if err != nil {
		return "", ErrValidationEmailInvalid
	}

	mailbox := strings.TrimSpace(strings.ToLower(addr.Address))
	if mailbox == "" {
		return "", ErrValidationEmailInvalid
	}

	return mailbox, nil
}

// validatePassword rejects whitespace-only passwords and enforces min 6 runes.
// Returns the trimmed password for hashing / comparison.
func validatePassword(raw string) (string, error) {
	pwd := strings.TrimSpace(raw)
	if pwd == "" {
		return "", ErrValidationPasswordRequired
	}
	if utf8.RuneCountInString(pwd) < 6 {
		return "", ErrValidationPasswordTooShort
	}
	return pwd, nil
}

func validatePasswordLogin(raw string) (string, error) {
	pwd := strings.TrimSpace(raw)
	if pwd == "" {
		return "", ErrValidationPasswordRequired
	}
	return pwd, nil
}