package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

const defaultHTTPAddress = "127.0.0.1:8080"

// HTTPConfig contains the safe transport defaults used by the example.
type HTTPConfig struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

// HTTPConfigProvider loads the example's typed HTTP configuration.
//
// @Bean
func HTTPConfigProvider() HTTPConfig {
	address := os.Getenv("SPICE_EXAMPLE_ADDRESS")
	if address == "" {
		address = defaultHTTPAddress
	}
	return HTTPConfig{
		Address:           address,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// Server owns the example's HTTP listener and graceful-drain lifecycle.
type Server struct {
	httpServer  *http.Server
	serveErrors chan error
	listener    net.Listener
}

// ServerProvider constructs the lifecycle-managed HTTP server.
//
// @Bean
func ServerProvider(config HTTPConfig, handler *http.ServeMux) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              config.Address,
			Handler:           handler,
			ReadHeaderTimeout: config.ReadHeaderTimeout,
			ReadTimeout:       config.ReadTimeout,
			WriteTimeout:      config.WriteTimeout,
			IdleTimeout:       config.IdleTimeout,
		},
		serveErrors: make(chan error, 1),
	}
}

// Start binds the configured HTTP listener.
//
// @OnStart
func (server *Server) Start(ctx context.Context) error {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", server.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", server.httpServer.Addr, err)
	}
	server.listener = listener
	go func() {
		server.serveErrors <- server.httpServer.Serve(listener)
	}()
	return nil
}

// Stop gracefully drains the HTTP server and observes its serve result.
//
// @OnStop
func (server *Server) Stop(ctx context.Context) error {
	shutdownErr := server.httpServer.Shutdown(ctx)
	var serveErr error
	select {
	case serveErr = <-server.serveErrors:
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
	case <-ctx.Done():
		serveErr = ctx.Err()
	}
	return errors.Join(shutdownErr, serveErr)
}
