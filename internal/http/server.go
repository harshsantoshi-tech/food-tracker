// Package http contains the HTTP server, routing, and handlers.
package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Pinger is implemented by any dependency whose connectivity we want to
// surface on the health endpoint
type Pinger interface {
	HealthCheck(ctx context.Context) error
}

// WhatsAppHandler is implemented by whatsapp.Handler. Declaring it here
// (rather than importing the whatsapp package's concrete type) keeps
// this package decoupled from WhatsApp-specific details.
type WhatsAppHandler interface {
	HandleVerify(w http.ResponseWriter, r *http.Request)
	HandleIncoming(w http.ResponseWriter, r *http.Request)
}


// Server holds dependencies needed to build the HTTP router.
type Server struct {
	logger   *slog.Logger
	postgres Pinger
	redis    Pinger
	whatsapp WhatsAppHandler
	timeout  time.Duration
}

func NewServer(logger *slog.Logger, postgres, redis Pinger, whatsapp WhatsAppHandler, timeout time.Duration) *Server {
	return &Server{logger: logger, postgres: postgres, redis: redis, whatsapp: whatsapp, timeout: timeout}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /webhook/whatsapp", s.whatsapp.HandleVerify)
	mux.HandleFunc("POST /webhook/whatsapp", s.whatsapp.HandleIncoming)
	return s.withMiddleware(mux)
}

type healthComponent struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type healthResponse struct {
	Status     string                     `json:"status"`
	Components map[string]healthComponent `json:"components"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.timeout)
	defer cancel()

	resp := healthResponse{Status: "ok", Components: map[string]healthComponent{}}

	if err := s.postgres.HealthCheck(ctx); err != nil {
		resp.Status = "degraded"
		resp.Components["postgres"] = healthComponent{Status: "down", Error: err.Error()}
	} else {
		resp.Components["postgres"] = healthComponent{Status: "ok"}
	}

	if err := s.redis.HealthCheck(ctx); err != nil {
		resp.Status = "degraded"
		resp.Components["redis"] = healthComponent{Status: "down", Error: err.Error()}
	} else {
		resp.Components["redis"] = healthComponent{Status: "ok"}
	}

	statusCode := http.StatusOK
	if resp.Status != "ok" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		s.logger.Error("failed to encode health response", "error", err)
	}
}
