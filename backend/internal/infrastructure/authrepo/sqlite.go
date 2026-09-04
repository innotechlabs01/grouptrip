// Package authrepo provides SQLite implementation for authentication persistence.
package authrepo

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SQLiteAuthRepo persists users and refresh tokens in SQLite/libsql.
type SQLiteAuthRepo struct {
	db *sql.DB
}

// NewSQLiteAuthRepo creates a new SQLite authentication repository.
func NewSQLiteAuthRepo(db *sql.DB) *SQLiteAuthRepo {
	return &SQLiteAuthRepo{db: db}
}

// Migrate creates the users and refresh_tokens tables.
func (r *SQLiteAuthRepo) Migrate() error {
	ctx := context.Background()
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			name TEXT,
			password_hash TEXT NOT NULL,
			created_at INTEGER NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("authrepo migrate users: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS refresh_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id),
			token_hash TEXT UNIQUE NOT NULL,
			expires_at INTEGER NOT NULL,
			created_at INTEGER NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("authrepo migrate refresh_tokens: %w", err)
	}
	return nil
}

// SaveUser persists a user, upserting on conflict.
func (r *SQLiteAuthRepo) SaveUser(user *User) error {
	ctx := context.Background()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, password_hash, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			email = excluded.email,
			name = excluded.name,
			password_hash = excluded.password_hash,
			created_at = excluded.created_at
	`, user.ID, user.Email, user.Name, user.PasswordHash, user.CreatedAt.Unix())
	if err != nil {
		return fmt.Errorf("authrepo save user: %w", err)
	}
	return nil
}

// FindUserByEmail retrieves a user by email.
func (r *SQLiteAuthRepo) FindUserByEmail(email string) (*User, error) {
	ctx := context.Background()
	var u User
	var createdUnix int64
	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, created_at
		FROM users WHERE email = ?
	`, email).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &createdUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("authrepo: user not found")
		}
		return nil, fmt.Errorf("authrepo find user by email: %w", err)
	}
	u.CreatedAt = time.Unix(createdUnix, 0).UTC()
	return &u, nil
}

// SaveRefreshToken persists a refresh token.
func (r *SQLiteAuthRepo) SaveRefreshToken(token *RefreshToken) error {
	ctx := context.Background()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			user_id = excluded.user_id,
			token_hash = excluded.token_hash,
			expires_at = excluded.expires_at,
			created_at = excluded.created_at
	`, token.ID, token.UserID, token.TokenHash, token.ExpiresAt.Unix(), token.CreatedAt.Unix())
	if err != nil {
		return fmt.Errorf("authrepo save refresh token: %w", err)
	}
	return nil
}

// FindRefreshToken retrieves a refresh token by token hash.
func (r *SQLiteAuthRepo) FindRefreshToken(tokenHash string) (*RefreshToken, error) {
	ctx := context.Background()
	var t RefreshToken
	var expiresUnix, createdUnix int64
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at
		FROM refresh_tokens WHERE token_hash = ?
	`, tokenHash).Scan(&t.ID, &t.UserID, &t.TokenHash, &expiresUnix, &createdUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("authrepo: refresh token not found")
		}
		return nil, fmt.Errorf("authrepo find refresh token: %w", err)
	}
	t.ExpiresAt = time.Unix(expiresUnix, 0).UTC()
	t.CreatedAt = time.Unix(createdUnix, 0).UTC()
	return &t, nil
}

// DeleteRefreshToken removes a refresh token by id.
func (r *SQLiteAuthRepo) DeleteRefreshToken(id string) error {
	ctx := context.Background()
	_, err := r.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("authrepo delete refresh token: %w", err)
	}
	return nil
}

// DeleteRefreshTokensForUser removes all refresh tokens for a user.
func (r *SQLiteAuthRepo) DeleteRefreshTokensForUser(userID string) error {
	ctx := context.Background()
	_, err := r.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("authrepo delete refresh tokens for user: %w", err)
	}
	return nil
}

// ListRefreshTokens returns all refresh tokens for a user.
func (r *SQLiteAuthRepo) ListRefreshTokens(userID string) ([]*RefreshToken, error) {
	ctx := context.Background()
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at
		FROM refresh_tokens WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("authrepo list refresh tokens: %w", err)
	}
	defer rows.Close()
	var tokens []*RefreshToken
	for rows.Next() {
		var t RefreshToken
		var createdUnix, expiresUnix int64
		if err := rows.Scan(&t.ID, &t.UserID, &t.TokenHash, &expiresUnix, &createdUnix); err != nil {
			return nil, fmt.Errorf("authrepo list scan: %w", err)
		}
		t.ExpiresAt = time.Unix(expiresUnix, 0).UTC()
		t.CreatedAt = time.Unix(createdUnix, 0).UTC()
		tokens = append(tokens, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("authrepo list rows: %w", err)
	}
	return tokens, nil
}
