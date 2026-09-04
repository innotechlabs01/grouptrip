package http

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/frg/grouptrip/internal/domain/fund"
	"github.com/frg/grouptrip/internal/infrastructure/contribrepo"
	"github.com/frg/grouptrip/internal/infrastructure/fundrepo"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

// testWebhookSecret is shared across webhook tests; requests are signed with it.
const testWebhookSecret = "polar_whs_testsecret_123456"

// signedRequest builds a POST with the Standard Webhooks signature headers set.
func signedRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	id := "evt_test"
	ts := strconvFormat(time.Now().Unix())
	req := httptest.NewRequest(http.MethodPost, "/webhooks/polar", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(headerWebhookID, id)
	req.Header.Set(headerWebhookTimestamp, ts)
	req.Header.Set(headerWebhookSignature, "v1,"+computeSignature(testWebhookSecret, id, ts, []byte(body)))
	return req
}

// setupWebhookServer creates a Server with both repos wired and a webhook secret set.
func setupWebhookServer(t *testing.T) (*Server, *fundrepo.SQLiteRepo, *contribrepo.SQLiteContribRepo) {
	t.Helper()

	tmpFile := "file:" + t.TempDir() + "/webhook_test.db"
	db, err := sql.Open("libsql", tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	fundRepo := fundrepo.NewSQLiteRepo(db)
	if err := fundRepo.Migrate(); err != nil {
		t.Fatal(err)
	}
	contribRepo := contribrepo.NewSQLiteContribRepo(db)
	if err := contribRepo.Migrate(); err != nil {
		t.Fatal(err)
	}

	srv := NewServerWithWebhook(fundRepo, contribRepo, testWebhookSecret)
	return srv, fundRepo, contribRepo
}

// seedActiveFundAndProcessingContribution creates an ACTIVE fund and a PROCESSING
// contribution referencing order_abc on it.
func seedActiveFundAndProcessingContribution(t *testing.T, fundRepo *fundrepo.SQLiteRepo, contribRepo *contribrepo.SQLiteContribRepo) {
	t.Helper()

	goal, _ := fund.NewMoney(100000, "usd")
	f, err := fund.NewFund("wf1", "wt1", goal)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.AddMember("u1")
	if err := f.Activate(); err != nil {
		t.Fatal(err)
	}
	if err := fundRepo.Save(f); err != nil {
		t.Fatal(err)
	}

	amount, _ := fund.NewMoney(5000, "usd")
	c, err := fund.NewContribution("wc1", "", "wf1", amount)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Authorize(); err != nil {
		t.Fatal(err)
	}
	if err := c.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	c.ExternalRef = "order_abc"
	if err := contribRepo.Save(c); err != nil {
		t.Fatal(err)
	}
}

func TestWebhookPolarHappyPath(t *testing.T) {
	srv, fundRepo, contribRepo := setupWebhookServer(t)
	seedActiveFundAndProcessingContribution(t, fundRepo, contribRepo)

	body := `{"type":"order.paid","data":{"order":{"id":"order_abc","status":"paid"}}}`
	req := signedRequest(t, body)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	// Contribution is now SUCCEEDED
	cont, err := contribRepo.FindByID("wc1")
	if err != nil {
		t.Fatal(err)
	}
	if cont.Status != fund.ContrSucceeded {
		t.Fatalf("expected SUCCEEDED, got %s", cont.Status)
	}

	// Fund collected reflects the amount
	f, err := fundRepo.Load("wf1")
	if err != nil {
		t.Fatal(err)
	}
	if f.Collected().Amount() != 5000 {
		t.Fatalf("expected collected 5000, got %d", f.Collected().Amount())
	}
}

func TestWebhookPolarNonRelevantEvent(t *testing.T) {
	srv, fundRepo, contribRepo := setupWebhookServer(t)
	seedActiveFundAndProcessingContribution(t, fundRepo, contribRepo)

	// Non-order.paid event → 200, no changes
	body := `{"type":"order.created","data":{"order":{"id":"order_abc","status":"draft"}}}`
	req := signedRequest(t, body)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for non-relevant event, got %d", w.Code)
	}

	// Contribution unchanged (still PROCESSING)
	cont, err := contribRepo.FindByID("wc1")
	if err != nil {
		t.Fatal(err)
	}
	if cont.Status != fund.ContrProcessing {
		t.Fatalf("expected PROCESSING (unchanged), got %s", cont.Status)
	}
}

func TestWebhookPolarUnknownOrder(t *testing.T) {
	srv, fundRepo, contribRepo := setupWebhookServer(t)
	seedActiveFundAndProcessingContribution(t, fundRepo, contribRepo)

	body := `{"type":"order.paid","data":{"order":{"id":"order_zzz","status":"paid"}}}`
	req := signedRequest(t, body)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown order, got %d", w.Code)
	}
}

func TestWebhookPolarIdempotentRedelivery(t *testing.T) {
	srv, fundRepo, contribRepo := setupWebhookServer(t)
	seedActiveFundAndProcessingContribution(t, fundRepo, contribRepo)

	// First delivery
	body := `{"type":"order.paid","data":{"order":{"id":"order_abc","status":"paid"}}}`
	req := signedRequest(t, body)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("first delivery: expected 200, got %d", w.Code)
	}

	// Second (redelivered) call — fresh body reader, should be idempotent
	fBefore, _ := fundRepo.Load("wf1")
	collectedBefore := fBefore.Collected().Amount()

	req2 := signedRequest(t, body)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("redelivery: expected 200, got %d", w2.Code)
	}

	// Collected amount must NOT change (idempotent I-5)
	fAfter, _ := fundRepo.Load("wf1")
	if fAfter.Collected().Amount() != collectedBefore {
		t.Fatalf("collected changed on redelivery: %d -> %d", collectedBefore, fAfter.Collected().Amount())
	}
}

func TestWebhookPolarRejectsBadSignature(t *testing.T) {
	srv, fundRepo, contribRepo := setupWebhookServer(t)
	seedActiveFundAndProcessingContribution(t, fundRepo, contribRepo)

	// Build a request signed with the WRONG secret → must be rejected before processing.
	body := `{"type":"order.paid","data":{"order":{"id":"order_abc","status":"paid"}}}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/polar", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	id := "evt_bad"
	ts := strconvFormat(time.Now().Unix())
	req.Header.Set(headerWebhookID, id)
	req.Header.Set(headerWebhookTimestamp, ts)
	req.Header.Set(headerWebhookSignature, "v1,"+computeSignature("polar_whs_wrongsecret_999", id, ts, []byte(body)))
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad signature, got %d", w.Code)
	}

	// Contribution unchanged (still PROCESSING) — nothing was processed.
	cont, err := contribRepo.FindByID("wc1")
	if err != nil {
		t.Fatal(err)
	}
	if cont.Status != fund.ContrProcessing {
		t.Fatalf("expected PROCESSING (rejected), got %s", cont.Status)
	}
}

func TestWebhookPolarFailsClosedWithoutSecret(t *testing.T) {
	fundRepo, contribRepo := newReposNoWebhook(t)
	srv := NewServerWithWebhook(fundRepo, contribRepo, "") // empty secret → fail closed

	body := `{"type":"order.paid","data":{"order":{"id":"order_abc","status":"paid"}}}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/polar", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 fail-closed when secret unset, got %d", w.Code)
	}
}

// newReposNoWebhook builds repos without a webhook server layer.
func newReposNoWebhook(t *testing.T) (*fundrepo.SQLiteRepo, *contribrepo.SQLiteContribRepo) {
	t.Helper()
	tmpFile := "file:" + t.TempDir() + "/webhook_test.db"
	db, err := sql.Open("libsql", tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	fundRepo := fundrepo.NewSQLiteRepo(db)
	if err := fundRepo.Migrate(); err != nil {
		t.Fatal(err)
	}
	contribRepo := contribrepo.NewSQLiteContribRepo(db)
	if err := contribRepo.Migrate(); err != nil {
		t.Fatal(err)
	}
	return fundRepo, contribRepo
}
