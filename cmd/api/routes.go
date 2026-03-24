package main

import (
	"database/sql"
	"net/http"
	"path/filepath"

	"rest.api/auth"
	"rest.api/internal/events"
)

func RegisterAuthRoutes(mux *http.ServeMux, usersController *auth.UsersController) {
	mux.HandleFunc("POST /signup", usersController.Signup)
	mux.HandleFunc("POST /login", usersController.Login)
	mux.HandleFunc("GET /users", usersController.ListUsers)
}

// RegisterEventRoutes wires CRUD and registration routes for events (Go 1.22+ method-aware ServeMux).
// POST, PUT, DELETE, and register/unregister require Authorization: Bearer <JWT>. GET routes are public.
// imagesDir is the absolute path where event images are stored (multipart create/update).
func RegisterEventRoutes(mux *http.ServeMux, db *sql.DB, jwtSecret []byte, imagesDir string) {
	h := &events.Handler{DB: db, ImagesDir: imagesDir}
	mux.HandleFunc("GET /events", h.List)
	mux.HandleFunc("POST /events", auth.RequireJWT(jwtSecret, h.Create))
	mux.HandleFunc("GET /events/{id}", h.Get)
	mux.HandleFunc("POST /events/{id}/register", auth.RequireJWT(jwtSecret, h.Register))
	mux.HandleFunc("DELETE /events/{id}/register", auth.RequireJWT(jwtSecret, h.Unregister))
	mux.HandleFunc("PUT /events/{id}", auth.RequireJWT(jwtSecret, h.Update))
	mux.HandleFunc("DELETE /events/{id}", auth.RequireJWT(jwtSecret, h.Delete))
}

// RegisterStaticImageRoutes serves GET /images/{file} from imagesDir (files created via event multipart).
func RegisterStaticImageRoutes(mux *http.ServeMux, imagesDir string) {
	abs, err := filepath.Abs(imagesDir)
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /images/", http.StripPrefix("/images/", http.FileServer(http.Dir(abs))))
}
