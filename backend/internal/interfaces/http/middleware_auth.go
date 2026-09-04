package http
import (
	"context"
	"net/http"
	"strings"
)
type contextKey string
const userIDKey contextKey = "userID"
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc{
	return func(w http.ResponseWriter,r *http.Request){
		if s.auth==nil{ next(w,r); return }
		h:=r.Header.Get("Authorization")
		if !strings.HasPrefix(h,"Bearer "){ http.Error(w,"Unauthorized",http.StatusUnauthorized); return }
		token:=strings.TrimPrefix(h,"Bearer ")
		// Validate via auth service
		claims,err:=s.auth.ValidateToken(token)
		if err!=nil{ http.Error(w,"Unauthorized",http.StatusUnauthorized); return }
		ctx:=context.WithValue(r.Context(),userIDKey,claims.UserID)
		next(w,r.WithContext(ctx))
	}
}
