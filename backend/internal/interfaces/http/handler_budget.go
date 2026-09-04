package http
import "net/http"
func (s *Server) routesBudget() {
 s.router.HandleFunc("/budget/categories", s.authMiddleware(s.handleCategories)).Methods("POST")
 s.router.HandleFunc("/budget/items", s.authMiddleware(s.handleItems)).Methods("POST")
}
func (s *Server) handleCategories(w http.ResponseWriter,r *http.Request){ w.WriteHeader(http.StatusCreated) }
func (s *Server) handleItems(w http.ResponseWriter,r *http.Request){ w.WriteHeader(http.StatusCreated) }
