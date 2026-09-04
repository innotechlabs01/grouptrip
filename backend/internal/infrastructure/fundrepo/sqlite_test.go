package fundrepo

import (
	"database/sql"
	"testing"
	"time"

	"github.com/frg/grouptrip/internal/domain/fund"
)

func TestSQLiteSaveAndLoadRoundTrip(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	repo := NewSQLiteRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	goal := mustMoney(t, 10000, "usd")
	f, err := fund.NewFund("fund-1", "trip-1", goal)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.AddMember("user-1")
	_, _ = f.AddMember("user-2")

	if err := repo.Save(f); err != nil {
		t.Fatal(err)
	}

	loaded, err := repo.Load("fund-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != "fund-1" {
		t.Fatalf("expected id fund-1, got %s", loaded.ID)
	}
	if loaded.TripID != "trip-1" {
		t.Fatalf("expected trip_id trip-1, got %s", loaded.TripID)
	}
	if loaded.Goal.Amount() != 10000 {
		t.Fatalf("expected goal 10000, got %d", loaded.Goal.Amount())
	}
	if loaded.Goal.Currency() != "usd" {
		t.Fatalf("expected goal currency usd, got %s", loaded.Goal.Currency())
	}
	if loaded.Status != fund.StatusOpen {
		t.Fatalf("expected status OPEN, got %s", loaded.Status)
	}
	if len(loaded.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(loaded.Members))
	}
	memberIDs := map[string]bool{}
	for _, m := range loaded.Members {
		memberIDs[m.UserID] = true
		if m.PerPersonTarget.Currency() != "usd" {
			t.Fatalf("expected member per-person currency usd, got %s", m.PerPersonTarget.Currency())
		}
	}
	if !memberIDs["user-1"] || !memberIDs["user-2"] {
		t.Fatalf("expected members user-1 and user-2, got %v", memberIDs)
	}
}

func TestSQLiteSaveAndLoadWithLedgerEntries(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	repo := NewSQLiteRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	goal := mustMoney(t, 10000, "usd")
	f, err := fund.NewFund("fund-2", "trip-2", goal)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.AddMember("user-1")
	if err := f.Activate(); err != nil {
		t.Fatal(err)
	}

	// RecordCollected appends two ledger entries
	if err := f.RecordCollected(mustMoney(t, 4000, "usd"), "contrib-1"); err != nil {
		t.Fatal(err)
	}
	if err := f.RecordCollected(mustMoney(t, 2500, "usd"), "contrib-2"); err != nil {
		t.Fatal(err)
	}

	originalCollected := f.Collected().Amount()
	originalPending := f.Pending().Amount()

	if err := repo.Save(f); err != nil {
		t.Fatal(err)
	}

	loaded, err := repo.Load("fund-2")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != fund.StatusActive {
		t.Fatalf("expected ACTIVE, got %s", loaded.Status)
	}
	if loaded.Collected().Amount() != originalCollected {
		t.Fatalf("expected collected %d, got %d", originalCollected, loaded.Collected().Amount())
	}
	if loaded.Pending().Amount() != originalPending {
		t.Fatalf("expected pending %d, got %d", originalPending, loaded.Pending().Amount())
	}

	ledger := loaded.Ledger()
	if len(ledger) != 2 {
		t.Fatalf("expected 2 ledger entries, got %d", len(ledger))
	}
	// Both entries are COLLECTED; check deltas exist regardless of order
	// (same occurred_at means ordering falls back to id ASC, which may differ from insertion order).
	deltaAmounts := map[int64]bool{}
	for _, e := range ledger {
		if e.Type != fund.LedgerCollected {
			t.Fatalf("expected COLLECTED, got %s", e.Type)
		}
		deltaAmounts[e.Delta.Amount()] = true
	}
	if !deltaAmounts[4000] || !deltaAmounts[2500] {
		t.Fatalf("expected deltas 4000 and 2500, got %v", deltaAmounts)
	}
}

func TestSQLiteLedgerAppendOnly(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	repo := NewSQLiteRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	goal := mustMoney(t, 5000, "cop")
	f, err := fund.NewFund("fund-3", "trip-3", goal)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.AddMember("user-1")
	if err := f.Activate(); err != nil {
		t.Fatal(err)
	}
	if err := f.RecordCollected(mustMoney(t, 1000, "cop"), "c1"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(f); err != nil {
		t.Fatal(err)
	}

	// Record another entry and save again — ledger should append, not replace
	if err := f.RecordCollected(mustMoney(t, 500, "cop"), "c2"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(f); err != nil {
		t.Fatal(err)
	}

	loaded, err := repo.Load("fund-3")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Ledger()) != 2 {
		t.Fatalf("expected 2 ledger entries after second save, got %d", len(loaded.Ledger()))
	}
	if loaded.Collected().Amount() != 1500 {
		t.Fatalf("expected collected 1500 after append, got %d", loaded.Collected().Amount())
	}
}

func TestSQLiteLoadReturnsErrorForMissingFund(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	repo := NewSQLiteRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	_, err := repo.Load("nonexistent")
	if err == nil {
		t.Fatal("expected error loading nonexistent fund")
	}
}

func TestSQLiteMemberPerPersonTargetRoundTrip(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	repo := NewSQLiteRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	goal := mustMoney(t, 10000, "usd")
	f, err := fund.NewFund("fund-4", "trip-4", goal)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.AddMember("user-1")
	_, _ = f.AddMember("user-2")
	_, _ = f.AddMember("user-3")

	// Store original per-person targets (int64 division truncates: 10000/3 = 3333)
	origTargets := make(map[string]int64)
	for _, m := range f.Members {
		origTargets[m.UserID] = m.PerPersonTarget.Amount()
	}

	if err := repo.Save(f); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.Load("fund-4")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range loaded.Members {
		expected, ok := origTargets[m.UserID]
		if !ok {
			t.Fatalf("unexpected member %s", m.UserID)
		}
		if m.PerPersonTarget.Amount() != expected {
			t.Fatalf("member %s: expected per-person %d, got %d", m.UserID, expected, m.PerPersonTarget.Amount())
		}
		if m.PerPersonTarget.Currency() != "usd" {
			t.Fatalf("member %s: expected currency usd, got %s", m.UserID, m.PerPersonTarget.Currency())
		}
	}
}

func TestSQLiteTimestampsPersistCorrectly(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	repo := NewSQLiteRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	goal := mustMoney(t, 1000, "usd")
	f, err := fund.NewFund("fund-5", "trip-5", goal)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(f); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.Load("fund-5")
	if err != nil {
		t.Fatal(err)
	}
	// Timestamps are truncated to second precision (Unix epoch)
	if loaded.CreatedAt.Year() != time.Now().Year() {
		t.Fatalf("expected current year, got %d", loaded.CreatedAt.Year())
	}
}

// --- helpers ---

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	// Use a unique temp file per test to avoid cross-test pollution.
	tmpFile := "file:" + t.TempDir() + "/fund_test.db"
	db, err := sql.Open("libsql", tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func mustMoney(t *testing.T, amt int64, cur string) fund.Money {
	t.Helper()
	m, err := fund.NewMoney(amt, cur)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
