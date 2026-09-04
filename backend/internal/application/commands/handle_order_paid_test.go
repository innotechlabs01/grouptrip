package commands_test

import (
	"errors"
	"testing"
	"time"

	"github.com/frg/grouptrip/internal/application/commands"
	"github.com/frg/grouptrip/internal/domain/fund"
	"github.com/frg/grouptrip/internal/infrastructure/fundrepo"
)

func TestHandleOrderPaidHappyPath(t *testing.T) {
	fundRepo := fundrepo.NewMemoryRepo()
	contrRepo := newFakeContribRepo()
	sink := &recordingSink{}

	// Set up an active fund
	g, _ := fund.NewMoney(100000, "usd")
	f, err := fund.NewFund("f10", "t10", g)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	if err := fundRepo.Save(f); err != nil {
		t.Fatal(err)
	}

	// Create a contribution in PROCESSING state (as if ContributeCommand left it)
	amount, _ := fund.NewMoney(5000, "usd")
	c, _ := fund.NewContribution("c10", "", "f10", amount)
	_ = c.Authorize()
	_ = c.MarkProcessing()
	c.ExternalRef = "order_webhook_1"
	if err := contrRepo.Save(c); err != nil {
		t.Fatal(err)
	}

	h := commands.HandleOrderPaid{
		Funds:  fundRepo,
		Contrs: contrRepo,
		Events: sink,
	}

	err = h.Execute(commands.OrderPaidInput{
		OrderID:        "order_webhook_1",
		ContributionID: "c10",
		OccurredAt:     time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Contribution is now SUCCEEDED
	loaded, err := contrRepo.FindByID("c10")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != fund.ContrSucceeded {
		t.Fatalf("expected SUCCEEDED, got %s", loaded.Status)
	}

	// Fund ledger reflects collected
	f2, _ := fundRepo.Load("f10")
	if f2.Collected().Amount() != 5000 {
		t.Fatalf("expected collected 5000, got %d", f2.Collected().Amount())
	}

	// Events: ContributionSucceeded + FundUpdated
	if len(sink.got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(sink.got))
	}
	if sink.got[0].Type != "ContributionSucceeded" {
		t.Fatalf("expected ContributionSucceeded, got %s", sink.got[0].Type)
	}
	if sink.got[1].Type != "FundUpdated" {
		t.Fatalf("expected FundUpdated, got %s", sink.got[1].Type)
	}
}

func TestHandleOrderPaidIdempotentAlreadySucceeded(t *testing.T) {
	fundRepo := fundrepo.NewMemoryRepo()
	contrRepo := newFakeContribRepo()
	sink := &recordingSink{}

	g, _ := fund.NewMoney(100000, "usd")
	f, _ := fund.NewFund("f11", "t11", g)
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	_ = fundRepo.Save(f)

	// Already SUCCEEDED contribution
	amount, _ := fund.NewMoney(5000, "usd")
	c, _ := fund.NewContribution("c11", "", "f11", amount)
	_ = c.Authorize()
	_ = c.MarkProcessing()
	_ = c.Succeed("order_already_done")
	_ = contrRepo.Save(c)

	h := commands.HandleOrderPaid{
		Funds:  fundRepo,
		Contrs: contrRepo,
		Events: sink,
	}

	// Re-delivery — should be a no-op
	err := h.Execute(commands.OrderPaidInput{
		OrderID:        "order_already_done",
		ContributionID: "c11",
		OccurredAt:     time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// No events (already succeeded — no-op)
	if len(sink.got) != 0 {
		t.Fatalf("expected 0 events on idempotent re-delivery, got %d", len(sink.got))
	}
}

func TestHandleOrderPaidUnknownContribution(t *testing.T) {
	fundRepo := fundrepo.NewMemoryRepo()
	contrRepo := newFakeContribRepo()
	sink := &recordingSink{}

	h := commands.HandleOrderPaid{
		Funds:  fundRepo,
		Contrs: contrRepo,
		Events: sink,
	}

	err := h.Execute(commands.OrderPaidInput{
		OrderID:        "order_unknown",
		ContributionID: "c_nonexistent",
		OccurredAt:     time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for unknown contribution")
	}
}

func TestHandleOrderPaidFundAlreadyCollected(t *testing.T) {
	// Test I-5 idempotency: RecordCollected rejects duplicate contributionID
	fundRepo := fundrepo.NewMemoryRepo()
	contrRepo := newFakeContribRepo()
	sink := &recordingSink{}

	g, _ := fund.NewMoney(100000, "usd")
	f, _ := fund.NewFund("f12", "t12", g)
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	_ = fundRepo.Save(f)

	amount, _ := fund.NewMoney(5000, "usd")
	c, _ := fund.NewContribution("c12", "", "f12", amount)
	_ = c.Authorize()
	_ = c.MarkProcessing()
	c.ExternalRef = "order_dup_1"
	_ = contrRepo.Save(c)

	h := commands.HandleOrderPaid{
		Funds:  fundRepo,
		Contrs: contrRepo,
		Events: sink,
	}

	// First webhook delivery
	err := h.Execute(commands.OrderPaidInput{
		OrderID:        "order_dup_1",
		ContributionID: "c12",
		OccurredAt:     time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Load fresh copy of fund (in-memory repo stores by reference)
	f2, _ := fundRepo.Load("f12")

	// Second delivery — contribution is already SUCCEEDED, so HandleOrderPaid
	// should short-circuit before calling RecordCollected.
	err = h.Execute(commands.OrderPaidInput{
		OrderID:        "order_dup_1",
		ContributionID: "c12",
		OccurredAt:     time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Collected amount should not have changed
	f3, _ := fundRepo.Load("f12")
	if f3.Collected().Amount() != f2.Collected().Amount() {
		t.Fatalf("collected changed on idempotent re-delivery: %d -> %d",
			f2.Collected().Amount(), f3.Collected().Amount())
	}
}

func TestHandleOrderPaidFundNotActive(t *testing.T) {
	// Fund in OPEN state — RecordCollected should reject
	fundRepo := fundrepo.NewMemoryRepo()
	contrRepo := newFakeContribRepo()
	sink := &recordingSink{}

	g, _ := fund.NewMoney(100000, "usd")
	f, _ := fund.NewFund("f13", "t13", g)
	// No members, not activated — still OPEN
	_ = fundRepo.Save(f)

	amount, _ := fund.NewMoney(5000, "usd")
	c, _ := fund.NewContribution("c13", "", "f13", amount)
	_ = c.Authorize()
	_ = c.MarkProcessing()
	c.ExternalRef = "order_open_fund"
	_ = contrRepo.Save(c)

	h := commands.HandleOrderPaid{
		Funds:  fundRepo,
		Contrs: contrRepo,
		Events: sink,
	}

	err := h.Execute(commands.OrderPaidInput{
		OrderID:        "order_open_fund",
		ContributionID: "c13",
		OccurredAt:     time.Now(),
	})
	// Should return an error (fund not ACTIVE)
	if err == nil {
		t.Fatal("expected error for non-ACTIVE fund")
	}
}

func TestHandleOrderPaidFundLoadError(t *testing.T) {
	fundRepo := fundrepo.NewMemoryRepo()
	contrRepo := newFakeContribRepo()
	sink := &recordingSink{}

	// Contribution references a fund that doesn't exist
	amount, _ := fund.NewMoney(5000, "usd")
	c, _ := fund.NewContribution("c14", "", "f_nonexistent", amount)
	_ = c.Authorize()
	_ = c.MarkProcessing()
	c.ExternalRef = "order_no_fund"
	_ = contrRepo.Save(c)

	h := commands.HandleOrderPaid{
		Funds:  fundRepo,
		Contrs: contrRepo,
		Events: sink,
	}

	err := h.Execute(commands.OrderPaidInput{
		OrderID:        "order_no_fund",
		ContributionID: "c14",
		OccurredAt:     time.Now(),
	})
	if err == nil {
		t.Fatal("expected error when fund is missing")
	}
}

// helper to suppress unused import
var _ = errors.New
