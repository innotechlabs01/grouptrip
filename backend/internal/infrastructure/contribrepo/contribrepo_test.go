package contribrepo

import (
	"database/sql"
	"testing"

	"github.com/frg/grouptrip/internal/domain/fund"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("libsql", "file:"+t.TempDir()+"/contrib_test.db")
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

func TestSaveAndFindByIDRoundTrip(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteContribRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	amount := mustMoney(t, 2500, "usd")
	c, err := fund.NewContribution("c1", "plan-1", "fund-1", amount)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Authorize(); err != nil {
		t.Fatal(err)
	}
	c.ExternalRef = "order_xyz"

	if err := repo.Save(c); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByID("c1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "c1" {
		t.Fatalf("expected id c1, got %s", got.ID)
	}
	if got.PlanID != "plan-1" {
		t.Fatalf("expected plan_id plan-1, got %s", got.PlanID)
	}
	if got.FundID != "fund-1" {
		t.Fatalf("expected fund_id fund-1, got %s", got.FundID)
	}
	if got.Amount.Amount() != 2500 {
		t.Fatalf("expected amount 2500, got %d", got.Amount.Amount())
	}
	if got.Amount.Currency() != "usd" {
		t.Fatalf("expected currency usd, got %s", got.Amount.Currency())
	}
	if got.Status != fund.ContrAuthorized {
		t.Fatalf("expected status AUTHORIZED, got %s", got.Status)
	}
	if got.ExternalRef != "order_xyz" {
		t.Fatalf("expected external_ref order_xyz, got %s", got.ExternalRef)
	}
}

func TestSaveAndFindByIDPendingStatus(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteContribRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	amount := mustMoney(t, 1000, "cop")
	c, err := fund.NewContribution("c2", "plan-2", "fund-2", amount)
	if err != nil {
		t.Fatal(err)
	}
	// Leave in PENDING (no state transitions)

	if err := repo.Save(c); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByID("c2")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != fund.ContrPending {
		t.Fatalf("expected PENDING, got %s", got.Status)
	}
	if got.ExternalRef != "" {
		t.Fatalf("expected empty external_ref, got %s", got.ExternalRef)
	}
}

func TestFindByIDNotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteContribRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	_, err := repo.FindByID("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent contribution")
	}
}

func TestFindByExternalRef(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteContribRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	amount := mustMoney(t, 5000, "usd")
	c, err := fund.NewContribution("c3", "", "fund-3", amount)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Authorize(); err != nil {
		t.Fatal(err)
	}
	if err := c.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	if err := c.Succeed("order_ext_1"); err != nil {
		t.Fatal(err)
	}

	if err := repo.Save(c); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByExternalRef("order_ext_1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "c3" {
		t.Fatalf("expected id c3, got %s", got.ID)
	}
	if got.Status != fund.ContrSucceeded {
		t.Fatalf("expected SUCCEEDED, got %s", got.Status)
	}
}

func TestFindByExternalRefNotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteContribRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	_, err := repo.FindByExternalRef("no_such_order")
	if err == nil {
		t.Fatal("expected error for nonexistent external ref")
	}
}

func TestFindByIDProcessingStatus(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteContribRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	amount := mustMoney(t, 100, "usd")
	c, err := fund.NewContribution("c4", "p4", "f4", amount)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Authorize(); err != nil {
		t.Fatal(err)
	}
	if err := c.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	c.ExternalRef = "order_p4"

	if err := repo.Save(c); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByID("c4")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != fund.ContrProcessing {
		t.Fatalf("expected PROCESSING, got %s", got.Status)
	}
	if got.ExternalRef != "order_p4" {
		t.Fatalf("expected external_ref order_p4, got %s", got.ExternalRef)
	}
}

func TestFindByIDFailedStatus(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteContribRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	amount := mustMoney(t, 100, "usd")
	c, err := fund.NewContribution("c5", "", "f5", amount)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Fail("card_declined"); err != nil {
		t.Fatal(err)
	}

	if err := repo.Save(c); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByID("c5")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != fund.ContrFailed {
		t.Fatalf("expected FAILED, got %s", got.Status)
	}
}

func TestFindNonTerminalByFundCredential(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteContribRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	// Create a non-terminal contribution (AUTHORIZED) for fund-6, plan-6
	amount := mustMoney(t, 3000, "usd")
	c, err := fund.NewContribution("c6a", "plan-6", "fund-6", amount)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Authorize(); err != nil {
		t.Fatal(err)
	}
	c.ExternalRef = "order_6a"
	if err := repo.Save(c); err != nil {
		t.Fatal(err)
	}

	// Find it via non-terminal query
	got, err := repo.FindNonTerminalByFundCredential("fund-6", "plan-6", "order_6a")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "c6a" {
		t.Fatalf("expected c6a, got %s", got.ID)
	}
	if got.Status != fund.ContrAuthorized {
		t.Fatalf("expected AUTHORIZED, got %s", got.Status)
	}
}

func TestFindNonTerminalByFundCredentialIgnoresTerminal(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteContribRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	// Create a terminal (SUCCEEDED) contribution
	amount := mustMoney(t, 3000, "usd")
	c, err := fund.NewContribution("c7a", "plan-7", "fund-7", amount)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Authorize(); err != nil {
		t.Fatal(err)
	}
	if err := c.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	if err := c.Succeed("order_7a"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(c); err != nil {
		t.Fatal(err)
	}

	// Should NOT be found (terminal is excluded)
	_, err = repo.FindNonTerminalByFundCredential("fund-7", "plan-7", "order_7a")
	if err == nil {
		t.Fatal("expected ErrNotFound for terminal contribution")
	}
}

func TestFindNonTerminalByFundCredentialEmptyPlanID(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteContribRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	// One-off contribution (empty planID)
	amount := mustMoney(t, 1500, "cop")
	c, err := fund.NewContribution("c8a", "", "fund-8", amount)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Authorize(); err != nil {
		t.Fatal(err)
	}
	c.ExternalRef = "order_8a"
	if err := repo.Save(c); err != nil {
		t.Fatal(err)
	}

	// Search with empty planID
	got, err := repo.FindNonTerminalByFundCredential("fund-8", "", "order_8a")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "c8a" {
		t.Fatalf("expected c8a, got %s", got.ID)
	}
}

func TestMigrationIdempotency(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteContribRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	// Second call must succeed
	if err := repo.Migrate(); err != nil {
		t.Fatalf("second Migrate call failed: %v", err)
	}
	// Third call too
	if err := repo.Migrate(); err != nil {
		t.Fatalf("third Migrate call failed: %v", err)
	}
}

func TestSaveUpsertUpdatesFields(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteContribRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	amount := mustMoney(t, 100, "usd")
	c, err := fund.NewContribution("c9", "", "f9", amount)
	if err != nil {
		t.Fatal(err)
	}
	// Start as PENDING, save
	if err := repo.Save(c); err != nil {
		t.Fatal(err)
	}

	// Advance to AUTHORIZED + set ExternalRef, save again
	if err := c.Authorize(); err != nil {
		t.Fatal(err)
	}
	c.ExternalRef = "order_updated"
	if err := repo.Save(c); err != nil {
		t.Fatal(err)
	}

	// Reload: should reflect updated state
	got, err := repo.FindByID("c9")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != fund.ContrAuthorized {
		t.Fatalf("expected AUTHORIZED after upsert, got %s", got.Status)
	}
	if got.ExternalRef != "order_updated" {
		t.Fatalf("expected external_ref order_updated after upsert, got %s", got.ExternalRef)
	}
}

func TestSaveAndFindByIDTimestampPreserved(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteContribRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}

	amount := mustMoney(t, 100, "usd")
	c, err := fund.NewContribution("c10", "", "f10", amount)
	if err != nil {
		t.Fatal(err)
	}

	createdBefore := c.CreatedAt.Unix()

	if err := repo.Save(c); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByID("c10")
	if err != nil {
		t.Fatal(err)
	}
	// CreatedAt is truncated to second precision via Unix epoch.
	// Compare as Unix seconds to avoid timezone mismatches.
	if got.CreatedAt.Unix() != createdBefore {
		t.Fatalf("expected created_at unix %d, got %d", createdBefore, got.CreatedAt.Unix())
	}
}
