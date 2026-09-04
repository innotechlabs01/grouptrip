package fund

import (
	"testing"
	"time"
)

func TestRebuildFundSuccess(t *testing.T) {
	goal := mustMoney(t, 10000, "usd")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	entry := FundLedgerEntry{
		ID:             "le1",
		Type:           LedgerCollected,
		Delta:          mustMoney(t, 3000, "usd"),
		ContributionID: "c1",
		OccurredAt:     now,
	}

	f, err := RebuildFund(
		"f1", "t1", goal, StatusActive,
		[]FundMember{
			{UserID: "u1", PerPersonTarget: mustMoney(t, 5000, "usd")},
			{UserID: "u2", PerPersonTarget: mustMoney(t, 5000, "usd")},
		},
		[]FundLedgerEntry{entry},
		0, now, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if f.ID != "f1" {
		t.Fatalf("expected id f1, got %s", f.ID)
	}
	if f.TripID != "t1" {
		t.Fatalf("expected trip_id t1, got %s", f.TripID)
	}
	if f.Goal.Amount() != 10000 {
		t.Fatalf("expected goal 10000, got %d", f.Goal.Amount())
	}
	if f.Status != StatusActive {
		t.Fatalf("expected ACTIVE, got %s", f.Status)
	}
	if len(f.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(f.Members))
	}
	if f.Members[0].UserID != "u1" {
		t.Fatalf("expected member u1, got %s", f.Members[0].UserID)
	}
	// Ledger was hydrated, not re-appended via domain write path
	if len(f.Ledger()) != 1 {
		t.Fatalf("expected 1 ledger entry, got %d", len(f.Ledger()))
	}
	if f.Collected().Amount() != 3000 {
		t.Fatalf("expected collected 3000, got %d", f.Collected().Amount())
	}
	if f.GoalAdjustments != 0 {
		t.Fatalf("expected goal adjustments 0, got %d", f.GoalAdjustments)
	}
	if !f.CreatedAt.Equal(now) {
		t.Fatalf("expected created_at %v, got %v", now, f.CreatedAt)
	}
	if !f.UpdatedAt.Equal(now) {
		t.Fatalf("expected updated_at %v, got %v", now, f.UpdatedAt)
	}
}

func TestRebuildFundRejectsEmptyID(t *testing.T) {
	goal := mustMoney(t, 1000, "cop")
	now := time.Now()
	_, err := RebuildFund("", "t1", goal, StatusOpen, nil, nil, 0, now, now)
	if err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestRebuildFundRejectsEmptyTripID(t *testing.T) {
	goal := mustMoney(t, 1000, "cop")
	now := time.Now()
	_, err := RebuildFund("f1", "", goal, StatusOpen, nil, nil, 0, now, now)
	if err == nil {
		t.Fatal("expected error for empty trip id")
	}
}

func TestRebuildFundRejectsEmptyCurrency(t *testing.T) {
	goal := Money{amount: 1000, currency: ""} // zero-value currency
	now := time.Now()
	_, err := RebuildFund("f1", "t1", goal, StatusOpen, nil, nil, 0, now, now)
	if err == nil {
		t.Fatal("expected error for empty currency")
	}
}

func TestRebuildFundAllowsNilMembersAndLedger(t *testing.T) {
	goal := mustMoney(t, 500, "eur")
	now := time.Now()
	f, err := RebuildFund("f2", "t2", goal, StatusOpen, nil, nil, 0, now, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Members) != 0 {
		t.Fatalf("expected 0 members, got %d", len(f.Members))
	}
	if len(f.Ledger()) != 0 {
		t.Fatalf("expected 0 ledger entries, got %d", len(f.Ledger()))
	}
}

func TestRebuildFundWithMultipleLedgerEntries(t *testing.T) {
	goal := mustMoney(t, 10000, "usd")
	now := time.Now()
	entries := []FundLedgerEntry{
		{ID: "le1", Type: LedgerCollected, Delta: mustMoney(t, 2000, "usd"), ContributionID: "c1", OccurredAt: now},
		{ID: "le2", Type: LedgerCollected, Delta: mustMoney(t, 1500, "usd"), ContributionID: "c2", OccurredAt: now},
		{ID: "le3", Type: LedgerPending, Delta: mustMoney(t, 500, "usd"), ContributionID: "c3", OccurredAt: now},
	}
	f, err := RebuildFund("f3", "t3", goal, StatusActive, nil, entries, 0, now, now)
	if err != nil {
		t.Fatal(err)
	}
	if f.Collected().Amount() != 3500 {
		t.Fatalf("expected collected 3500, got %d", f.Collected().Amount())
	}
	if f.Pending().Amount() != 500 {
		t.Fatalf("expected pending 500, got %d", f.Pending().Amount())
	}
	if f.Failed().Amount() != 0 {
		t.Fatalf("expected failed 0, got %d", f.Failed().Amount())
	}
}
