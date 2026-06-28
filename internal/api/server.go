// Package api serves the internal LAN HTTP API for managing Hover DNS records.
// Handlers are thin shims over the high-level hover ops; auth is a shared API
// key. It is intended to sit behind nginx (local CA TLS) on the loopback.
package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/bttnns/hover-dns/internal/hover"
)

type Server struct {
	c      *hover.Client
	apiKey string
	addr   string
}

func NewServer(c *hover.Client, apiKey, addr string) *Server {
	return &Server{c: c, apiKey: apiKey, addr: addr}
}

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", s.handleHealthz)

	r.Group(func(r chi.Router) {
		r.Use(s.requireAPIKey)
		r.Get("/v1/domains", s.handleListDomains)
		r.Get("/v1/domains/{domain}/records", s.handleListRecords)
		r.Post("/v1/domains/{domain}/records", s.handleAddRecord)
		r.Put("/v1/domains/{domain}/records/{id}", s.handleUpdateRecord)
		r.Delete("/v1/domains/{domain}/records/{id}", s.handleDeleteRecord)
	})

	return r
}

// Run serves until ctx is cancelled, then shuts down gracefully.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:              s.addr,
		Handler:           s.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       60 * time.Second,
		// WriteTimeout is generous: handlers proxy potentially slow Hover API
		// calls (each up to the client's 15s timeout, sometimes several in series).
		WriteTimeout: 90 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("api listening on %s", s.addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}
