package app

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGeneratedComponentGraphServesAndDrainsHTTP(t *testing.T) {
	t.Parallel()
	config := HTTPConfigProvider()
	config.Address = "127.0.0.1:0"
	handler := MuxProvider()
	handler.HandleFunc("GET /users/{id}", ControllerProvider().GetUser)
	server := ServerProvider(config, handler)
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://" + server.listener.Addr().String() + "/users/42")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	var body UserResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if body.ID != "42" || body.Message != "hello from Spice" {
		t.Fatalf("response = %#v", body)
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelShutdown()
	if err := server.Stop(shutdownContext); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}
