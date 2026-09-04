package http
import "net/http"
func cors(next http.Handler) http.Handler{
	return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		w.Header().Set("Access-Control-Allow-Origin","*")
		w.Header().Set("Access-Control-Allow-Headers","Content-Type,Authorization")
		w.Header().Set("Access-Control-Allow-Methods","GET,POST,OPTIONS")
		if r.Method=="OPTIONS"{ w.WriteHeader(http.StatusNoContent); return }
		next.ServeHTTP(w,r)
	})
}
