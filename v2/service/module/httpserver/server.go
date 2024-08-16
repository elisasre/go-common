// Package httpserver provides http server as module.
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Opt func(s *Server) error

type Server struct {
	srv             *http.Server
	ln              net.Listener
	shutdownTimeout time.Duration
	opts            []Opt
}

// New creates Server module with given options.
func New(opts ...Opt) *Server {
	return &Server{
		opts: opts,
	}
}

// Init starts net.Listener after applying all options.
// Options are applied in same order as they were provided.
func (s *Server) Init() error {
	s.srv = &http.Server{ReadHeaderTimeout: time.Second * 10}
	s.shutdownTimeout = time.Minute
	for _, opt := range s.opts {
		if err := opt(s); err != nil {
			return fmt.Errorf("httpserver.Server Option error: %w", err)
		}
	}

	ln, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return fmt.Errorf("failed to init listener: %w", err)
	}

	s.ln = ln
	return nil
}

// URL returns server's URL and can be called after initialization.
func (s *Server) URL() string {
	if s.srv.TLSConfig == nil {
		return "http://" + s.ln.Addr().String()
	}
	return "https://" + s.ln.Addr().String()
}

// Run starts serving http request and can be called after initialization.
func (s *Server) Run() error {
	err := s.srv.Serve(s.ln)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Stop calls shutdown for server.
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	return s.srv.Shutdown(ctx)
}

func (s *Server) Name() string {
	return "httpserver.Server"
}

// WithServer sets http.Server for module.
func WithServer(srv *http.Server) Opt {
	return func(s *Server) error {
		s.srv = srv
		return nil
	}
}

// WithAddr sets http.Server.Addr.
func WithAddr(addr string) Opt {
	return func(s *Server) error {
		s.srv.Addr = addr
		return nil
	}
}

// WithHandler sets http.Server.Handler.
func WithHandler(h http.Handler) Opt {
	return func(s *Server) error {
		s.srv.Handler = h
		return nil
	}
}

// WithShutdownTimeout sets timeout for graceful shutdown.
func WithShutdownTimeout(d time.Duration) Opt {
	return func(s *Server) error {
		s.shutdownTimeout = d
		return nil
	}
}
