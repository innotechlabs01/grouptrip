package http
import (
	"encoding/json"
	"net/http"
	"github.com/google/uuid"
)
func (s *Server) routesExpense(){
	s.mux.HandleFunc("GET /expenses", s.authMiddleware(s.handleListExpenses))
	s.mux.HandleFunc("POST /expenses", s.authMiddleware(s.handleCreateExpense))
}
func (s *Server) handleListExpenses(w http.ResponseWriter,r *http.Request){
	expensesMu.Lock(); defer expensesMu.Unlock()
	w.Header().Set("Content-Type","application/json")
	json.NewEncoder(w).Encode(expenses)
}
func (s *Server) handleCreateExpense(w http.ResponseWriter,r *http.Request){
	var body struct{ Title string `json:"title"`; Amount float64 `json:"amount"`; CategoryID string `json:"category_id"` }
	json.NewDecoder(r.Body).Decode(&body)
	id:=uuid.NewString()
	e:=map[string]interface{}{"id":id,"title":body.Title,"amount":body.Amount,"category_id":body.CategoryID}
	expensesMu.Lock(); expenses=append(expenses,e); expensesMu.Unlock()
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(e)
}
