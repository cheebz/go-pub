package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	// Get configuration
	ENV := os.Getenv("ENV")
	if ENV == "" {
		ENV = "dev"
	}
	fmt.Println(fmt.Sprintf("Running in ENV: %s", ENV))
	config = getConfig(ENV)

	db = connectDb(config.Db)
	defer db.Close()

	// Init router
	r := mux.NewRouter()
	r.HandleFunc("/.well-known/webfinger", getWebFinger)
	r.HandleFunc("/users/{name:[[:alnum:]]+}", getUser)
	r.HandleFunc("/users/{name:[[:alnum:]]+}/inbox", getInbox)
	r.HandleFunc("/users/{name:[[:alnum:]]+}/outbox", getOutbox)
	r.HandleFunc("/users/{name:[[:alnum:]]+}/following", getFollowing)
	r.HandleFunc("/users/{name:[[:alnum:]]+}/followers", getFollowers)
	r.HandleFunc("/users/{name:[[:alnum:]]+}/liked", getLiked)

	// Static files
	r.PathPrefix("/files/").Handler(http.StripPrefix("/files/", http.FileServer(http.Dir("./static/"))))
	// r.Use(jwtMiddleware)

	// Run server
	port := config.Port
	fmt.Println(fmt.Sprintf("Serving on port %d", port))

	// CORS in dev
	if ENV == "dev" {
		cors := cors.New(cors.Options{
			AllowedOrigins:   []string{"http://localhost:4200", "http://127.0.0.1:4200"},
			AllowCredentials: true,
		})
		r.Use(cors.Handler)
	}

	// TLS
	if config.SSLCert == "" {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
	}
	log.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%d", port), config.SSLCert, config.SSLKey, r))
}
