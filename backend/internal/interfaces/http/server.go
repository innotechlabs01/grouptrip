// Package http provides HTTP transport for the grouptrip API.
package http

import (
	"net/http"
	"os"
	"time"

	"github.com/frg/grouptrip/internal/application/authservice"
	"github.com/frg/grouptrip/internal/application/commands"
	"github.com/frg/grouptrip/internal/application/queries"
	"github.com/frg/grouptrip/internal/interfaces/http/middleware"
	"github.com/frg/grouptrip/internal/infrastructure/contribrepo"
	"github.com/frg/grouptrip/internal/infrastructure/fundrepo"
)

// Server is the HTTP server holding dependencies and routing.
type Server struct {
	repo          *fundrepo.SQLiteRepo
	contribs      *contribrepo.SQLiteContribRepo
	webhookSecret string
	contribute    *commands.ContributeCommand
	progress      *queries.GetFundProgress
	auth          *authservice.SessionService
	mux           *http.ServeMux
	rateLimiter   *middleware.RateLimiter
}

// NewServer creates a Server and registers all routes (without webhook deps).
func NewServer(repo *fundrepo.SQLiteRepo) *Server {
	return NewServerWithAuth(repo, nil, "", nil, nil, nil)
}

// NewServerWithWebhook creates a Server with the optional contribution repository and
// webhook secret wired, enabling the Polar webhook route. When contribs is nil the webhook
// route is not registered; when webhookSecret is empty the route fails closed (401/500).
func NewServerWithWebhook(repo *fundrepo.SQLiteRepo, contribs *contribrepo.SQLiteContribRepo, webhookSecret string) *Server {
	return NewServerWithAuth(repo, contribs, webhookSecret, nil, nil, nil)
}

// NewServerWithPayments creates a Server with all capabilities wired: fund CRUD,
// webhook, contribution charging, and fund progress queries. When cmd or progress
// is nil, their respective routes are not registered.
func NewServerWithPayments(
	repo *fundrepo.SQLiteRepo,
	contribs *contribrepo.SQLiteContribRepo,
	webhookSecret string,
	cmd *commands.ContributeCommand,
	progress *queries.GetFundProgress,
) *Server {
	return NewServerWithAuth(repo, contribs, webhookSecret, cmd, progress, nil)
}

// NewServerWithAuth wires every capability including auth. Passing nil for cmd,
// progress, or auth disables the corresponding routes.
func NewServerWithAuth(
	repo *fundrepo.SQLiteRepo,
	contribs *contribrepo.SQLiteContribRepo,
	webhookSecret string,
	cmd *commands.ContributeCommand,
	progress *queries.GetFundProgress,
	auth *authservice.SessionService,
) *Server {
	s := &Server{
		repo:          repo,
		contribs:      contribs,
		webhookSecret: webhookSecret,
		contribute:    cmd,
		progress:      progress,
		auth:          auth,
		mux:           http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// Trip routes
	s.mux.HandleFunc("GET /trips", s.listTrips)
	s.mux.HandleFunc("POST /trips", s.createTrip)
	s.mux.HandleFunc("POST /funds", s.createFund)
	// POST /funds/{id}/members — add a member to a fund
	s.mux.HandleFunc("POST /funds/{id}/members", s.addMember)
	// GET /funds/{id} — load and return a fund
	s.mux.HandleFunc("GET /funds/{id}", s.getFund)
	// POST /webhooks/polar — Polar payment webhook (only when wired)
	if s.contribs != nil {
		s.mux.HandleFunc("POST /webhooks/polar", s.webhookPolar)
	}
	// POST /funds/{id}/contributions — charge a contribution (only when wired)
	if s.contribute != nil {
		s.mux.HandleFunc("POST /funds/{id}/contributions", s.handleContribute)
	}
	// GET /funds/{id}/progress — fund progress query (only when wired)
	if s.progress != nil {
		s.mux.HandleFunc("GET /funds/{id}/progress", s.fundProgress)
	}
	// Auth routes
	if s.auth != nil {
		s.mux.HandleFunc("POST /auth/register", s.register)
		s.mux.HandleFunc("POST /auth/login", s.login)
		s.mux.HandleFunc("POST /auth/logout", s.logout)
		s.mux.HandleFunc("GET /auth/me", s.me)
		s.mux.HandleFunc("POST /auth/refresh", s.refresh)
	}
}

// ServeHTTP delegates to the underlying mux, optionally rate limited.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("RATE_LIMIT_ENABLED") == "true" {
		if s.rateLimiter == nil {
			s.rateLimiter = middleware.NewRateLimiter(100, time.Minute)
		}
		s.rateLimiter.Middleware(s.mux).ServeHTTP(w, r)
		return
	}
	s.mux.ServeHTTP(w, r)
}
