package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// UserController serves the example user API.
//
// @Controller(prefix="/users")
type UserController struct{}

// GetUser returns a user-shaped example response.
//
// @Get(path="/{id}")
func (UserController) GetUser(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(map[string]string{
		"id":      request.PathValue("id"),
		"message": "hello from Spice",
	}); err != nil {
		log.Printf("encode user response: %v", err)
	}
}

func newHandler() http.Handler {
	controller := UserController{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", controller.GetUser)
	return mux
}

func main() {
	if err := run(); err != nil {
		log.Printf("Spice example failed: %v", err)
		os.Exit(1) // Entrypoint exception: return a non-zero status when the server cannot run.
	}
}

func run() error {
	listen := flag.String("listen", ":8080", "HTTP listen address")
	check := flag.Bool("check", false, "build the application, print its route, and exit")
	flag.Parse()

	if *check {
		if _, err := fmt.Fprintln(os.Stdout, "Spice example ready: GET /users/{id}"); err != nil {
			return fmt.Errorf("write readiness message: %w", err)
		}
		return nil
	}

	log.Printf("Spice example listening on %s", *listen)
	server := &http.Server{
		Addr:              *listen,
		Handler:           newHandler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}
