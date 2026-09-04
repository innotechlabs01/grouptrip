package commands

import (
	"errors"
	"time"

	"github.com/frg/grouptrip/internal/application/events"
	"github.com/frg/grouptrip/internal/domain/fund"
)

// FundRepository persists a Fund aggregate.
type FundRepository interface {
	Save(f *fund.Fund) error
	Load(id string) (*fund.Fund, error)
}

// RecordContributionInput is the application command to record a provider-confirmed
// contribution success (I-3), projecting it into the ledger.
type RecordContributionInput struct {
	FundID         string
	ContributionID string
	Amount         int64
	Currency       string
	ExternalRef    string
	OccurredAt     time.Time
}

// RecordContributionSuccess handles the fund-update use case. It finds the Contribution,
// marks it SUCCEEDED, appends a COLLECTED ledger entry, and emits the funding success chain.
type RecordContributionSuccess struct {
	Funds  FundRepository
	Events events.EventSink
}

// Execute processes a provider-confirmed success.
func (h RecordContributionSuccess) Execute(in RecordContributionInput) error {
	if h.Funds == nil {
		return errors.New("commands: fund repository required")
	}
	f, err := h.Funds.Load(in.FundID)
	if err != nil {
		return err
	}
	amount, err := fund.NewMoney(in.Amount, in.Currency)
	if err != nil {
		return err
	}

	// Record the COLLECTED movement on the aggregate.
	if err := f.RecordCollected(amount, in.ContributionID); err != nil {
		return err
	}

	contribution, err := fund.NewContribution(in.ContributionID, "", in.FundID, amount)
	if err != nil {
		return err
	}
	if err := contribution.Authorize(); err != nil {
		return err
	}
	if err := contribution.MarkProcessing(); err != nil {
		return err
	}
	if err := contribution.Succeed(in.ExternalRef); err != nil {
		return err
	}

	if err := h.Funds.Save(f); err != nil {
		return err
	}

	// Funding success chain (domain event).
	_ = h.Events.Publish(events.Event{
		Type:       "ContributionSucceeded",
		OccurredAt: in.OccurredAt,
		Payload: map[string]any{
			"fund_id":         f.ID,
			"contribution_id": in.ContributionID,
			"amount":          in.Amount,
			"external_ref":    in.ExternalRef,
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
