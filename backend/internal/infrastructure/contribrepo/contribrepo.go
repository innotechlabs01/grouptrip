// Package contribrepo provides persistence for Contribution entities.
package contribrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/frg/grouptrip/internal/domain/fund"
)

// ErrNotFound indicates no matching contribution was found.
var ErrNotFound = errors.New("contribrepo: not found")

// SQLiteContribRepo persists *fund.Contribution rows in a libsql/SQLite database.
type SQLiteContribRepo struct {
	db *sql.DB
}

// NewSQLiteContribRepo creates a contribution repo backed by the given connection.
func NewSQLiteContribRepo(db *sql.DB) *SQLiteContribRepo {
	return &SQLiteContribRepo{db: db}
}

// Migrate creates the contributions table if it does not exist.
func (r *SQLiteContribRepo) Migrate() error {
	ctx := context.Background()
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS contributions (
			id           TEXT PRIMARY KEY,
			plan_id      TEXT NOT NULL,
			fund_id      TEXT NOT NULL,
			amount       INTEGER NOT NULL,
			currency     TEXT NOT NULL,
			status       TEXT NOT NULL,
			external_ref TEXT NOT NULL DEFAULT '',
			created_at   INTEGER NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("contribrepo migrate: %w", err)
	}
	return nil
}

// Save persists a Contribution, upserting on id conflict.
func (r *SQLiteContribRepo) Save(c *fund.Contribution) error {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("contribrepo save begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(ctx, `
		INSERT INTO contributions (id, plan_id, fund_id, amount, currency, status, external_ref, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			plan_id      = excluded.plan_id,
			fund_id      = excluded.fund_id,
			amount       = excluded.amount,
			currency     = excluded.currency,
			status       = excluded.status,
			external_ref = excluded.external_ref,
			created_at   = excluded.created_at
	`,
		c.ID, c.PlanID, c.FundID,
		c.Amount.Amount(), c.Amount.Currency(),
		string(c.Status), c.ExternalRef,
		c.CreatedAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("contribrepo save: %w", err)
	}

	return tx.Commit()
}

// FindByID retrieves a Contribution by its primary key.
// Returns ErrNotFound when no matching row exists.
func (r *SQLiteContribRepo) FindByID(id string) (*fund.Contribution, error) {
	return r.findByColumn("id", id)
}

// FindByExternalRef retrieves a Contribution by its provider order reference.
// Returns ErrNotFound when no matching row exists.
func (r *SQLiteContribRepo) FindByExternalRef(ref string) (*fund.Contribution, error) {
	if ref == "" {
		return nil, ErrNotFound
	}
	return r.findByColumn("external_ref", ref)
}

// FindNonTerminalByFundCredential finds a non-terminal contribution for the given
// fund+plan+externalRef. This is used for idempotent re-try: if a contribution
// is already in-flight (PENDING/AUTHORIZED/PROCESSING) with the same credentials,
// the caller reuses it instead of creating a new draft.
func (r *SQLiteContribRepo) FindNonTerminalByFundCredential(fundID, planID, externalRef string) (*fund.Contribution, error) {
	ctx := context.Background()

	var (
		dbID, dbPlanID, dbFundID, dbCurrency, dbStatus, dbExternalRef string
		dbAmount                                                      int64
		dbCreatedAtUnix                                               int64
	)

	err := r.db.QueryRowContext(ctx, `
		SELECT id, plan_id, fund_id, amount, currency, status, external_ref, created_at
		FROM contributions
		WHERE fund_id = ? AND plan_id = ? AND external_ref = ?
		  AND status NOT IN ('SUCCEEDED', 'FAILED', 'REFUNDED')
		ORDER BY created_at DESC
		LIMIT 1
	`, fundID, planID, externalRef).Scan(
		&dbID, &dbPlanID, &dbFundID, &dbAmount, &dbCurrency,
		&dbStatus, &dbExternalRef, &dbCreatedAtUnix,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("contribrepo find non-terminal: %w", err)
	}

	amountMoney := fund.NewRawMoney(dbAmount, dbCurrency)
	return hydrateContribution(dbID, dbPlanID, dbFundID, amountMoney, fund.ContributionStatus(dbStatus), dbExternalRef, dbCreatedAtUnix)
}

// findByColumn is a shared helper for single-column lookups.
func (r *SQLiteContribRepo) findByColumn(column, value string) (*fund.Contribution, error) {
	ctx := context.Background()

	var (
		dbID, dbPlanID, dbFundID, dbCurrency, dbStatus, dbExternalRef string
		dbAmount                                                      int64
		dbCreatedAtUnix                                               int64
	)

	err := r.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT id, plan_id, fund_id, amount, currency, status, external_ref, created_at
		FROM contributions WHERE %s = ?
	`, column), value).Scan(
		&dbID, &dbPlanID, &dbFundID, &dbAmount, &dbCurrency,
		&dbStatus, &dbExternalRef, &dbCreatedAtUnix,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("contribrepo find by %s: %w", column, err)
	}

	amountMoney := fund.NewRawMoney(dbAmount, dbCurrency)
	return hydrateContribution(dbID, dbPlanID, dbFundID, amountMoney, fund.ContributionStatus(dbStatus), dbExternalRef, dbCreatedAtUnix)
}

// hydrateContribution reconstructs a *fund.Contribution from persisted fields,
// advancing through the state machine to reach the stored status.
func hydrateContribution(id, planID, fundID string, amount fund.Money, status fund.ContributionStatus, externalRef string, createdAtUnix int64) (*fund.Contribution, error) {
	c, err := fund.NewContribution(id, planID, fundID, amount)
	if err != nil {
		return nil, fmt.Errorf("contribrepo hydrate: %w", err)
	}

	// Advance through the state machine to reach the persisted status.
	// PENDING is already the initial state set by NewContribution.
	switch status {
	case fund.ContrPending:
		// Already in PENDING — no-op.
	case fund.ContrAuthorized:
		if err := c.Authorize(); err != nil {
			return nil, fmt.Errorf("contribrepo hydrate authorize: %w", err)
		}
		c.ExternalRef = externalRef
	case fund.ContrProcessing:
		if err := c.Authorize(); err != nil {
			return nil, fmt.Errorf("contribrepo hydrate authorize: %w", err)
		}
		if err := c.MarkProcessing(); err != nil {
			return nil, fmt.Errorf("contribrepo hydrate mark_processing: %w", err)
		}
		c.ExternalRef = externalRef
	case fund.ContrSucceeded:
		if err := c.Authorize(); err != nil {
			return nil, fmt.Errorf("contribrepo hydrate authorize: %w", err)
		}
		if err := c.MarkProcessing(); err != nil {
			return nil, fmt.Errorf("contribrepo hydrate mark_processing: %w", err)
		}
		if err := c.Succeed(externalRef); err != nil {
			return nil, fmt.Errorf("contribrepo hydrate succeed: %w", err)
		}
	case fund.ContrFailed:
		if err := c.Fail(""); err != nil {
			return nil, fmt.Errorf("contribrepo hydrate fail: %w", err)
		}
	case fund.ContrRefunded:
		// Hydrate to SUCCEEDED first, then Refund.
		if err := c.Authorize(); err != nil {
			return nil, fmt.Errorf("contribrepo hydrate authorize: %w", err)
		}
		if err := c.MarkProcessing(); err != nil {
			return nil, fmt.Errorf("contribrepo hydrate mark_processing: %w", err)
		}
		if err := c.Succeed(externalRef); err != nil {
			return nil, fmt.Errorf("contribrepo hydrate succeed: %w", err)
		}
		if err := c.Refund(); err != nil {
			return nil, fmt.Errorf("contribrepo hydrate refund: %w", err)
		}
	default:
		return nil, fmt.Errorf("contribrepo hydrate: unknown status %s", status)
	}

	// Set the persisted created_at (truncated to second precision in the DB).
	c.CreatedAt = time.Unix(createdAtUnix, 0).UTC()
	return c, nil
}
