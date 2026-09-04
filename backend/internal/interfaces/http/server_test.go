package http

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frg/grouptrip/internal/infrastructure/fundrepo"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

func setupTestServer(t *testing.T) *Server {
	t.Helper()
	tmpFile := "file:" + t.TempDir() + "/http_test.db"
	db, err := sql.Open("libsql", tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	repo := fundrepo.NewSQLiteRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	return NewServer(repo)
}

func TestCreateFundSuccess(t *testing.T) {
	srv := setupTestServer(t)

	body := `{"trip_id":"t1","goal_amount":5000,"goal_currency":"usd"}`
	req := httptest.NewRequest(http.MethodPost, "/funds", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Fatal("expected non-empty id in response")
	}
	if resp["trip_id"] != "t1" {
		t.Fatalf("expected trip_id t1, got %v", resp["trip_id"])
	}
}

func TestCreateFundMissingFields(t *testing.T) {
	srv := setupTestServer(t)

	body := `{"trip_id":"t1"}`
	req := httptest.NewRequest(http.MethodPost, "/funds", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestAddMemberSuccess(t *testing.T) {
	srv := setupTestServer(t)

	// Create fund first
	body := `{"trip_id":"t1","goal_amount":10000,"goal_currency":"cop"}`
	req := httptest.NewRequest(http.MethodPost, "/funds", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	fundID := created["id"].(string)

	// Add member
	addBody := `{"user_id":"u1"}`
	req = httptest.NewRequest(http.MethodPost, "/funds/"+fundID+"/members", bytes.NewBufferString(addBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestGetFundReturnsCorrectData(t *testing.T) {
	srv := setupTestServer(t)

	// Create fund
	body := `{"trip_id":"t1","goal_amount":10000,"goal_currency":"usd"}`
	req := httptest.NewRequest(http.MethodPost, "/funds", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	fundID := created["id"].(string)

	// GET fund
	req = httptest.NewRequest(http.MethodGet, "/funds/"+fundID, nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["id"] != fundID {
		t.Fatalf("expected id %s, got %v", fundID, resp["id"])
	}
	if resp["status"] != "OPEN" {
		t.Fatalf("expected status OPEN, got %v", resp["status"])
	}
}

func TestGetFundNotFound(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/funds/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCreateAndGetFundIntegration(t *testing.T) {
	srv := setupTestServer(t)

	// Create fund
	body := `{"trip_id":"t-int","goal_amount":100000,"goal_currency":"cop"}`
	req := httptest.NewRequest(http.MethodPost, "/funds", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	fundID := created["id"].(string)

	// Add two members
	for _, uid := range []string{"u1", "u2"} {
		memberBody := `{"user_id":"` + uid + `"}`
		req = httptest.NewRequest(http.MethodPost, "/funds/"+fundID+"/members", bytes.NewBufferString(memberBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("add member %s: expected 201, got %d", uid, w.Code)
		}
	}

	// Get fund
	req = httptest.NewRequest(http.MethodGet, "/funds/"+fundID, nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	members, ok := resp["members"].([]interface{})
	if !ok {
		t.Fatalf("expected members array, got %T", resp["members"])
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
}

func TestCreateFundInvalidGoalCurrency(t *testing.T) {
	srv := setupTestServer(t)

	body := `{"trip_id":"t1","goal_amount":5000,"goal_currency":""}`
	req := httptest.NewRequest(http.MethodPost, "/funds", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty currency, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCreateFundZeroGoal(t *testing.T) {
	srv := setupTestServer(t)

	body := `{"trip_id":"t1","goal_amount":0,"goal_currency":"usd"}`
	req := httptest.NewRequest(http.MethodPost, "/funds", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for zero goal, got %d — body: %s", w.Code, w.Body.String())
	}
}
