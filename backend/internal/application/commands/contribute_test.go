package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/frg/grouptrip/internal/application/commands"
	"github.com/frg/grouptrip/internal/domain/fund"
	"github.com/frg/grouptrip/internal/infrastructure/fundrepo"
	"github.com/frg/grouptrip/internal/infrastructure/payments"
)

// ---------------------------------------------------------------------------
// Fake ContributionRepository (in-memory)
// ---------------------------------------------------------------------------

type fakeContribRepo struct {
	store map[string]*fund.Contribution
}

func newFakeContribRepo() *fakeContribRepo {
	return &fakeContribRepo{store: make(map[string]*fund.Contribution)}
}

func (r *fakeContribRepo) Save(c *fund.Contribution) error {
	r.store[c.ID] = c
	return nil
}

func (r *fakeContribRepo) FindByID(id string) (*fund.Contribution, error) {
	c, ok := r.store[id]
	if !ok {
		return nil, errors.New("contribrepo: not found")
	}
	return c, nil
}

func (r *fakeContribRepo) FindByExternalRef(ref string) (*fund.Contribution, error) {
	for _, c := range r.store {
		if c.ExternalRef == ref {
			return c, nil
		}
	}
	return nil, errors.New("contribrepo: not found")
}

func (r *fakeContribRepo) FindNonTerminalByFundCredential(fundID, planID, externalRef string) (*fund.Contribution, error) {
	for _, c := range r.store {
		if c.FundID == fundID && c.PlanID == planID && c.ExternalRef == externalRef && !c.IsTerminal() {
			return c, nil
		}
	}
	return nil, errors.New("contribrepo: not found")
}

// ---------------------------------------------------------------------------
// Fake PaymentProvider (recording calls, controllable errors)
// ---------------------------------------------------------------------------

type fakePaymentProvider struct {
	createCustomerCalls     int
	createDraftOrderCalls   int
	finalizeDraftOrderCalls int
	failCustomerError       error
	failDraftError          error
	failFinalizeError       error
	nextOrderID             string
}

func (f *fakePaymentProvider) CreateCustomer(_ context.Context, externalID string) (string, error) {
	f.createCustomerCalls++
	if f.failCustomerError != nil {
		return "", f.failCustomerError
	}
	return "cust_fake_1", nil
}

func (f *fakePaymentProvider) SavePaymentMethod(_ context.Context, _, _ string) error { return nil }

func (f *fakePaymentProvider) CreateDraftOrder(_ context.Context, in payments.DraftOrderInput) (string, error) {
	f.createDraftOrderCalls++
	if f.failDraftError != nil {
		return "", f.failDraftError
	}
	if f.nextOrderID != "" {
		return f.nextOrderID, nil
	}
	return "order_fake_1", nil
}

func (f *fakePaymentProvider) FinalizeDraftOrder(_ context.Context, orderID, _ string) (string, error) {
	f.finalizeDraftOrderCalls++
	if f.failFinalizeError != nil {
		return "", f.failFinalizeError
	}
	return orderID, nil
}

func (f *fakePaymentProvider) Refund(_ context.Context, _ string, _ int64) error { return nil }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestContributeHappyPath(t *testing.T) {
	fundRepo := fundrepo.NewMemoryRepo()
	contrRepo := newFakeContribRepo()
	pay := &fakePaymentProvider{nextOrderID: "order_happy_1"}
	sink := &recordingSink{}

	// Set up an active fund
	g, _ := fund.NewMoney(100000, "usd")
	f, err := fund.NewFund("f1", "t1", g)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	if err := fundRepo.Save(f); err != nil {
		t.Fatal(err)
	}

	h := commands.ContributeCommand{
		Funds:    fundRepo,
		Contrs:   contrRepo,
		Payments: pay,
		Events:   sink,
	}

	err = h.Execute(commands.ContributeInput{
		ContributionID: "cont-1",
		FundID:         "f1",
		ProductID:      "prod_x",
		CustomerEmail:  "user@example.com",
		Amount:         5000,
		Currency:       "usd",
		Description:    "Monthly contribution",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify contribution persisted as PROCESSING
	c, err := contrRepo.FindByID("cont-1")
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != fund.ContrProcessing {
		t.Fatalf("expected PROCESSING, got %s", c.Status)
	}
	if c.ExternalRef != "order_happy_1" {
		t.Fatalf("expected external_ref order_happy_1, got %s", c.ExternalRef)
	}

	// Verify provider calls: CreateCustomer + CreateDraftOrder + FinalizeDraftOrder
	if pay.createCustomerCalls != 1 {
		t.Fatalf("expected 1 CreateCustomer call, got %d", pay.createCustomerCalls)
	}
	if pay.createDraftOrderCalls != 1 {
		t.Fatalf("expected 1 CreateDraftOrder call, got %d", pay.createDraftOrderCalls)
	}
	if pay.finalizeDraftOrderCalls != 1 {
		t.Fatalf("expected 1 FinalizeDraftOrder call, got %d", pay.finalizeDraftOrderCalls)
	}

	// Verify events published
	if len(sink.got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(sink.got))
	}
	if sink.got[0].Type != "ContributionProcessing" {
		t.Fatalf("expected ContributionProcessing event, got %s", sink.got[0].Type)
	}
}

func TestContributeIdempotentRebidNonTerminal(t *testing.T) {
	fundRepo := fundrepo.NewMemoryRepo()
	contrRepo := newFakeContribRepo()
	pay := &fakePaymentProvider{nextOrderID: "order_existing"}
	sink := &recordingSink{}

	// Set up an active fund
	g, _ := fund.NewMoney(100000, "usd")
	f, err := fund.NewFund("f2", "t2", g)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	if err := fundRepo.Save(f); err != nil {
		t.Fatal(err)
	}

	// Pre-create a contribution that is already in-flight (AUTHORIZED)
	amount, _ := fund.NewMoney(5000, "usd")
	existing, _ := fund.NewContribution("cont-2", "", "f2", amount)
	_ = existing.Authorize()
	existing.ExternalRef = "order_existing"
	_ = contrRepo.Save(existing)

	h := commands.ContributeCommand{
		Funds:    fundRepo,
		Contrs:   contrRepo,
		Payments: pay,
		Events:   sink,
	}

	err = h.Execute(commands.ContributeInput{
		ContributionID: "cont-2",
		FundID:         "f2",
		ProductID:      "prod_x",
		CustomerEmail:  "user@example.com",
		Amount:         5000,
		Currency:       "usd",
		Description:    "Monthly contribution",
	})
	if err != nil {
		t.Fatal(err)
	}

	// No new provider calls — idempotent no-op
	if pay.createCustomerCalls != 0 {
		t.Fatalf("expected 0 CreateCustomer calls, got %d", pay.createCustomerCalls)
	}
	if pay.createDraftOrderCalls != 0 {
		t.Fatalf("expected 0 CreateDraftOrder calls, got %d", pay.createDraftOrderCalls)
	}
	if pay.finalizeDraftOrderCalls != 0 {
		t.Fatalf("expected 0 FinalizeDraftOrder calls, got %d", pay.finalizeDraftOrderCalls)
	}

	// Contribution status unchanged (still AUTHORIZED)
	c, _ := contrRepo.FindByID("cont-2")
	if c.Status != fund.ContrAuthorized {
		t.Fatalf("expected AUTHORIZED (unchanged), got %s", c.Status)
	}

	// No events published (idempotent)
	if len(sink.got) != 0 {
		t.Fatalf("expected 0 events, got %d", len(sink.got))
	}
}

func TestContributeTerminalContributionRefusesRecharge(t *testing.T) {
	fundRepo := fundrepo.NewMemoryRepo()
	contrRepo := newFakeContribRepo()
	pay := &fakePaymentProvider{nextOrderID: "order_terminal"}
	sink := &recordingSink{}

	g, _ := fund.NewMoney(100000, "usd")
	f, err := fund.NewFund("f3", "t3", g)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	_ = fundRepo.Save(f)

	// Pre-create a terminal (SUCCEEDED) contribution
	amount, _ := fund.NewMoney(5000, "usd")
	existing, _ := fund.NewContribution("cont-3", "", "f3", amount)
	_ = existing.Authorize()
	_ = existing.MarkProcessing()
	_ = existing.Succeed("order_terminal")
	_ = contrRepo.Save(existing)

	h := commands.ContributeCommand{
		Funds:    fundRepo,
		Contrs:   contrRepo,
		Payments: pay,
		Events:   sink,
	}

	err = h.Execute(commands.ContributeInput{
		ContributionID: "cont-3",
		FundID:         "f3",
		ProductID:      "prod_x",
		CustomerEmail:  "user@example.com",
		Amount:         5000,
		Currency:       "usd",
	})
	if err == nil {
		t.Fatal("expected error for terminal contribution re-charge attempt")
	}

	// No provider calls
	if pay.createCustomerCalls != 0 || pay.createDraftOrderCalls != 0 {
		t.Fatalf("expected 0 provider calls, got create_customer=%d create_draft=%d",
			pay.createCustomerCalls, pay.createDraftOrderCalls)
	}
}

func TestContributeCustomerCreationFailureMarksFailed(t *testing.T) {
	fundRepo := fundrepo.NewMemoryRepo()
	contrRepo := newFakeContribRepo()
	pay := &fakePaymentProvider{failCustomerError: errors.New("polar: customer creation error")}
	sink := &recordingSink{}

	g, _ := fund.NewMoney(100000, "usd")
	f, err := fund.NewFund("f4", "t4", g)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	_ = fundRepo.Save(f)

	h := commands.ContributeCommand{
		Funds:    fundRepo,
		Contrs:   contrRepo,
		Payments: pay,
		Events:   sink,
	}

	err = h.Execute(commands.ContributeInput{
		ContributionID: "cont-4",
		FundID:         "f4",
		ProductID:      "prod_x",
		CustomerEmail:  "user@example.com",
		Amount:         5000,
		Currency:       "usd",
	})
	if err == nil {
		t.Fatal("expected error when CreateCustomer fails")
	}

	// Contribution persisted as FAILED
	c, _ := contrRepo.FindByID("cont-4")
	if c.Status != fund.ContrFailed {
		t.Fatalf("expected FAILED after customer creation error, got %s", c.Status)
	}

	// No draft or finalize calls
	if pay.createDraftOrderCalls != 0 {
		t.Fatalf("expected 0 draft calls, got %d", pay.createDraftOrderCalls)
	}
}

func TestContributeDraftOrderFailureMarksFailed(t *testing.T) {
	fundRepo := fundrepo.NewMemoryRepo()
	contrRepo := newFakeContribRepo()
	pay := &fakePaymentProvider{failDraftError: errors.New("polar: draft order error")}
	sink := &recordingSink{}

	g, _ := fund.NewMoney(100000, "usd")
	f, err := fund.NewFund("f5", "t5", g)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	_ = fundRepo.Save(f)

	h := commands.ContributeCommand{
		Funds:    fundRepo,
		Contrs:   contrRepo,
		Payments: pay,
		Events:   sink,
	}

	err = h.Execute(commands.ContributeInput{
		ContributionID: "cont-5",
		FundID:         "f5",
		ProductID:      "prod_x",
		CustomerEmail:  "user@example.com",
		Amount:         5000,
		Currency:       "usd",
	})
	if err == nil {
		t.Fatal("expected error when CreateDraftOrder fails")
	}

	c, _ := contrRepo.FindByID("cont-5")
	if c.Status != fund.ContrFailed {
		t.Fatalf("expected FAILED after draft creation error, got %s", c.Status)
	}
	if pay.finalizeDraftOrderCalls != 0 {
		t.Fatalf("expected 0 finalize calls, got %d", pay.finalizeDraftOrderCalls)
	}
}

func TestContributeFinalizeCardDeclinedMarksFailed(t *testing.T) {
	fundRepo := fundrepo.NewMemoryRepo()
	contrRepo := newFakeContribRepo()
	pay := &fakePaymentProvider{failFinalizeError: payments.ErrCardDeclined}
	sink := &recordingSink{}

	g, _ := fund.NewMoney(100000, "usd")
	f, err := fund.NewFund("f6", "t6", g)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	_ = fundRepo.Save(f)

	h := commands.ContributeCommand{
		Funds:    fundRepo,
		Contrs:   contrRepo,
		Payments: pay,
		Events:   sink,
	}

	err = h.Execute(commands.ContributeInput{
		ContributionID: "cont-6",
		FundID:         "f6",
		ProductID:      "prod_x",
		CustomerEmail:  "user@example.com",
		Amount:         5000,
		Currency:       "usd",
	})
	if err == nil {
		t.Fatal("expected error when finalize returns ErrCardDeclined")
	}

	c, _ := contrRepo.FindByID("cont-6")
	if c.Status != fund.ContrFailed {
		t.Fatalf("expected FAILED after card_declined, got %s", c.Status)
	}
}

func TestContributeWithPreExistingCustomerID(t *testing.T) {
	fundRepo := fundrepo.NewMemoryRepo()
	contrRepo := newFakeContribRepo()
	pay := &fakePaymentProvider{nextOrderID: "order_pre_cust"}
	sink := &recordingSink{}

	g, _ := fund.NewMoney(100000, "usd")
	f, err := fund.NewFund("f7", "t7", g)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	_ = fundRepo.Save(f)

	h := commands.ContributeCommand{
		Funds:    fundRepo,
		Contrs:   contrRepo,
		Payments: pay,
		Events:   sink,
	}

	err = h.Execute(commands.ContributeInput{
		ContributionID: "cont-7",
		FundID:         "f7",
		CustomerID:     "cust_existing_123", // pre-created
		ProductID:      "prod_x",
		CustomerEmail:  "user@example.com",
		Amount:         5000,
		Currency:       "usd",
	})
	if err != nil {
		t.Fatal(err)
	}

	// CreateCustomer should NOT be called when CustomerID is provided
	if pay.createCustomerCalls != 0 {
		t.Fatalf("expected 0 CreateCustomer calls when CustomerID pre-set, got %d", pay.createCustomerCalls)
	}

	c, _ := contrRepo.FindByID("cont-7")
	if c.Status != fund.ContrProcessing {
		t.Fatalf("expected PROCESSING, got %s", c.Status)
	}
}

func TestContributeValidationRejectsMissingFields(t *testing.T) {
	h := commands.ContributeCommand{}

	// Empty ContributionID
	err := h.Execute(commands.ContributeInput{
		FundID:        "f1",
		ProductID:     "p1",
		CustomerEmail: "a@b.com",
		Amount:        100,
		Currency:      "usd",
	})
	if err == nil {
		t.Fatal("expected error for empty ContributionID")
	}

	// Zero Amount
	err = h.Execute(commands.ContributeInput{
		ContributionID: "c1",
		FundID:         "f1",
		ProductID:      "p1",
		CustomerEmail:  "a@b.com",
		Amount:         0,
		Currency:       "usd",
	})
	if err == nil {
		t.Fatal("expected error for zero Amount")
	}
}

func TestContributeIdempotentRebidInProgress(t *testing.T) {
	// Second call while contribution is PENDING (no ExternalRef yet)
	fundRepo := fundrepo.NewMemoryRepo()
	contrRepo := newFakeContribRepo()
	pay := &fakePaymentProvider{nextOrderID: "order_inflight"}
	sink := &recordingSink{}

	g, _ := fund.NewMoney(100000, "usd")
	f, err := fund.NewFund("f8", "t8", g)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.AddMember("u1")
	_ = f.Activate()
	_ = fundRepo.Save(f)

	// Pre-create a PENDING contribution (in-flight, no external ref)
	amount, _ := fund.NewMoney(5000, "usd")
	existing, _ := fund.NewContribution("cont-8", "", "f8", amount)
	// Still PENDING — no transitions called
	_ = contrRepo.Save(existing)

	h := commands.ContributeCommand{
		Funds:    fundRepo,
		Contrs:   contrRepo,
		Payments: pay,
		Events:   sink,
	}

	err = h.Execute(commands.ContributeInput{
		ContributionID: "cont-8",
		FundID:         "f8",
		ProductID:      "prod_x",
		CustomerEmail:  "user@example.com",
		Amount:         5000,
		Currency:       "usd",
	})
	if err != nil {
		t.Fatal(err)
	}

	// No new provider calls
	if pay.createCustomerCalls != 0 || pay.createDraftOrderCalls != 0 {
		t.Fatalf("expected 0 provider calls, got create_customer=%d create_draft=%d",
			pay.createCustomerCalls, pay.createDraftOrderCalls)
	}
}
