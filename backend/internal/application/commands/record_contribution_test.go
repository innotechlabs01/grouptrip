package commands_test

import (
	"testing"
	"time"

	"github.com/frg/grouptrip/internal/application/commands"
	"github.com/frg/grouptrip/internal/application/events"
	"github.com/frg/grouptrip/internal/domain/fund"
	"github.com/frg/grouptrip/internal/infrastructure/fundrepo"
)

type recordingSink struct {
	got []events.Event
}

func (r *recordingSink) Publish(e events.Event) error {
	r.got = append(r.got, e)
	return nil
}

func newActiveFund(t *testing.T, repo *fundrepo.MemoryRepo, id, trip string, goal int64) *fund.Fund {
	t.Helper()
	g, _ := fund.NewMoney(goal, "cop")
	f, err := fund.NewFund(id, trip, g)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	if err := repo.Save(f); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestRecordContributionSuccessProjectsLedgerAndEvents(t *testing.T) {
	repo := fundrepo.NewMemoryRepo()
	newActiveFund(t, repo, "f1", "t1", 1000000)

	sink := &recordingSink{}
	h := commands.RecordContributionSuccess{Funds: repo, Events: sink}

	err := h.Execute(commands.RecordContributionInput{
		FundID:         "f1",
		ContributionID: "c1",
		Amount:         250000,
		Currency:       "cop",
		ExternalRef:    "order_abc",
		OccurredAt:     time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// ledger reflects collected
	f, _ := repo.Load("f1")
	if f.Collected().Amount() != 250000 {
		t.Fatalf("expected collected 250000, got %d", f.Collected().Amount())
	}
	// contribution succeeded
	if len(sink.got) < 2 {
		t.Fatalf("expected >=2 events, got %d", len(sink.got))
	}
	if sink.got[0].Type != "ContributionSucceeded" {
		t.Fatalf("expected ContributionSucceeded first, got %s", sink.got[0].Type)
	}
	if sink.got[1].Type != "FundUpdated" {
		t.Fatalf("expected FundUpdated second, got %s", sink.got[1].Type)
	}
}

func TestRecordContributionRejectsMissingRepo(t *testing.T) {
	h := commands.RecordContributionSuccess{Events: events.NoopSink{}}
	err := h.Execute(commands.RecordContributionInput{FundID: "f1"})
	if err == nil {
		t.Fatal("expected error for missing repository")
	}
}
