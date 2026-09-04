package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Sentinel errors for Polar-specific failure modes.
var (
	ErrCardDeclined         = fmt.Errorf("payments: card declined or no payment method (402)")
	ErrOffSessionNotEnabled = fmt.Errorf("payments: off-session charges not enabled (403)")
	ErrOrderNotDraft        = fmt.Errorf("payments: order is not in draft status (412)")
	ErrAlreadyRefunded      = fmt.Errorf("payments: order is already fully refunded (403)")
)

// polarErrorResponse is the shape Polar returns on errors.
// Source: https://polar.sh/docs/api-reference/2026-04 (OpenAPI error schemas)
type polarErrorResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}

// PolarClient implements PaymentProvider against the Polar REST API.
// Source: https://polar.sh/docs/api-reference/2026-04
type PolarClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewPolarClient returns a ready-to-use PolarClient.
// If httpClient is nil, http.DefaultClient is used.
func NewPolarClient(baseURL, token string, httpClient *http.Client) *PolarClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &PolarClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		client:  httpClient,
	}
}

// NewPolarClientFromEnv constructs a PolarClient reading POLAR_BASE_URL and
// POLAR_TOKEN from the environment. Falls back to the production base URL if
// POLAR_BASE_URL is unset.
func NewPolarClientFromEnv(httpClient *http.Client) *PolarClient {
	baseURL := os.Getenv("POLAR_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.polar.sh"
	}
	return NewPolarClient(baseURL, os.Getenv("POLAR_TOKEN"), httpClient)
}

// ---------------------------------------------------------------------------
// PaymentProvider implementation
// ---------------------------------------------------------------------------

// CreateCustomer registers a customer with Polar and returns the Polar customer ID.
// Source: POST /v1/customers/ — https://polar.sh/docs/api-reference/2026-04/customers/create-customer
// Polar requires an `email`; the caller passes the customer's email as the externalID.
func (p *PolarClient) CreateCustomer(ctx context.Context, externalID string) (string, error) {
	body := map[string]interface{}{
		"external_id": externalID,
		"email":       externalID, // externalID doubles as the customer email
		"type":        "individual",
	}

	resp, err := p.do(ctx, http.MethodPost, "/v1/customers/", body)
	if err != nil {
		return "", fmt.Errorf("payments: create customer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("payments: create customer: %s", resp.Status)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("payments: decode customer response: %w", err)
	}
	return result.ID, nil
}

// SavePaymentMethod is a no-op that validates non-empty inputs.
// Polar stores payment methods via client-side checkout sessions; the server
// never touches raw card data. The payment_method_id is passed to
// FinalizeDraftOrder at charge time.
func (p *PolarClient) SavePaymentMethod(_ context.Context, customerID, paymentMethodID string) error {
	if customerID == "" {
		return fmt.Errorf("payments: customerID must not be empty")
	}
	if paymentMethodID == "" {
		return fmt.Errorf("payments: paymentMethodID must not be empty")
	}
	return nil
}

// CreateDraftOrder creates a draft order in Polar (no charge yet).
// Source: POST /v1/orders/ — https://polar.sh/docs/api-reference/2026-04/orders/create-draft-order
func (p *PolarClient) CreateDraftOrder(ctx context.Context, in DraftOrderInput) (string, error) {
	body := map[string]interface{}{
		"customer_id": in.CustomerID,
		"product_id":  in.ProductID,
	}
	if in.Amount > 0 {
		body["amount"] = in.Amount
	}
	if in.Currency != "" {
		body["currency"] = in.Currency
	}
	if in.Description != "" {
		body["description"] = in.Description
	}

	resp, err := p.do(ctx, http.MethodPost, "/v1/orders/", body)
	if err != nil {
		return "", fmt.Errorf("payments: create draft order: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("payments: create draft order: %s", resp.Status)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("payments: decode draft order response: %w", err)
	}
	return result.ID, nil
}

// FinalizeDraftOrder charges the customer's payment method off-session.
// Source: POST /v1/orders/{id}/finalize — https://polar.sh/docs/api-reference/2026-04/orders/finalize-order
func (p *PolarClient) FinalizeDraftOrder(ctx context.Context, orderID, paymentMethodID string) (string, error) {
	var body map[string]interface{}
	if paymentMethodID != "" {
		body = map[string]interface{}{
			"payment_method_id": paymentMethodID,
		}
	} else {
		body = map[string]interface{}{}
	}

	path := fmt.Sprintf("/v1/orders/%s/finalize", orderID)
	resp, err := p.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return "", fmt.Errorf("payments: finalize draft order: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		// success
	case http.StatusPaymentRequired: // 402
		return "", ErrCardDeclined
	case http.StatusForbidden: // 403
		return "", ErrOffSessionNotEnabled
	case http.StatusPreconditionFailed: // 412
		return "", ErrOrderNotDraft
	default:
		detail := readErrorDetail(resp)
		return "", fmt.Errorf("payments: finalize draft order: %s: %s", resp.Status, detail)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("payments: decode finalize response: %w", err)
	}
	return result.ID, nil
}

// Refund reverses a SUCCEEDED order by the given amount.
// Source: POST /v1/refunds/ — https://polar.sh/docs/api-reference/2026-04/refunds/create-refund
func (p *PolarClient) Refund(ctx context.Context, orderID string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("payments: refund amount must be positive (got %d)", amount)
	}
	body := map[string]interface{}{
		"order_id": orderID,
		"reason":   "customer_request",
		"amount":   amount,
	}

	resp, err := p.do(ctx, http.MethodPost, "/v1/refunds/", body)
	if err != nil {
		return fmt.Errorf("payments: refund: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		return nil
	case http.StatusForbidden: // 403 — RefundedAlready
		return ErrAlreadyRefunded
	default:
		detail := readErrorDetail(resp)
		return fmt.Errorf("payments: refund: %s: %s", resp.Status, detail)
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// do executes an HTTP request against the Polar API with auth headers.
func (p *PolarClient) do(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, &buf)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")

	return p.client.Do(req)
}

// readErrorDetail extracts the "detail" field from a Polar error response.
func readErrorDetail(resp *http.Response) string {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return ""
	}
	var errResp polarErrorResponse
	if json.Unmarshal(body, &errResp) != nil {
		return string(body)
	}
	if errResp.Detail != "" {
		return errResp.Detail
	}
	return errResp.Error
}
