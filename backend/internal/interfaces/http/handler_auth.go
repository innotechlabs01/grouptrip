package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/frg/grouptrip/internal/application/authservice"
)

const refreshCookieName = "refresh_token"

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	access, refresh, err := s.auth.Register(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		writeJSONError(w, statusForAuthErr(err), err.Error())
		return
	}
	setRefreshCookie(w, refresh)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"access_token": access})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	access, refresh, err := s.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeJSONError(w, statusForAuthErr(err), err.Error())
		return
	}
	setRefreshCookie(w, refresh)
	writeJSON(w, http.StatusOK, map[string]interface{}{"access_token": access})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.authFromReq(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := s.auth.Logout(r.Context(), sess.UserID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to logout")
		return
	}
	clearRefreshCookie(w)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.authFromReq(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	u, err := s.auth.Me(r.Context(), sess.UserID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"id": u.ID, "email": u.Email, "name": u.Name})
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(refreshCookieName)
	if err != nil || c.Value == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}
	access, newRefresh, err := s.auth.Refresh(r.Context(), c.Value)
	if err != nil {
		clearRefreshCookie(w)
		writeJSONError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	setRefreshCookie(w, newRefresh)
	writeJSON(w, http.StatusOK, map[string]interface{}{"access_token": access})
}

type sessionInfo struct{ UserID string }

func (s *Server) authFromReq(r *http.Request) (sessionInfo, bool) {
	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(authz, "Bearer ") {
		return sessionInfo{}, false
	}
	tok := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
	if tok == "" {
		return sessionInfo{}, false
	}
	uid, err := s.auth.ParseAccessToken(tok)
	if err != nil {
		return sessionInfo{}, false
	}
	return sessionInfo{UserID: uid}, true
}

func setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})
}

func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func statusForAuthErr(err error) int {
	switch err.Error() {
	case authservice.ErrEmailTaken.Error(),
		"auth: password required",
		"auth: password must be at least 8 characters",
		"auth: valid email required":
		return http.StatusBadRequest
	case authservice.ErrInvalidCredentials.Error(),
		authservice.ErrInvalidRefresh.Error(),
		authservice.ErrRefreshReuse.Error():
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
