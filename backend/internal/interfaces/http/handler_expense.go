package http
import "net/http"
func (s *Server) routesExpense() {
 s.router.HandleFunc("/expenses", s.authMiddleware(s.handleExpenses)).Methods("POST")
}
func (s *Server) handleExpenses(w http.ResponseWriter,r *http.Request){ w.WriteHeader(http.StatusCreated) }
