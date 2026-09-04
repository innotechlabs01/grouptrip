package fund

import "testing"

func TestContributionLifecycle(t *testing.T) {
	amt, _ := NewMoney(100000, "cop")
	c, err := NewContribution("c1", "p1", "f1", amt)
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != ContrPending {
		t.Fatalf("expected PENDING, got %s", c.Status)
	}
	_ = c.Authorize()
	if c.Status != ContrAuthorized {
		t.Fatalf("expected AUTHORIZED, got %s", c.Status)
	}
	_ = c.MarkProcessing()
	if c.Status != ContrProcessing {
		t.Fatalf("expected PROCESSING, got %s", c.Status)
	}
	if err := c.Succeed("order_123"); err != nil {
		t.Fatal(err)
	}
	if c.Status != ContrSucceeded {
		t.Fatalf("expected SUCCEEDED, got %s", c.Status)
	}
	if !c.IsTerminal() {
		t.Fatal("expected terminal state")
	}
}

func TestContributionNoShortcuts(t *testing.T) {
	amt, _ := NewMoney(1000, "cop")
	c, _ := NewContribution("c1", "p1", "f1", amt)
	// Cannot succeed from PENDING (I-3: only provider-confirmed processing path)
	if err := c.Succeed("order_1"); err == nil {
		t.Fatal("expected error succeeding a non-processing contribution")
	}
	// Cannot refund before success
	if err := c.Refund(); err == nil {
		t.Fatal("expected error refunding non-succeeded contribution")
	}
}
