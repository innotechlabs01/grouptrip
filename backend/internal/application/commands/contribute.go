package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/frg/grouptrip/internal/application/events"
	"github.com/frg/grouptrip/internal/domain/fund"
	"github.com/frg/grouptrip/internal/infrastructure/payments"
)

// ContributionRepository persists Contribution entities.
type ContributionRepository interface {
	Save(c *fund.Contribution) error
	FindByID(id string) (*fund.Contribution, error)
	FindByExternalRef(ref string) (*fund.Contribution, error)
	FindNonTerminalByFundCredential(fundID, planID, externalRef string) (*fund.Contribution, error)
}

// ContributeInput describes a charge request via Polar.
type ContributeInput struct {
	ContributionID string
	FundID         string
	PlanID         string // empty for one-off contributions
	ProductID      string // Polar one-time product (required)
	CustomerEmail  string // real email for Polar CreateCustomer
	CustomerID     string // optional pre-created Polar customer id
	Amount         int64
	Currency       string
	Description    string
}

// ContributeCommand charges a member via Polar idempotently.
type ContributeCommand struct {
	Funds    FundRepository
	Contrs   ContributionRepository
	Payments payments.PaymentProvider
	Events   events.EventSink
}

// Execute performs an idempotent contribution charge.
// If a contribution with the same ID already exists:
//   - Terminal → error (cannot re-charge)
//   - Non-terminal → nil (no-op: a webhook/retry will complete it)
//
// Otherwise a new charge flow is initiated and the contribution
// is persisted as PENDING before touching Polar (idempotency anchor).
func (h ContributeCommand) Execute(in ContributeInput) error {
	// --- Validate ---
	if in.ContributionID == "" {
		return errors.New("commands: contribution_id required")
	}
	if in.FundID == "" {
		return errors.New("commands: fund_id required")
	}
	if in.ProductID == "" {
		return errors.New("commands: product_id required")
	}
	if in.CustomerEmail == "" {
		return errors.New("commands: customer_email required")
	}
	if in.Amount <= 0 {
		return errors.New("commands: amount must be positive")
	}
	if in.Currency == "" {
		return errors.New("commands: currency required")
	}

	// --- Idempotency check ---
	existing, err := h.Contrs.FindByID(in.ContributionID)
	if err != nil {
		// Check if this is a "not found" error (use errors.Is on the sentinel).
		// For the in-memory fake and contribrepo: we treat any error as not-found
		// only if it's not an internal error. The real contribrepo returns a sentinel.
		// We check for the string pattern to avoid importing contribrepo here.
		if !isNotFoundErr(err) {
			return fmt.Errorf("commands: query existing contribution: %w", err)
		}
		// Not found → continue to create
	} else {
		// Contribution already exists
		if existing.IsTerminal() {
			return fmt.Errorf("commands: contribution %s already terminal (%s)", in.ContributionID, existing.Status)
		}
		// Non-terminal: idempotent no-op
		return nil
	}

	// --- Load fund (exists + active) ---
	if _, err := h.Funds.Load(in.FundID); err != nil {
		return fmt.Errorf("commands: load fund: %w", err)
	}

	// --- Create contribution in PENDING ---
	amount, err := fund.NewMoney(in.Amount, in.Currency)
	if err != nil {
		return fmt.Errorf("commands: invalid amount: %w", err)
	}
	cont, err := fund.NewContribution(in.ContributionID, in.PlanID, in.FundID, amount)
	if err != nil {
		return fmt.Errorf("commands: create contribution: %w", err)
	}

	// Persist as PENDING BEFORE touching Polar (idempotency anchor).
	if err := h.Contrs.Save(cont); err != nil {
		return fmt.Errorf("commands: save pending contribution: %w", err)
	}

	// --- Resolve Polar customer ---
	cid := in.CustomerID
	if cid == "" {
		var custErr error
		cid, custErr = h.Payments.CreateCustomer(context.Background(), in.CustomerEmail)
		if custErr != nil {
			_ = cont.Fail("customer_creation_failed")
			_ = h.Contrs.Save(cont)
			return fmt.Errorf("commands: create customer: %w", custErr)
		}
	}

	// --- Create draft order ---
	orderID, draftErr := h.Payments.CreateDraftOrder(context.Background(), payments.DraftOrderInput{
		CustomerID:  cid,
		ProductID:   in.ProductID,
		Amount:      in.Amount,
		Currency:    in.Currency,
		Description: in.Description,
	})
	if draftErr != nil {
		_ = cont.Fail("draft_failed")
		_ = h.Contrs.Save(cont)
		return fmt.Errorf("commands: create draft order: %w", draftErr)
	}

	// Store the draft order id (needed for idempotent recovery) and authorize.
	cont.ExternalRef = orderID
	if err := cont.Authorize(); err != nil {
		_ = cont.Fail("authorize_failed")
		_ = h.Contrs.Save(cont)
		return fmt.Errorf("commands: authorize contribution: %w", err)
	}
	if err := h.Contrs.Save(cont); err != nil {
		return fmt.Errorf("commands: save authorized contribution: %w", err)
	}

	// --- Finalize (charge) ---
	_, finalizeErr := h.Payments.FinalizeDraftOrder(context.Background(), orderID, "")
	if finalizeErr != nil {
		if errors.Is(finalizeErr, payments.ErrCardDeclined) {
			_ = cont.Fail("card_declined")
		} else {
			// Non-card-decline errors remain retryable (contribution stays AUTHORIZED/PROCESSING).
			// For simplicity in the initial implementation, we also fail on finalize errors.
			_ = cont.Fail("finalize_failed")
		}
		_ = h.Contrs.Save(cont)
		return fmt.Errorf("commands: finalize draft order: %w", finalizeErr)
	}

	// Charge accepted → mark PROCESSING. Success is confirmed by the webhook, not here.
	if err := cont.MarkProcessing(); err != nil {
		return fmt.Errorf("commands: mark processing: %w", err)
	}
	if err := h.Contrs.Save(cont); err != nil {
		return fmt.Errorf("commands: save processing contribution: %w", err)
	}

	// --- Events ---
	_ = h.Events.Publish(events.Event{
		Type:       "ContributionProcessing",
		OccurredAt: cont.CreatedAt,
		Payload: map[string]any{
			"fund_id":         in.FundID,
			"contribution_id": in.ContributionID,
			"amount":          in.Amount,
			"external_ref":    orderID,
		},
	})

	return nil
}

// isNotFoundErr checks whether an error represents "not found".
// This avoids importing the contribrepo package from the commands package.
func isNotFoundErr(err error) bool {
	return err != nil && (errors.Is(err, errNotFoundSentinel) || isGenericNotFoundError(err))
}

// errNotFoundSentinel is a local sentinel used to detect not-found errors
// returned by the ContributionRepository implementation.
// The commands package defines its own check; contribrepo.ErrNotFound
// is matched by string comparison if needed.
var errNotFoundSentinel = errors.New("contribrepo: not found")

// isNotFoundError checks if the error message contains the not-found pattern.
func isGenericNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "contribrepo: not found" ||
		err.Error() == "commands: query existing contribution: contribrepo: not found"
}
