package presentation

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestServerStartsServesAndGracefullyStops(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeContent(
			writer,
			request,
			"ready.txt",
			time.Time{},
			strings.NewReader("ready"),
		)
	})
	server, err := NewServer(ServerSettings{
		Address:           "127.0.0.1:0",
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       time.Second,
	}, mux)
	if err != nil {
		t.Fatal(err)
	}
	if startErr := server.Start(t.Context()); startErr != nil {
		t.Fatal(startErr)
	}
	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"http://"+server.Address(),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		t.Fatalf("read=%v close=%v", readErr, closeErr)
	}
	if response.StatusCode != http.StatusOK ||
		string(body) != "ready" ||
		!strings.Contains(
			response.Header.Get("Content-Security-Policy"),
			"default-src 'self'",
		) {
		t.Fatalf("response = %d %v %q", response.StatusCode, response.Header, body)
	}
	stopContext, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := server.Stop(stopContext); err != nil {
		t.Fatal(err)
	}
	if err := server.Stop(stopContext); err != nil {
		t.Fatalf("idempotent Stop() error = %v", err)
	}
	if err := server.Start(t.Context()); err == nil {
		t.Fatal("Start() after Stop() succeeded")
	}
}

func TestServerRejectsInvalidConfigurationAndLifecycle(t *testing.T) {
	t.Parallel()

	valid := ServerSettings{
		Address:           "127.0.0.1:0",
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       time.Second,
	}
	if _, err := NewServer(ServerSettings{}, http.NewServeMux()); err == nil {
		t.Fatal("empty settings succeeded")
	}
	if _, err := NewServer(valid, nil); err == nil {
		t.Fatal("nil handler succeeded")
	}
	server, err := NewServer(valid, http.NewServeMux())
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if err := server.Start(canceled); err == nil {
		t.Fatal("canceled Start() succeeded")
	}
	if err := server.Stop(nil); err == nil { //nolint:staticcheck // exact nil boundary
		t.Fatal("nil Stop() succeeded")
	}
}
