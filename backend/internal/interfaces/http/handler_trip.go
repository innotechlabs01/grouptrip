package http
import ("encoding/json"; "net/http"; "github.com/google/uuid")
func (s *Server) listTrips(w http.ResponseWriter,r *http.Request){ w.Header().Set("Content-Type","application/json"); json.NewEncoder(w).Encode([]interface{}{}) }
func (s *Server) createTrip(w http.ResponseWriter,r *http.Request){
 var body struct{ Name string `json:"name"`; OwnerID string `json:"owner_id"` }
 json.NewDecoder(r.Body).Decode(&body)
 w.WriteHeader(201)
 json.NewEncoder(w).Encode(map[string]string{"id":uuid.NewString(),"status":"created"})
}
