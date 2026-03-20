package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"rest.api/auth"
	"rest.api/internal/db"
	"rest.api/internal/storage"
)

func main() {
	srv := NewServer()
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", srv))
}

func NewServer() http.Handler {
	// Store persistent state in a local SQLite file.
	dbConn, err := db.OpenSQLite("file:app.db?_pragma=busy_timeout(5000)")
	if err != nil {
		panic(err)
	}

	if err := db.Migrate(dbConn); err != nil {
		panic(err)
	}

	jwtSecret := []byte(strings.TrimSpace(os.Getenv("JWT_SECRET")))
	if len(jwtSecret) == 0 {
		log.Println("warning: JWT_SECRET not set; login will not return a token")
	}

	jwtTTL := 24 * time.Hour
	if s := strings.TrimSpace(os.Getenv("JWT_TTL")); s != "" {
		d, err := time.ParseDuration(s)
		if err != nil {
			log.Printf("warning: invalid JWT_TTL %q, using 24h", s)
		} else {
			jwtTTL = d
		}
	}
	if len(jwtSecret) > 0 && jwtTTL <= 0 {
		log.Println("warning: JWT_TTL must be positive, using 24h")
		jwtTTL = 24 * time.Hour
	}

	userStore := storage.NewSQLiteUserStore(dbConn)
	usersController := auth.NewUsersController(userStore, jwtSecret, jwtTTL)

	mux := http.NewServeMux()

	RegisterAuthRoutes(mux, usersController)

	return mux
}
