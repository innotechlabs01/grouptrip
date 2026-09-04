package fund

import (
	"errors"
	"time"
)

// PlanStatus is the ContributionPlan state machine.
type PlanStatus string

const (
	PlanActive    PlanStatus = "ACTIVE"
	PlanPaused    PlanStatus = "PAUSED"
	PlanCompleted PlanStatus = "COMPLETED"
	PlanFailed    PlanStatus = "FAILED"
)

// ContributionPlan is a participant's authorized, recurring contribution schedule.
type ContributionPlan struct {
	ID              string
	FundID          string
	UserID          string
	Frequency       Frequency
	Amount          Money
	TotalExpected   Money
	NextChargeAt    time.Time
	Status          PlanStatus
	PaymentMethodID string // tokenized reference; never card data (ADR-001)
	occurrences     int
	completedCycles int
}

// NewContributionPlan creates a plan. Requires an authorized payment method (invariant I-4).
func NewContributionPlan(id, fundID, userID string, freq Frequency, amount Money, paymentMethodID string, nextChargeAt time.Time) (*ContributionPlan, error) {
	if id == "" {
		return nil, errors.New("fund: plan id must not be empty")
	}
	if userID == "" {
		return nil, errors.New("fund: plan user id must not be empty")
	}
	if err := freq.Validate(); err != nil {
		return nil, err
	}
	if amount.amount <= 0 {
		return nil, errors.New("fund: plan amount must be positive")
	}
	// I-4 — a plan requires an authorized payment method reference.
	if paymentMethodID == "" {
		return nil, errors.New("fund: plan requires an authorized payment method (I-4)")
	}
	return &ContributionPlan{
		ID:              id,
		FundID:          fundID,
		UserID:          userID,
		Frequency:       freq,
		Amount:          amount,
		PaymentMethodID: paymentMethodID,
		NextChargeAt:    nextChargeAt,
		Status:          PlanActive,
	}, nil
}

// occurrencesFor computes the total occurrences for a frequency given a horizon in days.
func occurrencesFor(freq Frequency, horizonDays int) (int, error) {
	switch freq {
	case FrequencySingle:
		return 1, nil
	case FrequencyMonthly:
		if horizonDays <= 0 {
			return 0, errors.New("fund: monthly plan requires future horizon")
		}
		m := (horizonDays + 29) / 30
		if m < 1 {
			m = 1
		}
		return m, nil
	case FrequencyBiweekly:
		if horizonDays <= 0 {
			return 0, errors.New("fund: biweekly plan requires future horizon")
		}
		bw := (horizonDays + 13) / 14
		if bw < 1 {
			bw = 1
		}
		return bw, nil
	case FrequencyWeekly:
		if horizonDays <= 0 {
			return 0, errors.New("fund: weekly plan requires future horizon")
		}
		w := (horizonDays + 6) / 7
		if w < 1 {
			w = 1
		}
		return w, nil
	default:
		return 0, errors.New("fund: invalid frequency")
	}
}

// ComputeTotalExpected sets TotalExpected = Amount × occurrences over the horizon (I-7).
func (p *ContributionPlan) ComputeTotalExpected(horizonDays int) error {
	occ, err := occurrencesFor(p.Frequency, horizonDays)
	if err != nil {
		return err
	}
	p.occurrences = occ
	p.TotalExpected = Money{amount: p.Amount.amount * int64(occ), currency: p.Amount.currency}
	return nil
}

// Pause stops future charges; existing ones unaffected.
func (p *ContributionPlan) Pause() error {
	if p.Status != PlanActive {
		return errors.New("fund: only ACTIVE plan can pause")
	}
	p.Status = PlanPaused
	return nil
}

// ChargeAllowed reports whether a charge may be dispatched now (idempotency, I-5).
func (p *ContributionPlan) ChargeAllowed() bool {
	return p.Status == PlanActive
}
