package payments

import (
	"context"
	"errors"
)

// PaymentProvider is the provider-agnostic interface the Fund depends on.
// Polar.sh is the concrete implementation (see docs/platform/01-architecture/fund-spec.md §9).
// The platform never stores card data — only PaymentMethodID references (ADR-001).
type PaymentProvider interface {
	// CreateCustomer registers a customer and returns a Polar customer ID.
	CreateCustomer(ctx context.Context, externalID string) (string, error)

	// SavePaymentMethod registers/returns a tokenized payment method reference
	// for a customer (stored at the provider, never here).
	SavePaymentMethod(ctx context.Context, customerID, paymentMethodID string) error

	// CreateDraftOrder creates a Polar draft order (does NOT charge).
	CreateDraftOrder(ctx context.Context, in DraftOrderInput) (string, error)

	// FinalizeDraftOrder charges the customer's saved payment method (off-session).
	// paymentMethodID is optional — pass "" to use the customer's default method.
	FinalizeDraftOrder(ctx context.Context, orderID, paymentMethodID string) (string, error)

	// Refund reverses a SUCCEEDED order by the given amount (smallest currency unit).
	// The caller supplies the amount from the Contribution/ledger record — Polar requires
	// an explicit amount on a refund and the platform must not guess or hardcode it.
	Refund(ctx context.Context, orderID string, amount int64) error
}

// DraftOrderInput describes a contribution charge (draft-order create).
type DraftOrderInput struct {
	CustomerID  string
	ProductID   string // Polar one-time product to charge (req by POST /v1/orders/)
	Amount      int64  // smallest currency unit (override of product price)
	Currency    string // ISO 4217 lowercase
	Description string
	MethodID    string // payment method to charge (optional, used at finalize)
}

// ErrNotImplemented signals a stub capability not yet wired to a real provider.
var ErrNotImplemented = errors.New("payments: not implemented")
