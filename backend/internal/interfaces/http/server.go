package http
import (
	"net/http"
	"os"
	"time"
	"github.com/frg/grouptrip/internal/application/authservice"
	"github.com/frg/grouptrip/internal/interfaces/http/middleware"
	"github.com/frg/grouptrip/internal/infrastructure/fundrepo"
	"github.com/frg/grouptrip/internal/infrastructure/budget_repo"
	"github.com/frg/grouptrip/internal/infrastructure/decision_repo"
	"github.com/frg/grouptrip/internal/infrastructure/expense_repo"
	"github.com/frg/grouptrip/internal/infrastructure/trip_repo"
)
type Server struct {
	repo *fundrepo.SQLiteRepo
	auth *authservice.SessionService
	mux *http.ServeMux
	rateLimiter *middleware.RateLimiter
	budgetRepo *budget_repo.Repo
	decisionRepo *decision_repo.Repo
	expenseRepo *expense_repo.Repo
	tripRepo *trip_repo.Repo
}
func NewServerWithAuth(repo *fundrepo.SQLiteRepo, auth *authservice.SessionService, budgetRepo *budget_repo.Repo, decisionRepo *decision_repo.Repo, expenseRepo *expense_repo.Repo, tripRepo *trip_repo.Repo) *Server{
	s:=&Server{repo:repo,auth:auth,mux:http.NewServeMux(),budgetRepo:budgetRepo,decisionRepo:decisionRepo,expenseRepo:expenseRepo,tripRepo:tripRepo}
	s.routes()
	return s
}
func (s *Server) routes(){
	s.routesBudget()
	s.routesDecision()
	s.routesExpense()
	s.mux.HandleFunc("GET /trips", s.listTrips)
	s.mux.HandleFunc("POST /trips", s.createTrip)
	if s.auth!=nil{
		s.mux.HandleFunc("POST /auth/register", s.register)
		s.mux.HandleFunc("POST /auth/login", s.login)
		s.mux.HandleFunc("POST /auth/logout", s.logout)
		s.mux.HandleFunc("GET /auth/me", s.me)
		s.mux.HandleFunc("POST /auth/refresh", s.refresh)
	}
}
func (s *Server) ServeHTTP(w http.ResponseWriter,r *http.Request){
	if os.Getenv("RATE_LIMIT_ENABLED")=="true"{...}
	s.mux.ServeHTTP(w,r)
}
