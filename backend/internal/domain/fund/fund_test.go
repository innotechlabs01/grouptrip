package fund

import "testing"

func TestNewFundEnforcesPositiveGoal(t *testing.T) {
	goal, _ := NewMoney(100, "cop")
	f, err := NewFund("f1", "t1", goal)
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusOpen {
		t.Fatalf("expected OPEN, got %s", f.Status)
	}
	// I-1
	zero, _ := NewMoney(0, "cop")
	if _, err := NewFund("f2", "t1", zero); err == nil {
		t.Fatal("expected error for zero goal (I-1)")
	}
}

func TestAddMemberRecomputesPerPersonTarget(t *testing.T) {
	f, _ := NewFund("f1", "t1", mustMoney(t, 1200, "cop"))
	if _, err := f.AddMember("u1"); err != nil {
		t.Fatal(err)
	}
	if f.Members[0].PerPersonTarget.Amount() != 1200 {
		t.Fatalf("expected per-person 1200, got %d", f.Members[0].PerPersonTarget.Amount())
	}
	if _, err := f.AddMember("u2"); err != nil {
		t.Fatal(err)
	}
	// per-person now 600
	if f.Members[1].PerPersonTarget.Amount() != 600 {
		t.Fatalf("expected per-person 600, got %d", f.Members[1].PerPersonTarget.Amount())
	}
}

func TestAddMemberRejectsDuplicate(t *testing.T) {
	f, _ := NewFund("f1", "t1", mustMoney(t, 1000, "cop"))
	_, _ = f.AddMember("u1")
	if _, err := f.AddMember("u1"); err == nil {
		t.Fatal("expected duplicate member error")
	}
}

func TestActivateRequiresParticipants(t *testing.T) {
	f, _ := NewFund("f1", "t1", mustMoney(t, 1000, "cop"))
	// No members -> cannot activate (I-2 guard)
	if err := f.Activate(); err == nil {
		t.Fatal("expected error activating without participants")
	}
	_, _ = f.AddMember("u1")
	if err := f.Activate(); err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusActive {
		t.Fatalf("expected ACTIVE, got %s", f.Status)
	}
}

func TestRecordCollectedMarksFunded(t *testing.T) {
	f, _ := NewFund("f1", "t1", mustMoney(t, 1000, "cop"))
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	amt, _ := NewMoney(1000, "cop")
	if err := f.RecordCollected(amt, "c1"); err != nil {
		t.Fatal(err)
	}
	if f.Collected().Amount() != 1000 {
		t.Fatalf("expected collected 1000, got %d", f.Collected().Amount())
	}
	// collected == goal -> FUNDED
	if f.Status != StatusFunded {
		t.Fatalf("expected FUNDED, got %s", f.Status)
	}
}

func TestRecordCollectedRequiresActive(t *testing.T) {
	f, _ := NewFund("f1", "t1", mustMoney(t, 1000, "cop"))
	// Not activated; also no member required here but status guards
	amt, _ := NewMoney(100, "cop")
	if err := f.RecordCollected(amt, "c1"); err == nil {
		t.Fatal("expected error recording collected on non-active fund")
	}
}

func TestRecordCollectedIdempotentByContribution(t *testing.T) {
	f, _ := NewFund("f1", "t1", mustMoney(t, 2000, "cop"))
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	if err := f.RecordCollected(mustMoney(t, 500, "cop"), "c1"); err != nil {
		t.Fatal(err)
	}
	// Re-recording the same contribution must be rejected (I-5): never double-count.
	if err := f.RecordCollected(mustMoney(t, 500, "cop"), "c1"); err == nil {
		t.Fatal("expected error recording duplicate contribution_id")
	}
	if f.Collected().Amount() != 500 {
		t.Fatalf("expected collected 500 (duplicate rejected), got %d", f.Collected().Amount())
	}
}

func TestLedgerIsAppendOnly(t *testing.T) {
	f, _ := NewFund("f1", "t1", mustMoney(t, 1000, "cop"))
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	_ = f.RecordCollected(mustMoney(t, 400, "cop"), "c1")
	_ = f.RecordCollected(mustMoney(t, 200, "cop"), "c2")
	if n := len(f.Ledger()); n != 2 {
		t.Fatalf("expected 2 ledger entries, got %d", n)
	}
	if f.Collected().Amount() != 600 {
		t.Fatalf("expected collected 600, got %d", f.Collected().Amount())
	}
}

func mustMoney(t *testing.T, amt int64, cur string) Money {
	t.Helper()
	m, err := NewMoney(amt, cur)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
