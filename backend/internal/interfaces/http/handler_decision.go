package http
import "net/http"
func (s *Server) routesDecision() {
 s.router.HandleFunc("/decisions", s.authMiddleware(s.handleDecisions)).Methods("POST")
 s.router.HandleFunc("/decisions/{id}/votes", s.authMiddleware(s.handleVotes)).Methods("POST")
}
func (s *Server) handleDecisions(w http.ResponseWriter,r *http.Request){ w.WriteHeader(http.StatusCreated) }
func (s *Server) handleVotes(w http.ResponseWriter,r *http.Request){ w.WriteHeader(http.StatusCreated) }
