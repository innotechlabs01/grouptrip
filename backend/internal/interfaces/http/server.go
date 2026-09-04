// Package http provides HTTP transport for the grouptrip API.
package http

import (
	"net/http"

	"github.com/frg/grouptrip/internal/infrastructure/fundrepo"
)

// Server is the HTTP server holding dependencies and routing.
type Server struct {
	repo *fundrepo.SQLiteRepo
	mux  *http.ServeMux
}

// NewServer creates a Server and registers all routes.
func NewServer(repo *fundrepo.SQLiteRepo) *Server {
	s := &Server{repo: repo, mux: http.NewServeMux()}
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
}

// ServeHTTP delegates to the underlying mux.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
