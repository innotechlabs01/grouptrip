package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frg/grouptrip/internal/application/commands"
	"github.com/frg/grouptrip/internal/application/events"
	"github.com/frg/grouptrip/internal/application/queries"
	"github.com/frg/grouptrip/internal/domain/fund"
	"github.com/frg/grouptrip/internal/infrastructure/contribrepo"
	"github.com/frg/grouptrip/internal/infrastructure/fundrepo"
	"github.com/frg/grouptrip/internal/infrastructure/payments"
)

// --- fakePaymentProvider implements payments.PaymentProvider for tests ---

type fakePaymentProvider struct {
	customerID    string
	orderID       string
	finalizeCalls int
}

var _ payments.PaymentProvider = (*fakePaymentProvider)(nil)

func (f *fakePaymentProvider) CreateCustomer(_ context.Context, _ string) (string, error) {
	return f.customerID, nil
}

func (f *fakePaymentProvider) SavePaymentMethod(_ context.Context, _, _ string) error {
	return nil
}

func (f *fakePaymentProvider) CreateDraftOrder(_ context.Context, _ payments.DraftOrderInput) (string, error) {
	return f.orderID, nil
}

func (f *fakePaymentProvider) FinalizeDraftOrder(_ context.Context, orderID, _ string) (string, error) {
	f.finalizeCalls++
	return orderID, nil
}

func (f *fakePaymentProvider) Refund(_ context.Context, _ string, _ int64) error {
	return nil
}

// --- helpers ---

// setupPaymentServer creates a Server with payment command + progress wired.
func setupPaymentServer(t *testing.T) (
	*Server, *fundrepo.SQLiteRepo, *contribrepo.SQLiteContribRepo, *fakePaymentProvider,
) {
	t.Helper()

	fundRepo, contribRepo := newReposNoWebhook(t)

	fp := &fakePaymentProvider{customerID: "cust_fake_1", orderID: "order_fake_1"}
	cmd := &commands.ContributeCommand{
		Funds:    fundRepo,
		Contrs:   contribRepo,
		Payments: fp,
		Events:   events.NoopSink{},
	}
	progress := &queries.GetFundProgress{Funds: fundRepo}

	srv := NewServerWithPayments(fundRepo, contribRepo, testWebhookSecret, cmd, progress)
	return srv, fundRepo, contribRepo, fp
}

// seedActiveFund creates an ACTIVE fund with one member and persists it.
func seedActiveFund(t *testing.T, fundRepo *fundrepo.SQLiteRepo) {
	t.Helper()

	goal, _ := fund.NewMoney(100000, "usd")
	f, err := fund.NewFund("pf1", "pt1", goal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.AddMember("pu1"); err != nil {
		t.Fatal(err)
	}
	if err := f.Activate(); err != nil {
		t.Fatal(err)
	}
	if err := fundRepo.Save(f); err != nil {
		t.Fatal(err)
	}
}

// --- tests ---

func TestContributeHandler_Success(t *testing.T) {
	srv, fundRepo, contribRepo, fp := setupPaymentServer(t)
	seedActiveFund(t, fundRepo)

	body := `{
		"contribution_id": "pc1",
		"product_id": "prod_abc",
		"customer_email": "test@example.com",
		"amount": 5000,
		"currency": "usd",
		"description": "Test contribution"
	}`
	req := httptest.NewRequest(http.MethodPost, "/funds/pf1/contributions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["contribution_id"] != "pc1" {
		t.Fatalf("expected contribution_id pc1, got %v", resp["contribution_id"])
	}
	if resp["status"] != "processing" {
		t.Fatalf("expected status processing, got %v", resp["status"])
	}

	// Verify contribution exists in DB with PROCESSING status.
	cont, err := contribRepo.FindByID("pc1")
	if err != nil {
		t.Fatal(err)
	}
	if cont.Status != fund.ContrProcessing {
		t.Fatalf("expected PROCESSING, got %s", cont.Status)
	}

	// Verify finalize was called exactly once.
	if fp.finalizeCalls != 1 {
		t.Fatalf("expected 1 finalize call, got %d", fp.finalizeCalls)
	}
}

func TestContributeHandler_MissingAmount(t *testing.T) {
	srv, _, _, _ := setupPaymentServer(t)

	// Body with amount = 0 (omitted in JSON → zero value).
	body := `{
		"contribution_id": "pc2",
		"product_id": "prod_abc",
		"customer_email": "test@example.com",
		"currency": "usd"
	}`
	req := httptest.NewRequest(http.MethodPost, "/funds/pf1/contributions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	errMsg, _ := resp["error"].(string)
	if errMsg != "amount must be positive" {
		t.Fatalf("expected 'amount must be positive', got %q", errMsg)
	}
}

func TestContributeHandler_MissingFund(t *testing.T) {
	srv, _, _, _ := setupPaymentServer(t)

	body := `{
		"contribution_id": "pc3",
		"product_id": "prod_abc",
		"customer_email": "test@example.com",
		"amount": 5000,
		"currency": "usd"
	}`
	req := httptest.NewRequest(http.MethodPost, "/funds/nonexistent/contributions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "fund not found" {
		t.Fatalf("expected 'fund not found', got %v", resp["error"])
	}
}

func TestProgressHandler_Success(t *testing.T) {
	srv, fundRepo, _, _ := setupPaymentServer(t)
	seedActiveFund(t, fundRepo)

	// First make a contribution so there's data to report.
	contributeBody := `{
		"contribution_id": "pp1",
		"product_id": "prod_abc",
		"customer_email": "test@example.com",
		"amount": 5000,
		"currency": "usd"
	}`
	contribReq := httptest.NewRequest(http.MethodPost, "/funds/pf1/contributions", bytes.NewBufferString(contributeBody))
	contribReq.Header.Set("Content-Type", "application/json")
	contribW := httptest.NewRecorder()
	srv.ServeHTTP(contribW, contribReq)

	if contribW.Code != http.StatusAccepted {
		t.Fatalf("contribute: expected 202, got %d — body: %s", contribW.Code, contribW.Body.String())
	}

	// Now GET progress.
	req := httptest.NewRequest(http.MethodGet, "/funds/pf1/progress", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp["fund_id"] != "pf1" {
		t.Fatalf("expected fund_id pf1, got %v", resp["fund_id"])
	}

	// Collected is 0 because the contribution is PROCESSING, not yet SUCCEEDED.
	// The progress query reflects ledger state, not in-flight charges.
	collected, _ := resp["collected"].(float64)
	if collected != 0 {
		t.Fatalf("expected collected 0 (PROCESSING), got %v", collected)
	}

	// Percent should be 0 since nothing is collected yet.
	percent, _ := resp["percent"].(float64)
	if percent != 0 {
		t.Fatalf("expected percent 0, got %v", percent)
	}

	// Goal should be 100000 usd.
	goalAmount, _ := resp["goal_amount"].(float64)
	if goalAmount != 100000 {
		t.Fatalf("expected goal_amount 100000, got %v", goalAmount)
	}
	goalCurrency, _ := resp["goal_currency"].(string)
	if goalCurrency != "usd" {
		t.Fatalf("expected goal_currency usd, got %v", goalCurrency)
	}

	// Status should be ACTIVE.
	status, _ := resp["status"].(string)
	if status != "ACTIVE" {
		t.Fatalf("expected status ACTIVE, got %v", status)
	}
}

func TestProgressHandler_NotFound(t *testing.T) {
	srv, _, _, _ := setupPaymentServer(t)

	req := httptest.NewRequest(http.MethodGet, "/funds/nonexistent/progress", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "fund not found" {
		t.Fatalf("expected 'fund not found', got %v", resp["error"])
	}
}
