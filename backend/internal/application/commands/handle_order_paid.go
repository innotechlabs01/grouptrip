package commands

import (
	"errors"
	"fmt"
	"time"

	"github.com/frg/grouptrip/internal/application/events"
	"github.com/frg/grouptrip/internal/domain/fund"
)

// OrderPaidInput describes a Polar webhook order.paid event.
type OrderPaidInput struct {
	OrderID        string
	ContributionID string
	OccurredAt     time.Time
}

// HandleOrderPaid processes a Polar order.paid webhook by advancing the
// contribution to SUCCEEDED and recording the collected amount on the fund.
// It is idempotent: if the contribution is already SUCCEEDED the handler
// returns nil without double-recording (I-5).
type HandleOrderPaid struct {
	Funds  FundRepository
	Contrs ContributionRepository
	Events events.EventSink
}

// Execute processes a single order.paid event.
func (h HandleOrderPaid) Execute(in OrderPaidInput) error {
	if h.Funds == nil {
		return errors.New("commands: fund repository required")
	}

	// 1. Load contribution by external reference.
	cont, err := h.Contrs.FindByExternalRef(in.OrderID)
	if err != nil {
		return fmt.Errorf("commands: find contribution by order %s: %w", in.OrderID, err)
	}

	// 2. Idempotent: already terminal → no-op.
	if cont.IsTerminal() {
		return nil
	}

	// 3. Load fund.
	f, err := h.Funds.Load(cont.FundID)
	if err != nil {
		return fmt.Errorf("commands: load fund %s: %w", cont.FundID, err)
	}

	// 4. Advance contribution through PROCESSING → SUCCEEDED.
	//    If already PROCESSING (re-delivery), MarkProcessing returns an error
	//    since only AUTHORIZED can transition to PROCESSING — so we handle
	//    both cases: AUTHORIZED → PROCESSING → SUCCEEDED, or directly succeed
	//    if already in PROCESSING.
	if cont.Status == fund.ContrAuthorized {
		if err := cont.MarkProcessing(); err != nil {
			return fmt.Errorf("commands: mark processing: %w", err)
		}
	}
	// cont.Status is now PROCESSING (or was already).
	if err := cont.Succeed(in.OrderID); err != nil {
		// If already SUCCEEDED (concurrent delivery race), treat as idempotent.
		if cont.Status == fund.ContrSucceeded {
			return nil
		}
		return fmt.Errorf("commands: succeed contribution: %w", err)
	}

	// 5. Record collected on the fund aggregate (I-5: rejects duplicate contribution_id).
	if err := f.RecordCollected(cont.Amount, cont.ID); err != nil {
		// I-5 duplicate from the domain: treat as idempotent success (money already counted).
		if errors.Is(err, fund.ErrContributionAlreadyRecorded) {
			// Fund ledger already has this contribution — save contribution status and return.
			if saveErr := h.Contrs.Save(cont); saveErr != nil {
				return fmt.Errorf("commands: save contribution after duplicate: %w", saveErr)
			}
			return nil
		}
		return fmt.Errorf("commands: record collected: %w", err)
	}

	// 6. Persist both fund and contribution.
	if err := h.Funds.Save(f); err != nil {
		return fmt.Errorf("commands: save fund: %w", err)
	}
	if err := h.Contrs.Save(cont); err != nil {
		return fmt.Errorf("commands: save contribution: %w", err)
	}

	// 7. Publish events.
	_ = h.Events.Publish(events.Event{
		Type:       "ContributionSucceeded",
		OccurredAt: in.OccurredAt,
		Payload: map[string]any{
			"fund_id":         f.ID,
			"contribution_id": cont.ID,
			"amount":          cont.Amount.Amount(),
			"external_ref":    in.OrderID,
		},
	})
	_ = h.Events.Publish(events.Event{
		Type:       "FundUpdated",
		OccurredAt: in.OccurredAt,
		Payload: map[string]any{
			"fund_id":   f.ID,
			"collected": f.Collected().Amount(),
		},
	})

	return nil
}
