package fund

import (
	"testing"
	"time"
)

func TestNewContributionPlanRequiresMethod(t *testing.T) {
	future := time.Now().AddDate(0, 2, 0)
	amt, _ := NewMoney(100000, "cop")
	// I-4 — missing payment method must be rejected
	if _, err := NewContributionPlan("p1", "f1", "u1", FrequencyMonthly, amt, "", future); err == nil {
		t.Fatal("expected error for missing payment method (I-4)")
	}
	plan, err := NewContributionPlan("p1", "f1", "u1", FrequencyMonthly, amt, "pm_123", future)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != PlanActive {
		t.Fatalf("expected ACTIVE, got %s", plan.Status)
	}
}

func TestComputeTotalExpected(t *testing.T) {
	amt, _ := NewMoney(100000, "cop")
	future := time.Now().AddDate(0, 0, 60)
	plan, err := NewContributionPlan("p1", "f1", "u1", FrequencyMonthly, amt, "pm_1", future)
	if err != nil {
		t.Fatal(err)
	}
	// 60 days -> 2 months -> 2 occurrences
	if err := plan.ComputeTotalExpected(60); err != nil {
		t.Fatal(err)
	}
	if plan.TotalExpected.Amount() != 200000 {
		t.Fatalf("expected total expected 200000, got %d", plan.TotalExpected.Amount())
	}
}

func TestPauseOnlyActive(t *testing.T) {
	amt, _ := NewMoney(100000, "cop")
	plan, err := NewContributionPlan("p1", "f1", "u1", FrequencyMonthly, amt, "pm_1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Pause(); err != nil {
		t.Fatal(err)
	}
	if plan.Status != PlanPaused {
		t.Fatalf("expected PAUSED, got %s", plan.Status)
	}
	// double pause should fail
	if err := plan.Pause(); err == nil {
		t.Fatal("expected error pausing non-active plan")
	}
}
