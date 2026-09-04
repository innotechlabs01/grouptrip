package http
import (
	"encoding/json"
	"net/http"
	"context"
	"github.com/google/uuid"
	"github.com/frg/grouptrip/internal/infrastructure/budget_repo"
)
func (s *Server) routesBudget(){
	s.mux.HandleFunc("GET /budget/categories", s.authMiddleware(s.handleListCategories))
	s.mux.HandleFunc("POST /budget/categories", s.authMiddleware(s.handleCreateCategory))
	s.mux.HandleFunc("GET /budget/items", s.authMiddleware(s.handleListItems))
	s.mux.HandleFunc("POST /budget/items", s.authMiddleware(s.handleCreateItem))
}
func (s *Server) handleListCategories(w http.ResponseWriter,r *http.Request){
	cats,err:=s.budgetRepo.ListCategories(r.Context(), r.URL.Query().Get("trip_id"))
	if err!=nil{ http.Error(w,"",500); return }
	json.NewEncoder(w).Encode(cats)
}
func (s *Server) handleCreateCategory(w http.ResponseWriter,r *http.Request){
	var in struct{Name string `json:"name"`; TripID string `json:"trip_id"`}
	json.NewDecoder(r.Body).Decode(&in)
	c:=budget_repo.Category{ID:uuid.NewString(), TripID:in.TripID, Name:in.Name}
	if err:=s.budgetRepo.CreateCategory(r.Context(),c); err!=nil{ http.Error(w,"",500); return }
	w.WriteHeader(http.StatusCreated); json.NewEncoder(w).Encode(c)
}
func (s *Server) handleListItems(w http.ResponseWriter,r *http.Request){ /* ... */ }
func (s *Server) handleCreateItem(w http.ResponseWriter,r *http.Request){ /* ... */ }
