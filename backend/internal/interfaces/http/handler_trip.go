package http
import ("encoding/json"; "net/http")
func (s *Server) handleCreateTrip(w http.ResponseWriter,r *http.Request){
 var body struct{ Name string `json:"name"`; OwnerID string `json:"owner_id"` }
 json.NewDecoder(r.Body).Decode(&body)
 w.WriteHeader(201)
 json.NewEncoder(w).Encode(map[string]string{"status":"created"})
}
