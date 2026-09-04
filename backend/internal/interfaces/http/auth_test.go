package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/frg/grouptrip/internal/application/authservice"
	"github.com/frg/grouptrip/internal/infrastructure/authrepo"
)

func newAuthServer(t *testing.T) *Server {
	t.Helper()
	dir, _ := os.MkdirTemp("", "authsrv")
	t.Cleanup(func() { os.RemoveAll(dir) })
	db, _ := openServerTestDB(dir)
	ar := authrepo.NewSQLiteAuthRepo(db)
	if err := ar.Migrate(); err != nil {
		t.Fatal(err)
	}
	svc := authservice.NewSessionService(ar, []byte("test-secret-32-bytes-long-secret!"))
	return NewServerWithAuth(nil, nil, "", nil, nil, svc)
}

func doJSON(t *testing.T, srv *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	return w
}

func TestRegisterEndpoint(t *testing.T) {
	srv := newAuthServer(t)
	w := doJSON(t, srv, http.MethodPost, "/auth/register", `{"email":"a@b.com","password":"password123","name":"Ana"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !hasCookie(w, "refresh_token") {
		t.Fatal("expected refresh_token cookie")
	}
	var resp map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["access_token"] == "" || resp["access_token"] == nil {
		t.Fatal("expected access_token in body")
	}
}

func TestLoginFlow(t *testing.T) {
	srv := newAuthServer(t)
	doJSON(t, srv, http.MethodPost, "/auth/register", `{"email":"a@b.com","password":"password123","name":"Ana"}`)
	w := doJSON(t, srv, http.MethodPost, "/auth/login", `{"email":"a@b.com","password":"password123"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLoginWrongPassword(t *testing.T) {
	srv := newAuthServer(t)
	doJSON(t, srv, http.MethodPost, "/auth/register", `{"email":"a@b.com","password":"password123","name":"Ana"}`)
	w := doJSON(t, srv, http.MethodPost, "/auth/login", `{"email":"a@b.com","password":"nope"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMeRequiresToken(t *testing.T) {
	srv := newAuthServer(t)
	w := doJSON(t, srv, http.MethodGet, "/auth/me", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d: %s", w.Code, w.Body.String())
	}
}

func hasCookie(resp *httptest.ResponseRecorder, name string) bool {
	cookies := resp.Result().Cookies()
	for _, c := range cookies {
		if c.Name == name {
			return true
		}
	}
	return false
}
