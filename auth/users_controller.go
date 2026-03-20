package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type UsersController struct {
	store     UserStore
	jwtSecret []byte
	jwtTTL    time.Duration
}

func NewUsersController(store UserStore, jwtSecret []byte, jwtTTL time.Duration) *UsersController {
	return &UsersController{store: store, jwtSecret: jwtSecret, jwtTTL: jwtTTL}
}

func (c *UsersController) Signup(w http.ResponseWriter, r *http.Request) {
	var input SignupInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	email, err := validateEmail(input.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	password, err := validatePassword(input.Password)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}

	user, err := c.store.CreateUser(r.Context(), input.Name, email, string(passwordHash))
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "user registered successfully",
		"user": map[string]string{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
	})
}

func (c *UsersController) Login(w http.ResponseWriter, r *http.Request) {
	var input LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	email, err := validateEmail(input.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	password, err := validatePasswordLogin(input.Password)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	user, err := c.store.GetUserByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to login"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	resp := map[string]any{
		"message": "login successful",
		"user": map[string]string{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
	}

	if len(c.jwtSecret) > 0 {
		token, err := GenerateUserJWT(user.ID, user.Email, c.jwtSecret, c.jwtTTL)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to issue token"})
			return
		}
		resp["token"] = token
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
