package http
import (
	"encoding/json"
	"net/http"
	"github.com/google/uuid"
)
func (s *Server) routesPolar(){
	s.mux.HandleFunc("POST /trips/{id}/contribute", s.authMiddleware(s.handleContributeTrip))
}
func (s *Server) handleContributeTrip(w http.ResponseWriter,r *http.Request){
	// stub: simulate Polar contribution
	resp:=map[string]string{"id":uuid.NewString(),"status":"pending"}
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
