// Package http provides HTTP transport for the grouptrip API.
package http

import (
	"net/http"

	"github.com/frg/grouptrip/internal/infrastructure/contribrepo"
	"github.com/frg/grouptrip/internal/infrastructure/fundrepo"
)

// Server is the HTTP server holding dependencies and routing.
type Server struct {
	repo          *fundrepo.SQLiteRepo
	contribs      *contribrepo.SQLiteContribRepo
	webhookSecret string
	mux           *http.ServeMux
}

// NewServer creates a Server and registers all routes (without webhook deps).
func NewServer(repo *fundrepo.SQLiteRepo) *Server {
	return NewServerWithWebhook(repo, nil, "")
}

// NewServerWithWebhook creates a Server with the optional contribution repository and
// webhook secret wired, enabling the Polar webhook route. When contribs is nil the webhook
// route is not registered; when webhookSecret is empty the route fails closed (401/500).
func NewServerWithWebhook(repo *fundrepo.SQLiteRepo, contribs *contribrepo.SQLiteContribRepo, webhookSecret string) *Server {
	s := &Server{repo: repo, contribs: contribs, webhookSecret: webhookSecret, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) routes() {
	// POST /funds — create a new fund
	s.mux.HandleFunc("POST /funds", s.createFund)
	// POST /funds/{id}/members — add a member to a fund
	s.mux.HandleFunc("POST /funds/{id}/members", s.addMember)
	// GET /funds/{id} — load and return a fund
	s.mux.HandleFunc("GET /funds/{id}", s.getFund)
	// POST /webhooks/polar — Polar payment webhook (only when wired)
	if s.contribs != nil {
		s.mux.HandleFunc("POST /webhooks/polar", s.webhookPolar)
	}
}

// ServeHTTP delegates to the underlying mux.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
