package fund

import (
	"errors"
	"time"
)

// Status is the Fund aggregate state machine.
type Status string

const (
	StatusOpen   Status = "OPEN"
	StatusActive Status = "ACTIVE"
	StatusFunded Status = "FUNDED"
)

// LedgerType classifies a FundLedgerEntry.
type LedgerType string

const (
	LedgerExpected  LedgerType = "EXPECTED"
	LedgerCollected LedgerType = "COLLECTED"
	LedgerPending   LedgerType = "PENDING"
	LedgerFailed    LedgerType = "FAILED"
)

// ErrContributionAlreadyRecorded is returned by RecordCollected when the same
// contribution_id already has a ledger entry (idempotency, I-5).
var ErrContributionAlreadyRecorded = errors.New("fund: contribution already recorded (idempotency, I-5)")

// FundMember is a participant in a Fund with a derived per-person target.
type FundMember struct {
	UserID          string
	PerPersonTarget Money
}

// FundLedgerEntry is an append-only, immutable ledger record (invariant I-6).
type FundLedgerEntry struct {
	ID             string
	Type           LedgerType
	Delta          Money
	ContributionID string
	OccurredAt     time.Time
}

// Fund is the aggregate root (financial heart).
type Fund struct {
	ID              string
	TripID          string
	Goal            Money
	Status          Status
	Members         []FundMember
	ledger          []FundLedgerEntry
	GoalAdjustments int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewFund creates a Fund. Enforces invariant I-1 (goal > 0).
func NewFund(id, tripID string, goal Money) (*Fund, error) {
	if id == "" {
		return nil, errors.New("fund: id must not be empty")
	}
	if tripID == "" {
		return nil, errors.New("fund: trip id must not be empty")
	}
	// I-1
	if goal.amount <= 0 {
		return nil, errors.New("fund: goal must be positive")
	}
	now := time.Now()
	return &Fund{
		ID:        id,
		TripID:    tripID,
		Goal:      goal,
		Status:    StatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// AddMember adds a participant, recomputing the per-person target from the goal and
// current participant count (invariant I-2).
func (f *Fund) AddMember(userID string) (FundMember, error) {
	if userID == "" {
		return FundMember{}, errors.New("fund: user id must not be empty")
	}
	// No duplicate members.
	for _, m := range f.Members {
		if m.UserID == userID {
			return FundMember{}, errors.New("fund: member already exists")
		}
	}
	target, err := f.Goal.Div(len(f.Members) + 1)
	if err != nil {
		return FundMember{}, err
	}
	member := FundMember{UserID: userID, PerPersonTarget: target}
	f.Members = append(f.Members, member)
	f.UpdatedAt = time.Now()
	return member, nil
}

// Activate transitions OPEN -> ACTIVE. Requires at least one participant (I-2 guard).
func (f *Fund) Activate() error {
	if f.Status != StatusOpen {
		return errors.New("fund: only OPEN fund can activate")
	}
	if len(f.Members) == 0 {
		return errors.New("fund: cannot activate without participants (I-2)")
	}
	f.Status = StatusActive
	f.UpdatedAt = time.Now()
	return nil
}

// Ledger returns a copy of the append-only ledger entries.
func (f *Fund) Ledger() []FundLedgerEntry {
	out := make([]FundLedgerEntry, len(f.ledger))
	copy(out, f.ledger)
	return out
}

// appendLedger is the single write-path for ledger entries (invariant I-6: append-only).
func (f *Fund) appendLedger(entry FundLedgerEntry) {
	f.ledger = append(f.ledger, entry)
	f.UpdatedAt = time.Now()
}

// hasContribution reports whether the ledger already holds an entry for contributionID (I-5).
func (f *Fund) hasContribution(contributionID string) bool {
	for _, e := range f.ledger {
		if e.ContributionID == contributionID {
			return true
		}
	}
	return false
}

// Collected projects the COLLECTED balance from ledger deltas.
func (f *Fund) Collected() Money {
	zero := ZeroMoney(f.Goal.currency)
	for _, e := range f.ledger {
		if e.Type == LedgerCollected {
			zero, _ = zero.Add(e.Delta)
		}
	}
	return zero
}

// Pending projects the PENDING balance from ledger deltas.
func (f *Fund) Pending() Money {
	zero := ZeroMoney(f.Goal.currency)
	for _, e := range f.ledger {
		if e.Type == LedgerPending {
			zero, _ = zero.Add(e.Delta)
		}
	}
	return zero
}

// Failed projects the FAILED balance from ledger deltas.
func (f *Fund) Failed() Money {
	zero := ZeroMoney(f.Goal.currency)
	for _, e := range f.ledger {
		if e.Type == LedgerFailed {
			zero, _ = zero.Add(e.Delta)
		}
	}
	return zero
}

// RebuildFund reconstructs a Fund from persisted state for repository hydration only.
// It does NOT re-run write-path invariants — the data is assumed to have been validated
// when originally persisted. Validates only structural invariants (non-empty id, trip_id,
// and currency) to prevent obviously corrupt data from entering the domain layer.
func RebuildFund(
	id, tripID string,
	goal Money,
	status Status,
	members []FundMember,
	ledger []FundLedgerEntry,
	goalAdjustments int,
	createdAt, updatedAt time.Time,
) (*Fund, error) {
	if id == "" {
		return nil, errors.New("fund: id must not be empty")
	}
	if tripID == "" {
		return nil, errors.New("fund: trip id must not be empty")
	}
	if goal.currency == "" {
		return nil, errors.New("fund: currency must not be empty")
	}
	// Defensive copy: callers retain ownership of their slices.
	m := make([]FundMember, len(members))
	copy(m, members)
	l := make([]FundLedgerEntry, len(ledger))
	copy(l, ledger)
	return &Fund{
		ID:              id,
		TripID:          tripID,
		Goal:            goal,
		Status:          status,
		Members:         m,
		ledger:          l,
		GoalAdjustments: goalAdjustments,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}, nil
}

// RecordCollected appends a COLLECTED ledger entry on provider-confirmed success (I-3).
// It is the authority that turns an authorized contribution into real collected money.
// Idempotent by contributionID (I-5): a contribution that already has a ledger entry is
// rejected, never double-counted. The caller must also guard at persistence with a UNIQUE
// constraint on contribution_id so a concurrent processor cannot race past this check.
func (f *Fund) RecordCollected(amount Money, contributionID string) error {
	if contributionID == "" {
		return errors.New("fund: contribution id must not be empty")
	}
	if f.Status != StatusActive {
		return errors.New("fund: can only record collected money on an ACTIVE fund")
	}
	if f.hasContribution(contributionID) {
		return ErrContributionAlreadyRecorded
	}
	f.appendLedger(FundLedgerEntry{
		ID:             newID(),
		Type:           LedgerCollected,
		Delta:          amount,
		ContributionID: contributionID,
		OccurredAt:     time.Now(),
	})
	// Mark FUNDED when collected >= goal (I-2 completion guard).
	if ok, _ := f.Collected().Gte(f.Goal); ok {
		f.Status = StatusFunded
	}
	return nil
}
