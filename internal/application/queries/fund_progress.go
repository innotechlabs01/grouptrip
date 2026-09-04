package queries

import (
	"errors"

	"github.com/frg/grouptrip/internal/domain/fund"
)

// FundRepository reads a Fund aggregate (read model source).
type FundRepository interface {
	Load(id string) (*fund.Fund, error)
}

// Progress projects the Fund progress for the dashboard.
type Progress struct {
	Goal            Money
	Collected       Money
	Pending         Money
	Failed          Money
	PerPersonTarget Money
	Percent         int
	Status          fund.Status
}

// Money is an alias so callers share one money type.
type Money = fund.Money

// GetFundProgress is a query: return the funding progress of a Fund.
type GetFundProgress struct {
	Funds FundRepository
}

// Execute returns the progress projection.
func (q GetFundProgress) Execute(fundID string) (*Progress, error) {
	if q.Funds == nil {
		return nil, errors.New("queries: fund repository required")
	}
	f, err := q.Funds.Load(fundID)
	if err != nil {
		return nil, err
	}
	collected := f.Collected()
	pct := 0
	if f.Goal.Amount() > 0 {
		pct = int(collected.Amount() * 100 / f.Goal.Amount())
		if pct > 100 {
			pct = 100
		}
	} else {
		pct = 0
	}
	// Per-person target is derived (I-2): goal / participant_count, computed at read time
	// (not from the per-member frozen field, which is historical).
	var target Money = fund.ZeroMoney(f.Goal.Currency())
	count := len(f.Members)
	if count < 1 {
		count = 1
	}
	if t, err := f.Goal.Div(count); err == nil {
		target = t
	}
	return &Progress{
		Goal:            f.Goal,
		Collected:       collected,
		Pending:         f.Pending(),
		Failed:          f.Failed(),
		PerPersonTarget: target,
		Percent:         pct,
		Status:          f.Status,
	}, nil
}
