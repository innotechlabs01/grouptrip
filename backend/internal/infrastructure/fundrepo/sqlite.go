// Package fundrepo provides persistence implementations for the Fund aggregate.
package fundrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/frg/grouptrip/internal/domain/fund"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite" // pure-Go sqlite driver, required by libsql for file: DSNs
)

// SQLiteRepo persists Fund aggregates in a libsql / SQLite database.
// Implements the same interface as MemoryRepo: Save + Load.
type SQLiteRepo struct {
	db *sql.DB
}

// NewSQLiteRepo creates a SQLiteRepo backed by the given database connection.
func NewSQLiteRepo(db *sql.DB) *SQLiteRepo {
	return &SQLiteRepo{db: db}
}

// OpenTurso opens a database connection. If TURSO_DATABASE_URL is set, connects
// to the remote Turso database with auth token. Otherwise falls back to a local
// file so tests and development work without remote configuration.
//
// DSN pattern from official docs:
// https://docs.turso.tech/sdk/go/quickstart (libsql-client-go section)
func OpenTurso() (*sql.DB, error) {
	url := os.Getenv("TURSO_DATABASE_URL")
	if url == "" {
		// Local file fallback — libsql supports local SQLite files.
		return sql.Open("libsql", "file:grouptrip.db")
	}
	token := os.Getenv("TURSO_AUTH_TOKEN")
	if token != "" {
		url += "?authToken=" + token
	}
	return sql.Open("libsql", url)
}

// Migrate creates the schema if it does not exist.
func (r *SQLiteRepo) Migrate() error {
	ctx := context.Background()
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS funds (
			id                  TEXT PRIMARY KEY,
			trip_id             TEXT NOT NULL,
			goal_amount         INTEGER NOT NULL,
			goal_currency       TEXT NOT NULL,
			status              TEXT NOT NULL,
			goal_adjustments    INTEGER NOT NULL DEFAULT 0,
			created_at          INTEGER NOT NULL,
			updated_at          INTEGER NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("fundrepo migrate funds: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS fund_members (
			fund_id                     TEXT NOT NULL REFERENCES funds(id),
			user_id                     TEXT NOT NULL,
			position                    INTEGER NOT NULL,
			per_person_target_amount    INTEGER NOT NULL DEFAULT 0,
			per_person_target_currency  TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (fund_id, user_id)
		);
	`)
	if err != nil {
		return fmt.Errorf("fundrepo migrate fund_members: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS fund_ledger (
			id              TEXT PRIMARY KEY,
			fund_id         TEXT NOT NULL REFERENCES funds(id),
			type            TEXT NOT NULL,
			delta_amount    INTEGER NOT NULL,
			delta_currency  TEXT NOT NULL,
			contribution_id TEXT NOT NULL,
			occurred_at     INTEGER NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("fundrepo migrate fund_ledger: %w", err)
	}
	return nil
}

// Save persists a Fund aggregate within a transaction.
// The fund row is upserted, members are replaced wholesale, and new ledger entries
// are appended (INSERT OR IGNORE on ledger id — invariant I-6: append-only).
func (r *SQLiteRepo) Save(f *fund.Fund) error {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("fundrepo save begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Upsert fund row.
	_, err = tx.ExecContext(ctx, `
		INSERT INTO funds (id, trip_id, goal_amount, goal_currency, status, goal_adjustments, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			trip_id          = excluded.trip_id,
			goal_amount      = excluded.goal_amount,
			goal_currency    = excluded.goal_currency,
			status           = excluded.status,
			goal_adjustments = excluded.goal_adjustments,
			created_at       = excluded.created_at,
			updated_at       = excluded.updated_at
	`,
		f.ID, f.TripID,
		f.Goal.Amount(), f.Goal.Currency(),
		string(f.Status),
		f.GoalAdjustments,
		f.CreatedAt.Unix(), f.UpdatedAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("fundrepo save fund: %w", err)
	}

	// Replace all members (delete + re-insert).
	_, err = tx.ExecContext(ctx, `DELETE FROM fund_members WHERE fund_id = ?`, f.ID)
	if err != nil {
		return fmt.Errorf("fundrepo save delete members: %w", err)
	}
	for i, m := range f.Members {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO fund_members (fund_id, user_id, position, per_person_target_amount, per_person_target_currency)
			VALUES (?, ?, ?, ?, ?)
		`, f.ID, m.UserID, i, m.PerPersonTarget.Amount(), m.PerPersonTarget.Currency())
		if err != nil {
			return fmt.Errorf("fundrepo save insert member %s: %w", m.UserID, err)
		}
	}

	// Append new ledger entries (INSERT OR IGNORE preserves append-only invariant I-6).
	for _, entry := range f.Ledger() {
		_, err = tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO fund_ledger (id, fund_id, type, delta_amount, delta_currency, contribution_id, occurred_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`,
			entry.ID, f.ID,
			string(entry.Type),
			entry.Delta.Amount(), entry.Delta.Currency(),
			entry.ContributionID,
			entry.OccurredAt.Unix(),
		)
		if err != nil {
			return fmt.Errorf("fundrepo save insert ledger %s: %w", entry.ID, err)
		}
	}

	return tx.Commit()
}

// Load reconstructs a Fund aggregate from persisted rows.
// Returns an error if the fund does not exist.
func (r *SQLiteRepo) Load(id string) (*fund.Fund, error) {
	ctx := context.Background()

	// Read fund row.
	var (
		dbID, tripID, goalCurrency, status string
		goalAmount, goalAdjustments        int64
		createdAtUnix, updatedAtUnix       int64
	)
	err := r.db.QueryRowContext(ctx, `
		SELECT id, trip_id, goal_amount, goal_currency, status, goal_adjustments, created_at, updated_at
		FROM funds WHERE id = ?
	`, id).Scan(&dbID, &tripID, &goalAmount, &goalCurrency, &status, &goalAdjustments, &createdAtUnix, &updatedAtUnix)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("fundrepo: fund %s not found", id)
		}
		return nil, fmt.Errorf("fundrepo load fund: %w", err)
	}

	// Read members.
	memberRows, err := r.db.QueryContext(ctx, `
		SELECT user_id, per_person_target_amount, per_person_target_currency
		FROM fund_members WHERE fund_id = ? ORDER BY position
	`, id)
	if err != nil {
		return nil, fmt.Errorf("fundrepo load members: %w", err)
	}
	defer memberRows.Close()

	var members []fund.FundMember
	for memberRows.Next() {
		var (
			userID         string
			targetAmount   int64
			targetCurrency string
		)
		if err := memberRows.Scan(&userID, &targetAmount, &targetCurrency); err != nil {
			return nil, fmt.Errorf("fundrepo load scan member: %w", err)
		}
		members = append(members, fund.FundMember{
			UserID:          userID,
			PerPersonTarget: fund.NewRawMoney(targetAmount, targetCurrency),
		})
	}
	if err := memberRows.Err(); err != nil {
		return nil, fmt.Errorf("fundrepo load member rows: %w", err)
	}

	// Read ledger entries ordered by occurred_at (chronological order).
	ledgerRows, err := r.db.QueryContext(ctx, `
		SELECT id, type, delta_amount, delta_currency, contribution_id, occurred_at
		FROM fund_ledger WHERE fund_id = ? ORDER BY occurred_at ASC, id ASC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("fundrepo load ledger: %w", err)
	}
	defer ledgerRows.Close()

	var ledger []fund.FundLedgerEntry
	for ledgerRows.Next() {
		var (
			leID, leType, leCurrency, contributionID string
			deltaAmount                              int64
			occurredAtUnix                           int64
		)
		if err := ledgerRows.Scan(&leID, &leType, &deltaAmount, &leCurrency, &contributionID, &occurredAtUnix); err != nil {
			return nil, fmt.Errorf("fundrepo load scan ledger: %w", err)
		}
		ledger = append(ledger, fund.FundLedgerEntry{
			ID:             leID,
			Type:           fund.LedgerType(leType),
			Delta:          fund.NewRawMoney(deltaAmount, leCurrency),
			ContributionID: contributionID,
			OccurredAt:     unixToTime(occurredAtUnix),
		})
	}
	if err := ledgerRows.Err(); err != nil {
		return nil, fmt.Errorf("fundrepo load ledger rows: %w", err)
	}

	goal := fund.NewRawMoney(goalAmount, goalCurrency)
	return fund.RebuildFund(
		dbID, tripID, goal,
		fund.Status(status),
		members, ledger,
		int(goalAdjustments),
		unixToTime(createdAtUnix), unixToTime(updatedAtUnix),
	)
}

func unixToTime(unix int64) time.Time {
	return time.Unix(unix, 0).UTC()
}
