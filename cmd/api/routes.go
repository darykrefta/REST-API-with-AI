package main

import (
	"net/http"

	"rest.api/auth"
)

func RegisterAuthRoutes(mux *http.ServeMux, usersController *auth.UsersController) {
	mux.HandleFunc("POST /signup", usersController.Signup)
	mux.HandleFunc("POST /login", usersController.Login)
}
