package presentation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
)

// @import { Bean } from "github.com/StevenBuglione/spice/annotation/core"
// @import { OnStart, OnStop } from "github.com/StevenBuglione/spice/annotation/lifecycle"

// Server owns the Petclinic HTTP listener and graceful-drain lifecycle.
type Server struct {
	mu          sync.RWMutex
	httpServer  *http.Server
	serveErrors chan error
	listener    net.Listener
	started     bool
	stopping    bool
	completed   bool
	stopDone    chan struct{}
	stopErr     error
}

// NewServer constructs a lifecycle-managed server with bounded timeouts.
//
// @Bean
func NewServer(
	settings ServerSettings,
	handler *http.ServeMux,
) (*Server, error) {
	if strings.TrimSpace(settings.Address) == "" {
		return nil, errors.New("construct Petclinic server: address is empty")
	}
	if settings.ReadHeaderTimeout <= 0 ||
		settings.ReadTimeout <= 0 ||
		settings.WriteTimeout <= 0 ||
		settings.IdleTimeout <= 0 {
		return nil, errors.New(
			"construct Petclinic server: all HTTP timeouts must be positive",
		)
	}
	if handler == nil {
		return nil, errors.New("construct Petclinic server: handler is nil")
	}
	return &Server{
		httpServer: &http.Server{
			Addr:              settings.Address,
			Handler:           securityHeaders(handler),
			ReadHeaderTimeout: settings.ReadHeaderTimeout,
			ReadTimeout:       settings.ReadTimeout,
			WriteTimeout:      settings.WriteTimeout,
			IdleTimeout:       settings.IdleTimeout,
		},
		serveErrors: make(chan error, 1),
	}, nil
}

// Start binds the configured listener.
//
// @OnStart
func (server *Server) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("start Petclinic server: context is nil")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("start Petclinic server: %w", err)
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.started {
		return errors.New("start Petclinic server: server is already started")
	}
	if server.completed {
		return errors.New("start Petclinic server: stopped server cannot be restarted")
	}
	listener, err := (&net.ListenConfig{}).Listen(
		ctx,
		"tcp",
		server.httpServer.Addr,
	)
	if err != nil {
		return fmt.Errorf(
			"start Petclinic server on %s: %w",
			server.httpServer.Addr,
			err,
		)
	}
	server.listener = listener
	server.started = true
	server.stopDone = make(chan struct{})
	go func() {
		server.serveErrors <- server.httpServer.Serve(listener)
	}()
	return nil
}

// Stop gracefully drains the listener and observes its serve result.
//
// @OnStop
func (server *Server) Stop(ctx context.Context) error {
	if ctx == nil {
		return errors.New("stop Petclinic server: context is nil")
	}
	server.mu.Lock()
	switch {
	case server.completed:
		err := server.stopErr
		server.mu.Unlock()
		return err
	case !server.started:
		server.mu.Unlock()
		return nil
	case server.stopping:
		done := server.stopDone
		server.mu.Unlock()
		select {
		case <-done:
			server.mu.RLock()
			err := server.stopErr
			server.mu.RUnlock()
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	server.stopping = true
	done := server.stopDone
	server.mu.Unlock()

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
	result := errors.Join(shutdownErr, serveErr)
	server.mu.Lock()
	server.started = false
	server.stopping = false
	server.completed = true
	server.stopErr = result
	close(done)
	server.mu.Unlock()
	return result
}

// Address returns the bound address, or the configured address before start.
func (server *Server) Address() string {
	server.mu.RLock()
	defer server.mu.RUnlock()
	if server.listener != nil {
		return server.listener.Addr().String()
	}
	return server.httpServer.Addr
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		header := writer.Header()
		header.Set(
			"Content-Security-Policy",
			"default-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'",
		)
		header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		header.Set("Referrer-Policy", "same-origin")
		header.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(writer, request)
	})
}
