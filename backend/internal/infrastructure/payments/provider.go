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
	FinalizeDraftOrder(ctx context.Context, orderID string) (string, error)

	// Refund reverses a SUCCEEDED order.
	Refund(ctx context.Context, orderID string) error
}

// DraftOrderInput describes a contribution charge (draft-order create).
type DraftOrderInput struct {
	CustomerID  string
	Amount      int64  // smallest currency unit
	Currency    string // ISO 4217 lowercase
	Description string
	MethodID    string // payment method to charge (optional)
}

// ErrNotImplemented signals a stub capability not yet wired to a real provider.
var ErrNotImplemented = errors.New("payments: not implemented")
