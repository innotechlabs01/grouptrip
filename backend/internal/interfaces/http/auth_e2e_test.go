package http

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frg/grouptrip/internal/application/authservice"
	"github.com/frg/grouptrip/internal/infrastructure/authrepo"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

func TestE2EAuthFlow(t *testing.T) {
	// In-memory SQLite
	db, err := sql.Open("libsql", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ar := authrepo.NewSQLiteAuthRepo(db)
	if err := ar.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := authservice.NewSessionService(ar, []byte("test-secret-32-bytes-long-secret!"))
	srv := NewServerWithAuth(nil, nil, "", nil, nil, svc)

	// Start HTTP server on random port
	ts := httptest.NewUnstartedServer(srv)
	ts.StartTLS()
	defer ts.Close()

	// Register
	regPayload := map[string]string{"email": "alice@example.com", "password": "password123", "name": "Alice"}
	resp, err := ts.Client().Post(ts.URL+"/auth/register", "application/json", bytes.NewReader(mustJSON(t, regPayload)))
	if err != nil {
		t.Fatalf("register request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status: %d", resp.StatusCode)
	}
	var regResp map[string]string
	json.NewDecoder(resp.Body).Decode(&regResp)
	accessToken := regResp["access_token"]
	if accessToken == "" {
		t.Fatal("no access token")
	}
	// extract refresh cookie
	var refreshCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "refresh_token" {
			refreshCookie = c
			break
		}
	}
	if refreshCookie == nil {
		t.Fatal("no refresh cookie")
	}
	resp.Body.Close()

	// Me
	req, _ := http.NewRequest("GET", ts.URL+"/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp2, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("me request: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("me status: %d", resp2.StatusCode)
	}
	var meResp map[string]string
	json.NewDecoder(resp2.Body).Decode(&meResp)
	if meResp["email"] != "alice@example.com" {
		t.Fatalf("unexpected email: %s", meResp["email"])
	}
	resp2.Body.Close()

	// Refresh
	req3, _ := http.NewRequest("POST", ts.URL+"/auth/refresh", nil)
	req3.AddCookie(refreshCookie)
	resp3, err := ts.Client().Do(req3)
	if err != nil {
		t.Fatalf("refresh request: %v", err)
	}
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("refresh status: %d", resp3.StatusCode)
	}
	var refreshResp map[string]string
	json.NewDecoder(resp3.Body).Decode(&refreshResp)
	newAccess := refreshResp["access_token"]
	if newAccess == "" {
		t.Fatal("no new access token")
	}
	// get new refresh cookie
	var newRefreshCookie *http.Cookie
	for _, c := range resp3.Cookies() {
		if c.Name == "refresh_token" {
			newRefreshCookie = c
			break
		}
	}
	if newRefreshCookie == nil {
		t.Fatal("no new refresh cookie")
	}
	resp3.Body.Close()

	// Old refresh token should be rejected (reuse)
	reqReuse, _ := http.NewRequest("POST", ts.URL+"/auth/refresh", nil)
	reqReuse.AddCookie(refreshCookie)
	respReuse, err := ts.Client().Do(reqReuse)
	if err != nil {
		t.Fatalf("reuse request: %v", err)
	}
	if respReuse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected reuse unauthorized, got %d", respReuse.StatusCode)
	}
	respReuse.Body.Close()

	// Logout
	reqLogout, _ := http.NewRequest("POST", ts.URL+"/auth/logout", nil)
	reqLogout.Header.Set("Authorization", "Bearer "+newAccess)
	reqLogout.AddCookie(newRefreshCookie)
	respLogout, err := ts.Client().Do(reqLogout)
	if err != nil {
		t.Fatalf("logout request: %v", err)
	}
	if respLogout.StatusCode != http.StatusOK {
		t.Fatalf("logout status: %d", respLogout.StatusCode)
	}
	respLogout.Body.Close()
}

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}
