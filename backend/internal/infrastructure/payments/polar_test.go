package payments

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- helpers ---------------------------------------------------------------

// newTestServer creates a fake Polar API server and returns the client
// pointed at it. The handler is caller-controlled.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*PolarClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := NewPolarClient(srv.URL, "test-token", srv.Client())
	return c, srv
}

// --- Tests -----------------------------------------------------------------

func TestCreateCustomer_Success(t *testing.T) {
	const wantID = "cust_abc-123"
	const externalID = "user-42@example.com"

	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Assert path & method
		if r.URL.Path != "/v1/customers/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		// Assert auth header
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		// Assert Content-Type
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("unexpected Content-Type: %s", ct)
		}
		// Read and inspect body
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]interface{}
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("failed to parse request body: %v", err)
		}
		if parsed["external_id"] != externalID {
			t.Fatalf("expected external_id %q, got %v", externalID, parsed["external_id"])
		}
		if parsed["email"] != externalID {
			t.Fatalf("expected email %q, got %v", externalID, parsed["email"])
		}
		// Return fake customer
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    wantID,
			"email": externalID,
			"type":  "individual",
		})
	})
	defer srv.Close()

	id, err := c.CreateCustomer(context.Background(), externalID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != wantID {
		t.Fatalf("expected customer ID %q, got %q", wantID, id)
	}
}

func TestCreateDraftOrder_Success(t *testing.T) {
	const wantID = "order_xyz-789"

	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/orders/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]interface{}
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("failed to parse request body: %v", err)
		}
		if parsed["customer_id"] != "cust_abc" {
			t.Fatalf("expected customer_id 'cust_abc', got %v", parsed["customer_id"])
		}
		if parsed["product_id"] != "prod_123" {
			t.Fatalf("expected product_id 'prod_123', got %v", parsed["product_id"])
		}
		if parsed["amount"].(float64) != 2500 {
			t.Fatalf("expected amount 2500, got %v", parsed["amount"])
		}
		if parsed["currency"] != "usd" {
			t.Fatalf("expected currency 'usd', got %v", parsed["currency"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     wantID,
			"status": "draft",
		})
	})
	defer srv.Close()

	id, err := c.CreateDraftOrder(context.Background(), DraftOrderInput{
		CustomerID: "cust_abc",
		ProductID:  "prod_123",
		Amount:     2500,
		Currency:   "usd",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != wantID {
		t.Fatalf("expected order ID %q, got %q", wantID, id)
	}
}

func TestFinalizeDraftOrder_Success(t *testing.T) {
	const wantID = "order_xyz-789"

	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/orders/"+wantID+"/finalize" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		// Body should contain payment_method_id
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]interface{}
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("failed to parse request body: %v", err)
		}
		if parsed["payment_method_id"] != "pm_visa_4242" {
			t.Fatalf("expected payment_method_id 'pm_visa_4242', got %v", parsed["payment_method_id"])
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     wantID,
			"status": "paid",
		})
	})
	defer srv.Close()

	id, err := c.FinalizeDraftOrder(context.Background(), wantID, "pm_visa_4242")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != wantID {
		t.Fatalf("expected order ID %q, got %q", wantID, id)
	}
}

func TestFinalizeDraftOrder_EmptyPaymentMethod_OmitsField(t *testing.T) {
	const orderID = "order_no_pm"

	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]interface{}
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("failed to parse request body: %v", err)
		}
		if _, exists := parsed["payment_method_id"]; exists {
			t.Fatalf("payment_method_id should be omitted when empty, got %v", parsed["payment_method_id"])
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     orderID,
			"status": "paid",
		})
	})
	defer srv.Close()

	_, err := c.FinalizeDraftOrder(context.Background(), orderID, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFinalizeDraftOrder_402_CardDeclined(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "payment_failed",
			"detail": "card was declined",
		})
	})
	defer srv.Close()

	_, err := c.FinalizeDraftOrder(context.Background(), "order_1", "pm_bad")
	if err == nil {
		t.Fatal("expected error for 402 response")
	}
	if err != ErrCardDeclined {
		t.Fatalf("expected ErrCardDeclined, got: %v", err)
	}
}

func TestFinalizeDraftOrder_403_OffSessionNotEnabled(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "forbidden",
			"detail": "off-session charges not enabled",
		})
	})
	defer srv.Close()

	_, err := c.FinalizeDraftOrder(context.Background(), "order_2", "pm_visa")
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if err != ErrOffSessionNotEnabled {
		t.Fatalf("expected ErrOffSessionNotEnabled, got: %v", err)
	}
}

func TestFinalizeDraftOrder_412_OrderNotDraft(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPreconditionFailed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "precondition_failed",
			"detail": "order not in draft status",
		})
	})
	defer srv.Close()

	_, err := c.FinalizeDraftOrder(context.Background(), "order_3", "pm_visa")
	if err == nil {
		t.Fatal("expected error for 412 response")
	}
	if err != ErrOrderNotDraft {
		t.Fatalf("expected ErrOrderNotDraft, got: %v", err)
	}
}

func TestRefund_Success(t *testing.T) {
	const wantRefundID = "refund_abc-001"

	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/refunds/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]interface{}
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("failed to parse request body: %v", err)
		}
		if parsed["order_id"] != "order_to_refund" {
			t.Fatalf("expected order_id 'order_to_refund', got %v", parsed["order_id"])
		}
		if parsed["reason"] != "customer_request" {
			t.Fatalf("expected reason 'customer_request', got %v", parsed["reason"])
		}
		// Amount must be passed through from the caller (no guessing/hardcoding).
		if parsed["amount"].(float64) != 2500 {
			t.Fatalf("expected amount 2500 (from caller), got %v", parsed["amount"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     wantRefundID,
			"status": "pending",
		})
	})
	defer srv.Close()

	err := c.Refund(context.Background(), "order_to_refund", 2500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = wantRefundID // refund ID not surfaced by interface, just verify no error
}

func TestRefund_NonPositiveAmount(t *testing.T) {
	c := NewPolarClient("https://api.polar.sh", "test-token", &http.Client{})
	err := c.Refund(context.Background(), "order_1", 0)
	if err == nil {
		t.Fatal("expected error for zero refund amount")
	}
	if err := c.Refund(context.Background(), "order_1", -100); err == nil {
		t.Fatal("expected error for negative refund amount")
	}
}

func TestRefund_403_AlreadyRefunded(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "RefundedAlready",
			"detail": "Order is already fully refunded.",
		})
	})
	defer srv.Close()

	err := c.Refund(context.Background(), "order_already_refunded", 1000)
	if err == nil {
		t.Fatal("expected error for 403 refund response")
	}
	if err != ErrAlreadyRefunded {
		t.Fatalf("expected ErrAlreadyRefunded, got: %v", err)
	}
}

func TestGeneric500_ReturnsError(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "internal_server_error",
			"detail": "something broke",
		})
	})
	defer srv.Close()

	_, err := c.CreateDraftOrder(context.Background(), DraftOrderInput{
		CustomerID: "cust_x",
		ProductID:  "prod_x",
	})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestAuthHeader_PresentOnEveryRequest(t *testing.T) {
	var capturedAuth string
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "x"})
	})
	defer srv.Close()

	// CreateDraftOrder
	_, _ = c.CreateDraftOrder(context.Background(), DraftOrderInput{CustomerID: "a", ProductID: "b"})
	if capturedAuth != "Bearer test-token" {
		t.Fatalf("CreateDraftOrder: expected auth 'Bearer test-token', got %q", capturedAuth)
	}

	// FinalizeDraftOrder
	_, _ = c.FinalizeDraftOrder(context.Background(), "order_x", "pm_x")
	if capturedAuth != "Bearer test-token" {
		t.Fatalf("FinalizeDraftOrder: expected auth 'Bearer test-token', got %q", capturedAuth)
	}

	// Refund
	_ = c.Refund(context.Background(), "order_x", 500)
	if capturedAuth != "Bearer test-token" {
		t.Fatalf("Refund: expected auth 'Bearer test-token', got %q", capturedAuth)
	}

	// CreateCustomer
	_, _ = c.CreateCustomer(context.Background(), "ext_1")
	if capturedAuth != "Bearer test-token" {
		t.Fatalf("CreateCustomer: expected auth 'Bearer test-token', got %q", capturedAuth)
	}
}

func TestSavePaymentMethod_ValidatesInputs(t *testing.T) {
	c := NewPolarClient("http://unused", "tok", nil)

	// Empty customerID should error
	err := c.SavePaymentMethod(context.Background(), "", "pm_x")
	if err == nil {
		t.Fatal("expected error for empty customerID")
	}

	// Empty paymentMethodID should error
	err = c.SavePaymentMethod(context.Background(), "cust_x", "")
	if err == nil {
		t.Fatal("expected error for empty paymentMethodID")
	}

	// Both non-empty should succeed (no-op)
	err = c.SavePaymentMethod(context.Background(), "cust_x", "pm_x")
	if err != nil {
		t.Fatalf("expected nil error for valid inputs, got: %v", err)
	}
}
