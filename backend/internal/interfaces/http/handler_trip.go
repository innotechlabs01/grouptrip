package http
import (
	"encoding/json"
	"net/http"
	"github.com/google/uuid"
)
func (s *Server) listTrips(w http.ResponseWriter,r *http.Request){
	tripsMu.Lock(); defer tripsMu.Unlock()
	w.Header().Set("Content-Type","application/json")
	json.NewEncoder(w).Encode(trips)
}
func (s *Server) createTrip(w http.ResponseWriter,r *http.Request){
	var body struct{ Name string `json:"name"` }
	json.NewDecoder(r.Body).Decode(&body)
	id:=uuid.NewString()
	t:=map[string]string{"id":id,"name":body.Name}
	tripsMu.Lock(); trips=append(trips,t); tripsMu.Unlock()
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}
