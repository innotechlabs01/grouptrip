package http
import (
	"encoding/json"
	"net/http"
	"github.com/google/uuid"
)
func (s *Server) routesDecision(){
	s.mux.HandleFunc("GET /decisions", s.authMiddleware(s.handleListDecisions))
	s.mux.HandleFunc("POST /decisions", s.authMiddleware(s.handleCreateDecision))
	s.mux.HandleFunc("POST /decisions/{id}/vote", s.authMiddleware(s.handleVoteDecision))
}
func (s *Server) handleListDecisions(w http.ResponseWriter,r *http.Request){
	decisionsMu.Lock(); defer decisionsMu.Unlock()
	votesMu.Lock(); defer votesMu.Unlock()
	w.Header().Set("Content-Type","application/json")
	json.NewEncoder(w).Encode(decisions)
}
func (s *Server) handleCreateDecision(w http.ResponseWriter,r *http.Request){
	var body struct{ Title string `json:"title"`; Options []string `json:"options"` }
	json.NewDecoder(r.Body).Decode(&body)
	id:=uuid.NewString()
	d:=map[string]interface{}{"id":id,"title":body.Title,"options":body.Options}
	decisionsMu.Lock(); decisions=append(decisions,d); decisionsMu.Unlock()
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(d)
}
func (s *Server) handleVoteDecision(w http.ResponseWriter,r *http.Request){
	var body struct{ Option string `json:"option"` }
	json.NewDecoder(r.Body).Decode(&body)
	v:=map[string]string{"id":uuid.NewString(),"option":body.Option}
	votesMu.Lock(); votes=append(votes,v); votesMu.Unlock()
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(v)
}
