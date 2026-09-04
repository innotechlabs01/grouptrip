package fund

import (
	"errors"
	"time"
)

// ContributionStatus is the per-charge state machine.
type ContributionStatus string

const (
	ContrPending    ContributionStatus = "PENDING"
	ContrAuthorized ContributionStatus = "AUTHORIZED"
	ContrProcessing ContributionStatus = "PROCESSING"
	ContrSucceeded  ContributionStatus = "SUCCEEDED"
	ContrFailed     ContributionStatus = "FAILED"
	ContrRefunded   ContributionStatus = "REFUNDED"
)

// Contribution is a single attempted charge, one per (plan, cycle). Enforces idempotency (I-5).
type Contribution struct {
	ID          string
	PlanID      string // empty for a one-off contribution
	FundID      string
	Amount      Money
	Status      ContributionStatus
	ExternalRef string // provider order/payment reference
	CreatedAt   time.Time
}

// NewContribution creates a contribution in PENDING state.
func NewContribution(id, planID, fundID string, amount Money) (*Contribution, error) {
	if id == "" {
		return nil, errors.New("fund: contribution id must not be empty")
	}
	if fundID == "" {
		return nil, errors.New("fund: contribution fund id must not be empty")
	}
	if amount.amount <= 0 {
		return nil, errors.New("fund: contribution amount must be positive")
	}
	return &Contribution{
		ID:        id,
		PlanID:    planID,
		FundID:    fundID,
		Amount:    amount,
		Status:    ContrPending,
		CreatedAt: time.Now(),
	}, nil
}

// Authorize transitions PENDING -> AUTHORIZED (user authorized the charge).
func (c *Contribution) Authorize() error {
	if c.Status != ContrPending {
		return errors.New("fund: only PENDING contribution can be authorized")
	}
	c.Status = ContrAuthorized
	return nil
}

// MarkProcessing transitions to PROCESSING once dispatched to the provider.
func (c *Contribution) MarkProcessing() error {
	if c.Status != ContrAuthorized {
		return errors.New("fund: only AUTHORIZED contribution can start processing")
	}
	c.Status = ContrProcessing
	return nil
}

// Succeed marks a provider-confirmed success (I-3 signal).
func (c *Contribution) Succeed(externalRef string) error {
	if c.Status != ContrProcessing {
		return errors.New("fund: only PROCESSING contribution can succeed")
	}
	if externalRef == "" {
		return errors.New("fund: success requires provider external reference")
	}
	c.Status = ContrSucceeded
	c.ExternalRef = externalRef
	return nil
}

// Fail marks a terminal failure.
func (c *Contribution) Fail(reason string) error {
	if c.Status == ContrSucceeded || c.Status == ContrRefunded {
		return errors.New("fund: succeeded/refunded contribution cannot fail")
	}
	c.Status = ContrFailed
	return nil
}

// Refund transitions SUCCEEDED -> REFUNDED.
func (c *Contribution) Refund() error {
	if c.Status != ContrSucceeded {
		return errors.New("fund: only SUCCEEDED contribution can refund")
	}
	c.Status = ContrRefunded
	return nil
}

// IsTerminal reports whether the contribution reached a final state.
func (c *Contribution) IsTerminal() bool {
	switch c.Status {
	case ContrSucceeded, ContrFailed, ContrRefunded:
		return true
	default:
		return false
	}
}
