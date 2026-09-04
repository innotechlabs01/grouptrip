package queries_test

import (
	"testing"

	"github.com/frg/grouptrip/internal/application/queries"
	"github.com/frg/grouptrip/internal/domain/fund"
	"github.com/frg/grouptrip/internal/infrastructure/fundrepo"
)

func TestGetFundProgress(t *testing.T) {
	repo := fundrepo.NewMemoryRepo()
	g, _ := fund.NewMoney(1000000, "cop")
	f, _ := fund.NewFund("f1", "t1", g)
	_, _ = f.AddMember("u1")
	_, _ = f.AddMember("u2")
	_ = f.Activate()
	_ = f.RecordCollected(mustMoney(t, 500000, "cop"), "c1")
	_ = repo.Save(f)

	q := queries.GetFundProgress{Funds: repo}
	p, err := q.Execute("f1")
	if err != nil {
		t.Fatal(err)
	}
	if p.Percent != 50 {
		t.Fatalf("expected 50 percent, got %d", p.Percent)
	}
	if p.Collected.Amount() != 500000 {
		t.Fatalf("expected collected 500000, got %d", p.Collected.Amount())
	}
	// per-person target with 2 members = 500000
	if p.PerPersonTarget.Amount() != 500000 {
		t.Fatalf("expected per-person 500000, got %d", p.PerPersonTarget.Amount())
	}
}

func TestGetFundProgressClampsPercent(t *testing.T) {
	repo := fundrepo.NewMemoryRepo()
	g, _ := fund.NewMoney(1000000, "cop")
	f, _ := fund.NewFund("f1", "t1", g)
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	_ = f.RecordCollected(mustMoney(t, 2000000, "cop"), "c1") // over goal
	_ = repo.Save(f)

	q := queries.GetFundProgress{Funds: repo}
	p, err := q.Execute("f1")
	if err != nil {
		t.Fatal(err)
	}
	if p.Percent != 100 {
		t.Fatalf("expected clamped 100, got %d", p.Percent)
	}
}

func mustMoney(t *testing.T, amt int64, cur string) fund.Money {
	t.Helper()
	m, err := fund.NewMoney(amt, cur)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
