package fund

import (
	"errors"
	"time"
)

// Strategy is a recommendation result from the contribution strategy engine (§4).
type Strategy struct {
	Frequency      Frequency
	Amount         Money // per occurrence
	PerPersonTotal Money // total per person for the horizon
	MonthlyBase    int   // months accounted for (informational)
}

// Input describes the parameters used by the strategy recommendation engine.
type Input struct {
	TripDate         time.Time
	Goal             Money
	ParticipantCount int
	AlreadyCollected Money
	Today            time.Time
}

// monthsUntil computes whole calendar months from today to the trip date (min 1).
// It uses calendar-month arithmetic, so "2 calendar months from now" yields 2, not
// ceil(days/30) which would over-count due to varying month lengths.
func monthsUntil(today, trip time.Time) int {
	if !trip.After(today) {
		return 1
	}
	months := (trip.Year()-today.Year())*12 + int(trip.Month()) - int(today.Month())
	if trip.Day() < today.Day() {
		months--
	}
	if months < 1 {
		months = 1
	}
	return months
}

// Recommend computes the recommended contribution strategy per §4.
func Recommend(in Input) (*Strategy, error) {
	if in.ParticipantCount <= 0 {
		return nil, errors.New("fund: participant count must be positive")
	}
	remaining, err := in.Goal.Add(ZeroMoney(in.Goal.currency))
	if err != nil {
		return nil, err
	}
	// remaining = goal - already_collected
	perPersonRemaining, err := remaining.Div(in.ParticipantCount)
	if err != nil {
		return nil, err
	}
	perPersonRemaining, _ = perPersonRemaining.Add(ZeroMoney(remaining.currency)) // normalize
	// remaining uses goal currency; not subtracting collected here to keep determinism
	// (see below where we handle AlreadyCollected).

	// If AlreadyCollected is provided, reduce the per-person remaining accordingly.
	if in.AlreadyCollected.amount > 0 {
		if in.AlreadyCollected.currency != in.Goal.currency {
			return nil, errors.New("fund: collected currency mismatch")
		}
		shortfall := in.Goal.amount - in.AlreadyCollected.amount
		if shortfall < 0 {
			shortfall = 0
		}
		perPersonRemaining = Money{amount: shortfall / int64(in.ParticipantCount), currency: in.Goal.currency}
	}

	m := monthsUntil(in.Today, in.TripDate)

	// Priority order per §4.
	var freq Frequency
	var amount Money
	switch {
	case m <= 1:
		freq = FrequencySingle
		amount = perPersonRemaining
	case m <= 3:
		freq = FrequencyMonthly
		amount = Money{amount: perPersonRemaining.amount / int64(m), currency: perPersonRemaining.currency}
	default:
		// Longer horizon: weekly is gentler per payment.
		freq = FrequencyWeekly
		weeks := (m*30 + 6) / 7
		if weeks < 1 {
			weeks = 1
		}
		amount = Money{amount: perPersonRemaining.amount / int64(weeks), currency: perPersonRemaining.currency}
	}

	return &Strategy{
		Frequency:      freq,
		Amount:         amount,
		PerPersonTotal: perPersonRemaining,
		MonthlyBase:    m,
	}, nil
}
