# Auth Backend (Fase 0) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a secure authentication domain to the grouptrip Go backend: register, login, logout, `/auth/me`, and rotating refresh tokens, so the PWA can authenticate for real from day one.

**Architecture:** New `auth` domain following the existing Clean/DDD pattern (domain entity + repositories + HTTP handlers wired through the existing `Server`). Passwords hashed with argon2id; access is a short-lived JWT; refresh tokens are long-lived, hashed in the DB, and rotated on every use with replay detection.

**Tech Stack:** Go 1.26, Turso (libSQL/SQLite), `github.com/alexedwards/argon2id`, `github.com/golang-jwt/jwt/v5`, `github.com/google/uuid`, stdlib `crypto/rand` + `crypto/sha256`.

**Spec:** `docs/superpowers/specs/2026-09-04-pwa-and-backend-completion-design.md` (§4.1 Auth domain, §2 Decisions)

## Global Constraints

- Backend module: `github.com/frg/grouptrip`, run all `go get` / `go build` / `go test` / `go vet` from `backend/`.
- Follow the existing backend patterns exactly: repos use `db *sql.DB` + `Migrate()`; handlers are `Server` methods using `writeJSON`/`writeJSONError`; constructor-based wiring through `NewServer*`.
- Passwords: **argon2id** only — never store plaintext or reversible hashes.
- Refresh tokens: hashed (SHA-256) in the DB, never stored plaintext.
- Access token: JWT HS256, `exp` 15 minutes. Use env `AUTH_JWT_SECRET` (fallback only for dev).
- Cookie: `httpOnly`, `Secure`, `SameSite=Lax`, Path `/auth`.
- All new code TDD: write the failing test first, run it, implement, run green, commit.

---

### Task 1: Add dependencies and auth user domain

**Files:**
- Modify: `backend/go.mod` (via `go get`)
- Create: `backend/internal/domain/auth/user.go`
- Test: `backend/internal/domain/auth/user_test.go`

**Interfaces:**
- Produces: `type User struct { ID, Email, Name, PasswordHash string; CreatedAt time.Time }`, `func NewUser(id, email, name, passwordHash string) (*User, error)`, `func (u *User) Validate() error`

- [ ] **Step 1: Add dependencies**

Run from `backend/`:
```bash
go get github.com/alexedwards/argon2id@latest
go get github.com/golang-jwt/jwt/v5@latest
go get github.com/google/uuid@latest
```
Expected: `go.mod` updated, no errors.

- [ ] **Step 2: Write the failing test**

Create `backend/internal/domain/auth/user_test.go`:
```go
package auth

import "testing"

func TestNewUser_Valid(t *testing.T) {
	u, err := NewUser("u_1", "a@b.com", "Ana", "hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != "u_1" || u.Email != "a@b.com" || u.Name != "Ana" {
		t.Fatalf("fields not set: %+v", u)
	}
}

func TestNewUser_InvalidEmail(t *testing.T) {
	if _, err := NewUser("u_1", "not-an-email", "Ana", "hash"); err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestNewUser_EmptyNameOk(t *testing.T) {
	if _, err := NewUser("u_1", "a@b.com", "", "hash"); err != nil {
		t.Fatalf("empty name should be allowed: %v", err)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run from `backend/`: `go test ./internal/domain/auth/ -run TestNewUser -v`
Expected: FAIL — package `auth` not yet created / `NewUser` undefined.

- [ ] **Step 4: Write minimal implementation**

Create `backend/internal/domain/auth/user.go`:
```go
// Package auth provides the authentication and user domain.
package auth

import (
	"errors"
	"strings"
	"time"
)

// User is an authenticated account.
type User struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	CreatedAt    time.Time
}

// NewUser validates and constructs a User. Email must be non-empty and
// contain a '@'. Name may be empty.
func NewUser(id, email, name, passwordHash string) (*User, error) {
	u := &User{ID: id, Email: strings.ToLower(strings.TrimSpace(email)), Name: name, PasswordHash: passwordHash, CreatedAt: time.Now().UTC()}
	if err := u.Validate(); err != nil {
		return nil, err
	}
	return u, nil
}

// Validate enforces user invariants.
func (u *User) Validate() error {
	if u.ID == "" {
		return errors.New("auth: user id required")
	}
	if u.Email == "" || !strings.Contains(u.Email, "@") {
		return errors.New("auth: valid email required")
	}
	if u.PasswordHash == "" {
		return errors.New("auth: password hash required")
	}
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run from `backend/`: `go test ./internal/domain/auth/ -v`
Expected: PASS (3 tests).

- [ ] **Step 6: Commit**

```bash
git add backend/go.mod backend/go.sum backend/internal/domain/auth/
git commit -m "feat: auth user domain entity"
```

---

### Task 2: Auth repository (users) + refresh token store

**Files:**
- Create: `backend/internal/infrastructure/authrepo/authrepo.go`
- Test: `backend/internal/infrastructure/authrepo/authrepo_test.go`

**Interfaces:**
- Consumes: `auth.User` from Task 1.
- Produces:
  - `func NewSQLiteAuthRepo(db *sql.DB) *SQLiteAuthRepo`
  - `func (r *SQLiteAuthRepo) Migrate() error`
  - `func (r *SQLiteAuthRepo) CreateUser(ctx context.Context, u *auth.User) error` (error if email exists)
  - `func (r *SQLiteAuthRepo) FindByEmail(ctx context.Context, email string) (*auth.User, error)`
  - `func (r *SQLiteAuthRepo) FindByID(ctx context.Context, id string) (*auth.User, error)`
  - `func (r *SQLiteAuthRepo) StoreRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error`
  - `func (r *SQLiteAuthRepo) FindRefreshToken(ctx context.Context, tokenHash string) (userID string, expiresAt time.Time, err error)`
  - `func (r *SQLiteAuthRepo) RevokeUserTokens(ctx context.Context, userID string) error`
  - `func (r *SQLiteAuthRepo) IsTokenUsed(ctx context.Context, tokenHash string) (bool, error)`
  - `func (r *SQLiteAuthRepo) MarkTokenUsed(ctx context.Context, tokenHash string) error`
  - `func (r *SQLiteAuthRepo) ErrEmailTaken` (sentinel), `ErrNotFound` (sentinel)

- [ ] **Step 1: Write the failing test**

Create `backend/internal/infrastructure/authrepo/authrepo_test.go`:
```go
package authrepo

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/frg/grouptrip/internal/domain/auth"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *SQLiteAuthRepo {
	t.Helper()
	dir, err := os.MkdirTemp("", "authrepo")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	db, err := openTest(dir)
	if err != nil {
		t.Fatal(err)
	}
	r := NewSQLiteAuthRepo(db)
	if err := r.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return r
}

func TestCreateAndFindByEmail(t *testing.T) {
	r := newTestDB(t)
	u, _ := auth.NewUser("u1", "a@b.com", "Ana", "ph")
	if err := r.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := r.FindByEmail(context.Background(), "a@b.com")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.ID != "u1" {
		t.Fatalf("expected u1, got %s", got.ID)
	}
}

func TestCreateDuplicateEmail(t *testing.T) {
	r := newTestDB(t)
	u, _ := auth.NewUser("u1", "a@b.com", "Ana", "ph")
	if err := r.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	u2, _ := auth.NewUser("u2", "a@b.com", "Bo", "ph2")
	if err := r.CreateUser(context.Background(), u2); err != ErrEmailTaken {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestRefreshTokenLifecycle(t *testing.T) {
	r := newTestDB(t)
	u, _ := auth.NewUser("u1", "a@b.com", "Ana", "ph")
	_ = r.CreateUser(context.Background(), u)

	hash := "sha256hashvalue000"
	if err := r.StoreRefreshToken(context.Background(), "u1", hash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("store: %v", err)
	}
	uid, _, err := r.FindRefreshToken(context.Background(), hash)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if uid != "u1" {
		t.Fatalf("expected u1, got %s", uid)
	}

	if used, _ := r.IsTokenUsed(context.Background(), hash); used {
		t.Fatal("token should not be used yet")
	}
	if err := r.MarkTokenUsed(context.Background(), hash); err != nil {
		t.Fatal(err)
	}
	if used, _ := r.IsTokenUsed(context.Background(), hash); !used {
		t.Fatal("token should be used after mark")
	}

	if err := r.RevokeUserTokens(context.Background(), "u1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.FindRefreshToken(context.Background(), hash); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after revoke, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run from `backend/`: `go test ./internal/infrastructure/authrepo/ -v`
Expected: FAIL (package / symbols undefined).

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/infrastructure/authrepo/authrepo.go`:
```go
// Package authrepo provides persistence for the auth domain (users + refresh tokens).
package authrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/frg/grouptrip/internal/domain/auth"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

// ErrEmailTaken is returned when creating a user with an existing email.
var ErrEmailTaken = errors.New("authrepo: email already taken")

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("authrepo: not found")

// SQLiteAuthRepo persists users and refresh tokens.
type SQLiteAuthRepo struct {
	db *sql.DB
}

// NewSQLiteAuthRepo creates an auth repo over the given DB connection.
func NewSQLiteAuthRepo(db *sql.DB) *SQLiteAuthRepo {
	return &SQLiteAuthRepo{db: db}
}

// Migrate creates the users and refresh_tokens tables.
func (r *SQLiteAuthRepo) Migrate() error {
	ctx := context.Background()
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id            TEXT PRIMARY KEY,
			email         TEXT NOT NULL UNIQUE,
			name          TEXT NOT NULL DEFAULT '',
			password_hash TEXT NOT NULL,
			created_at    INTEGER NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("authrepo migrate users: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS refresh_tokens (
			id         TEXT PRIMARY KEY,
			user_id    TEXT NOT NULL REFERENCES users(id),
			token_hash TEXT NOT NULL,
			expires_at INTEGER NOT NULL,
			used       INTEGER NOT NULL DEFAULT 0,
			revoked_at INTEGER,
			created_at INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash);
	`)
	if err != nil {
		return fmt.Errorf("authrepo migrate refresh_tokens: %w", err)
	}
	return nil
}

func (r *SQLiteAuthRepo) CreateUser(ctx context.Context, u *auth.User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, password_hash, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, u.ID, u.Email, u.Name, u.PasswordHash, u.CreatedAt.Unix())
	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailTaken
		}
		return fmt.Errorf("authrepo create user: %w", err)
	}
	return nil
}

func (r *SQLiteAuthRepo) FindByEmail(ctx context.Context, email string) (*auth.User, error) {
	return r.find(ctx, `SELECT id, email, name, password_hash, created_at FROM users WHERE email = ?`, email)
}

func (r *SQLiteAuthRepo) FindByID(ctx context.Context, id string) (*auth.User, error) {
	return r.find(ctx, `SELECT id, email, name, password_hash, created_at FROM users WHERE id = ?`, id)
}

func (r *SQLiteAuthRepo) find(ctx context.Context, q, arg string) (*auth.User, error) {
	var (
		id, email, name, hash string
		createdAt             int64
	)
	err := r.db.QueryRowContext(ctx, q, arg).Scan(&id, &email, &name, &hash, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("authrepo find: %w", err)
	}
	return &auth.User{ID: id, Email: email, Name: name, PasswordHash: hash, CreatedAt: time.Unix(createdAt, 0).UTC()}, nil
}

func (r *SQLiteAuthRepo) StoreRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	id := newID()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, used, created_at)
		VALUES (?, ?, ?, ?, 0, ?)
	`, id, userID, tokenHash, expiresAt.Unix(), time.Now().Unix())
	if err != nil {
		return fmt.Errorf("authrepo store refresh: %w", err)
	}
	return nil
}

func (r *SQLiteAuthRepo) FindRefreshToken(ctx context.Context, tokenHash string) (string, time.Time, error) {
	var userID string
	var expiresAt, revokedAt sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT user_id, expires_at, revoked_at FROM refresh_tokens WHERE token_hash = ?
	`, tokenHash).Scan(&userID, &expiresAt, &revokedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", time.Time{}, ErrNotFound
		}
		return "", time.Time{}, fmt.Errorf("authrepo find refresh: %w", err)
	}
	if revokedAt.Valid {
		return "", time.Time{}, ErrNotFound
	}
	return userID, time.Unix(expiresAt.Int64, 0).UTC(), nil
}

func (r *SQLiteAuthRepo) IsTokenUsed(ctx context.Context, tokenHash string) (bool, error) {
	var used int
	err := r.db.QueryRowContext(ctx, `SELECT used FROM refresh_tokens WHERE token_hash = ?`, tokenHash).Scan(&used)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("authrepo is used: %w", err)
	}
	return used == 1, nil
}

func (r *SQLiteAuthRepo) MarkTokenUsed(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE refresh_tokens SET used = 1 WHERE token_hash = ?`, tokenHash)
	if err != nil {
		return fmt.Errorf("authrepo mark used: %w", err)
	}
	return nil
}

func (r *SQLiteAuthRepo) RevokeUserTokens(ctx context.Context, userID string) error {
	now := time.Now().Unix()
	_, err := r.db.ExecContext(ctx, `UPDATE refresh_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`, now, userID)
	if err != nil {
		return fmt.Errorf("authrepo revoke: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	msg := err.Error()
	return containsStr(msg, "UNIQUE") || containsStr(msg, "unique constraint")
}

func containsStr(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexStr(haystack, needle) >= 0
}

func indexStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 4: Add test-only DB open helper**

Create `backend/internal/infrastructure/authrepo/testdb.go`:
```go
package authrepo

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
)

func openTest(dir string) (*sql.DB, error) {
	dsn := "file:" + filepath.Join(dir, "test.db")
	db, err := sql.Open("libsql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open test db: %w", err)
	}
	return db, nil
}

func newID() string {
	return uuid.NewString()
}
```
Note: `newID` defined here is used by `StoreRefreshToken`; it lives in `testdb.go` for now and will move to a shared helper in Task 3.

- [ ] **Step 5: Run tests to verify they pass**

Run from `backend/`: `go test ./internal/infrastructure/authrepo/ -v`
Expected: PASS (3 tests).

- [ ] **Step 6: Commit**

```bash
git add backend/internal/infrastructure/authrepo/
git commit -m "feat: auth persistence (users + refresh tokens)"
```

---

### Task 3: Password hashing + secure token utilities

**Files:**
- Create: `backend/internal/domain/auth/security.go`
- Test: `backend/internal/domain/auth/security_test.go`

**Interfaces:**
- Consumes: nothing (standalone).
- Produces:
  - `func HashPassword(plain string) (string, error)`
  - `func VerifyPassword(plain, hash string) (bool, error)`
  - `func GenerateRefreshToken() (token string, hash string, err error)` — token is a 32-byte base64url random string; hash is its SHA-256 hex.
  - `func HashToken(token string) string`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/domain/auth/security_test.go`:
```go
package auth

import "testing"

func TestHashVerifyPassword(t *testing.T) {
	h, err := HashPassword("s3cret!")
	if err != nil {
		t.Fatal(err)
	}
	if h == "s3cret!" {
		t.Fatal("hash must not equal plaintext")
	}
	ok, err := VerifyPassword("s3cret!", h)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected password to verify")
	}
	ok, _ = VerifyPassword("wrong", h)
	if ok {
		t.Fatal("wrong password must not verify")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	tok, hash, err := GenerateRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if tok == "" || hash == "" {
		t.Fatal("token and hash must be non-empty")
	}
	if HashToken(tok) != hash {
		t.Fatal("hash must equal HashToken(token)")
	}
	tok2, _, _ := GenerateRefreshToken()
	if tok == tok2 {
		t.Fatal("tokens must be unique")
	}
}

func TestHashTokenDeterministic(t *testing.T) {
	if HashToken("abc") != HashToken("abc") {
		t.Fatal("hash must be deterministic")
	}
	if HashToken("abc") == HashToken("abd") {
		t.Fatal("different tokens must hash differently")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run from `backend/`: `go test ./internal/domain/auth/ -run TestHash -v`
Expected: FAIL (functions undefined).

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/domain/auth/security.go`:
```go
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"github.com/alexedwards/argon2id"
)

// HashPassword hashes a plaintext password with argon2id.
func HashPassword(plain string) (string, error) {
	return argon2id.CreateHash(plain, argon2id.DefaultParams)
}

// VerifyPassword checks a plaintext password against an argon2id hash.
func VerifyPassword(plain, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(plain, hash)
}

// GenerateRefreshToken returns a fresh 32-byte random token and its SHA-256 hash.
func GenerateRefreshToken() (token string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	return token, HashToken(token), nil
}

// HashToken returns the SHA-256 hex of a token (used to store refresh tokens safely).
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run from `backend/`: `go test ./internal/domain/auth/ -v`
Expected: PASS (6 tests total — 3 from Task 1, 3 from this task).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/domain/auth/security.go backend/internal/domain/auth/security_test.go
git commit -m "feat: argon2id password hashing and token utilities"
```

---

### Task 4: Session service (register/login/logout/refresh/me logic)

**Files:**
- Create: `backend/internal/application/authservice/authservice.go`
- Test: `backend/internal/application/authservice/authservice_test.go`

**Interfaces:**
- Consumes: `authrepo.SQLiteAuthRepo` (Task 2), `auth.HashPassword/VerifyPassword/GenerateRefreshToken/HashToken` (Task 3).
- Produces:
  - `type SessionService struct { Users *authrepo.SQLiteAuthRepo; JWTSecret []byte }`
  - `func NewSessionService(users *authrepo.SQLiteAuthRepo, jwtSecret []byte) *SessionService`
  - `Register(ctx, email, password, name string) (accessToken string, refreshToken string, err error)`
  - `Login(ctx, email, password string) (accessToken string, refreshToken string, err error)`
  - `Refresh(ctx, providedRefresh string) (accessToken string, newRefresh string, err error)`
  - `Logout(ctx, userID string) error`
  - `Me(ctx, userID string) (*UserView, error)` where `type UserView struct { ID, Email, Name string }`
  - `type TokenClaims struct { UserID string; jwt.RegisteredClaims }`
  - `SignAccessToken(userID string) (string, error)`
  - `ParseAccessToken(token string) (string, error)` returns userID
  - Sentinels: `ErrInvalidCredentials`, `ErrEmailTaken`, `ErrInvalidRefresh`, `ErrRefreshReuse` (replay detected)

- [ ] **Step 1: Write the failing test**

Create `backend/internal/application/authservice/authservice_test.go`:
```go
package authservice

import (
	"context"
	"os"
	"testing"

	"github.com/frg/grouptrip/internal/infrastructure/authrepo"
)

func newService(t *testing.T) *SessionService {
	t.Helper()
	dir, _ := os.MkdirTemp("", "authsvc")
	t.Cleanup(func() { os.RemoveAll(dir) })
	db, _ := openTestDB(dir)
	r := authrepo.NewSQLiteAuthRepo(db)
	if err := r.Migrate(); err != nil {
		t.Fatal(err)
	}
	return NewSessionService(r, []byte("test-secret-32-bytes-long-secret!"))
}

func TestRegisterAndLogin(t *testing.T) {
	svc := newService(t)
	_, _, err := svc.Register(context.Background(), "a@b.com", "password123", "Ana")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	_, _, err = svc.Login(context.Background(), "a@b.com", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	svc := newService(t)
	_, _, _ = svc.Register(context.Background(), "a@b.com", "password123", "Ana")
	if _, _, err := svc.Register(context.Background(), "a@b.com", "password123", "Bo"); err != ErrEmailTaken {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc := newService(t)
	_, _, _ = svc.Register(context.Background(), "a@b.com", "password123", "Ana")
	if _, _, err := svc.Login(context.Background(), "a@b.com", "wrong"); err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestRefreshRotationAndReplayDetection(t *testing.T) {
	svc := newService(t)
	_, rt, _ := svc.Register(context.Background(), "a@b.com", "password123", "Ana")

	// First refresh rotates: old token becomes unusable.
	_, newRT, err := svc.Refresh(context.Background(), rt)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if newRT == rt {
		t.Fatal("refresh must rotate")
	}

	// Replaying the OLD token now signals reuse (replay detection).
	if _, _, err := svc.Refresh(context.Background(), rt); err != ErrRefreshReuse {
		t.Fatalf("expected ErrRefreshReuse on replay, got %v", err)
	}
}

func TestRefreshWithBogusToken(t *testing.T) {
	svc := newService(t)
	if _, _, err := svc.Refresh(context.Background(), "not-a-real-token"); err != ErrInvalidRefresh {
		t.Fatalf("expected ErrInvalidRefresh, got %v", err)
	}
}

func TestAccessTokenRoundTrip(t *testing.T) {
	svc := newService(t)
	tok, err := svc.SignAccessToken("u_1")
	if err != nil {
		t.Fatal(err)
	}
	uid, err := svc.ParseAccessToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if uid != "u_1" {
		t.Fatalf("expected u_1, got %s", uid)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run from `backend/`: `go test ./internal/application/authservice/ -v`
Expected: FAIL (package / symbols undefined).

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/application/authservice/authservice.go`:
```go
// Package authservice implements authentication use cases.
package authservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/frg/grouptrip/internal/domain/auth"
	"github.com/frg/grouptrip/internal/infrastructure/authrepo"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Sentinels.
var (
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrEmailTaken         = authrepo.ErrEmailTaken
	ErrInvalidRefresh     = errors.New("auth: invalid refresh token")
	ErrRefreshReuse       = errors.New("auth: refresh token reuse detected")
	ErrUserNotFound       = authrepo.ErrNotFound
)

const accessTokenTTL = 15 * time.Minute
const refreshTokenTTL = 30 * 24 * time.Hour // 30 days

// TokenClaims are the JWT access-token claims.
type TokenClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

// UserView is the user as exposed over HTTP.
type UserView struct {
	ID, Email, Name string
}

// SessionService orchestrates authentication.
type SessionService struct {
	Users     *authrepo.SQLiteAuthRepo
	JWTSecret []byte
}

// NewSessionService builds a SessionService.
func NewSessionService(users *authrepo.SQLiteAuthRepo, jwtSecret []byte) *SessionService {
	return &SessionService{Users: users, JWTSecret: jwtSecret}
}

// Register creates a user and returns access + refresh tokens.
func (s *SessionService) Register(ctx context.Context, email, password, name string) (string, string, error) {
	if password == "" {
		return "", "", fmt.Errorf("auth: password required")
	}
	if len(password) < 8 {
		return "", "", fmt.Errorf("auth: password must be at least 8 characters")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return "", "", err
	}
	u, err := auth.NewUser(uuid.NewString(), email, name, hash)
	if err != nil {
		return "", "", err
	}
	if err := s.Users.CreateUser(ctx, u); err != nil {
		return "", "", err
	}
	return s.issueTokens(ctx, u.ID)
}

// Login validates credentials and issues tokens.
func (s *SessionService) Login(ctx context.Context, email, password string) (string, string, error) {
	u, err := s.Users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", err
	}
	ok, err := auth.VerifyPassword(password, u.PasswordHash)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", ErrInvalidCredentials
	}
	return s.issueTokens(ctx, u.ID)
}

// Refresh rotates a refresh token and returns a new access + refresh pair.
func (s *SessionService) Refresh(ctx context.Context, provided string) (string, string, error) {
	hash := auth.HashToken(provided)
	userID, expiresAt, err := s.Users.FindRefreshToken(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", "", ErrInvalidRefresh
		}
		return "", "", err
	}
	if time.Now().After(expiresAt) {
		return "", "", ErrInvalidRefresh
	}
	used, err := s.Users.IsTokenUsed(ctx, hash)
	if err != nil {
		return "", "", err
	}
	if used {
		// Replay: revoke the whole session chain and signal reuse.
		_ = s.Users.RevokeUserTokens(ctx, userID)
		return "", "", ErrRefreshReuse
	}
	if err := s.Users.MarkTokenUsed(ctx, hash); err != nil {
		return "", "", err
	}
	return s.issueTokens(ctx, userID)
}

// Logout revokes all refresh tokens for the user.
func (s *SessionService) Logout(ctx context.Context, userID string) error {
	return s.Users.RevokeUserTokens(ctx, userID)
}

// Me returns the user view for a user id.
func (s *SessionService) Me(ctx context.Context, userID string) (*UserView, error) {
	u, err := s.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &UserView{ID: u.ID, Email: u.Email, Name: u.Name}, nil
}

// SignAccessToken signs a JWT for a user.
func (s *SessionService) SignAccessToken(userID string) (string, error) {
	claims := TokenClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "grouptrip",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.JWTSecret)
}

// ParseAccessToken validates a JWT and returns the user id.
func (s *SessionService) ParseAccessToken(tokenStr string) (string, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &TokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method %v", t.Header["alg"])
		}
		return s.JWTSecret, nil
	})
	if err != nil {
		return "", fmt.Errorf("auth: invalid access token: %w", err)
	}
	claims, ok := tok.Claims.(*TokenClaims)
	if !ok || !tok.Valid {
		return "", fmt.Errorf("auth: invalid access token")
	}
	return claims.UserID, nil
}

func (s *SessionService) issueTokens(ctx context.Context, userID string) (string, string, error) {
	access, err := s.SignAccessToken(userID)
	if err != nil {
		return "", "", err
	}
	rt, rtHash, err := auth.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}
	if err := s.Users.StoreRefreshToken(ctx, userID, rtHash, time.Now().Add(refreshTokenTTL)); err != nil {
		return "", "", err
	}
	return access, rt, nil
}
```

- [ ] **Step 4: Add test helper to open a DB**

Create `backend/internal/application/authservice/testdb.go`:
```go
package authservice

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

func openTestDB(dir string) (*sql.DB, error) {
	db, err := sql.Open("libsql", "file:"+filepath.Join(dir, "test.db"))
	if err != nil {
		return nil, fmt.Errorf("open test db: %w", err)
	}
	return db, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run from `backend/`: `go test ./internal/application/authservice/ -v`
Expected: PASS (6 tests).

- [ ] **Step 6: Commit**

```bash
git add backend/internal/application/authservice/
git commit -m "feat: session service (register/login/refresh rotation/logout/me)"
```

---

### Task 5: Auth HTTP handlers + routes + wiring

**Files:**
- Modify: `backend/internal/interfaces/http/server.go`
- Create: `backend/internal/interfaces/http/handler_auth.go`
- Create: `backend/internal/interfaces/http/auth_test.go`

**Interfaces:**
- Consumes: `authservice.SessionService`, `authservice.UserView`, `authservice.TokenClaims`.
- Produces:
  - `func (s *Server) register`, `login`, `logout`, `me`, `refresh`
  - Cookie name: `refresh_token`
  - Routes: `POST /auth/register`, `POST /auth/login`, `POST /auth/logout`, `GET /auth/me`, `POST /auth/refresh`.

- [ ] **Step 1: Write the failing test**

Create `backend/internal/interfaces/http/auth_test.go`:
```go
package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/frg/grouptrip/internal/application/authservice"
	"github.com/frg/grouptrip/internal/infrastructure/authrepo"
)

func newAuthServer(t *testing.T) *Server {
	t.Helper()
	dir, _ := os.MkdirTemp("", "authsrv")
	t.Cleanup(func() { os.RemoveAll(dir) })
	db, _ := openServerTestDB(dir)
	ar := authrepo.NewSQLiteAuthRepo(db)
	if err := ar.Migrate(); err != nil {
		t.Fatal(err)
	}
	svc := authservice.NewSessionService(ar, []byte("test-secret-32-bytes-long-secret!"))
	return NewServerWithAuth(nil, nil, "", nil, nil, svc)
}

func doJSON(t *testing.T, srv *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	return w
}

func TestRegisterEndpoint(t *testing.T) {
	srv := newAuthServer(t)
	w := doJSON(t, srv, http.MethodPost, "/auth/register", `{"email":"a@b.com","password":"password123","name":"Ana"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !hasCookie(w, "refresh_token") {
		t.Fatal("expected refresh_token cookie")
	}
	var resp map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["access_token"] == "" || resp["access_token"] == nil {
		t.Fatal("expected access_token in body")
	}
}

func TestLoginFlow(t *testing.T) {
	srv := newAuthServer(t)
	doJSON(t, srv, http.MethodPost, "/auth/register", `{"email":"a@b.com","password":"password123","name":"Ana"}`)
	w := doJSON(t, srv, http.MethodPost, "/auth/login", `{"email":"a@b.com","password":"password123"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLoginWrongPassword(t *testing.T) {
	srv := newAuthServer(t)
	doJSON(t, srv, http.MethodPost, "/auth/register", `{"email":"a@b.com","password":"password123","name":"Ana"}`)
	w := doJSON(t, srv, http.MethodPost, "/auth/login", `{"email":"a@b.com","password":"nope"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMeRequiresToken(t *testing.T) {
	srv := newAuthServer(t)
	w := doJSON(t, srv, http.MethodGet, "/auth/me", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d: %s", w.Code, w.Body.String())
	}
}

func hasCookie(resp *httptest.ResponseRecorder, name string) bool {
	cookies := resp.Result().Cookies()
	for _, c := range cookies {
		if c.Name == name {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run test to verify it fails**

Run from `backend/`: `go test ./internal/interfaces/http/ -run TestRegisterEndpoint -v`
Expected: FAIL (route not registered / symbols undefined).

- [ ] **Step 3: Write header parsing helper + handler implementation**

Create `backend/internal/interfaces/http/handler_auth.go`:
```go
package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/frg/grouptrip/internal/application/authservice"
)

const refreshCookieName = "refresh_token"

// register handles POST /auth/register.
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	access, refresh, err := s.auth.Register(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		writeJSONError(w, statusForAuthErr(err), err.Error())
		return
	}
	setRefreshCookie(w, refresh)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"access_token": access})
}

// login handles POST /auth/login.
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	access, refresh, err := s.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeJSONError(w, statusForAuthErr(err), err.Error())
		return
	}
	setRefreshCookie(w, refresh)
	writeJSON(w, http.StatusOK, map[string]interface{}{"access_token": access})
}

// logout handles POST /auth/logout.
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.authFromReq(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := s.auth.Logout(r.Context(), sess.UserID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to logout")
		return
	}
	clearRefreshCookie(w)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// me handles GET /auth/me.
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.authFromReq(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	u, err := s.auth.Me(r.Context(), sess.UserID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"id": u.ID, "email": u.Email, "name": u.Name})
}

// refresh handles POST /auth/refresh using the cookie refresh token.
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(refreshCookieName)
	if err != nil || c.Value == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}
	access, newRefresh, err := s.auth.Refresh(r.Context(), c.Value)
	if err != nil {
		clearRefreshCookie(w)
		writeJSONError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	setRefreshCookie(w, newRefresh)
	writeJSON(w, http.StatusOK, map[string]interface{}{"access_token": access})
}

// authFromReq extracts the user id from the Authorization Bearer token.
type sessionInfo struct{ UserID string }

func (s *Server) authFromReq(r *http.Request) (sessionInfo, bool) {
	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(authz, "Bearer ") {
		return sessionInfo{}, false
	}
	tok := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
	if tok == "" {
		return sessionInfo{}, false
	}
	uid, err := s.auth.ParseAccessToken(tok)
	if err != nil {
		return sessionInfo{}, false
	}
	return sessionInfo{UserID: uid}, true
}

func setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})
}

func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func statusForAuthErr(err error) int {
	switch err.Error() {
	case authservice.ErrEmailTaken.Error(),
		"auth: password required",
		"auth: password must be at least 8 characters",
		"auth: valid email required":
		return http.StatusBadRequest
	case authservice.ErrInvalidCredentials.Error(),
		authservice.ErrInvalidRefresh.Error(),
		authservice.ErrRefreshReuse.Error():
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
```

- [ ] **Step 4: Modify server.go to add the auth service and routes**

Edit `backend/internal/interfaces/http/server.go`:
- Add import: `"github.com/frg/grouptrip/internal/application/authservice"`.
- Add field `auth *authservice.SessionService` to `Server`.
- Add new constructor `NewServerWithAuth` (keep existing constructors delegating to it):

```go
// NewServerWithAuth wires every capability including auth. Passing nil for cmd,
// progress, or auth disables the corresponding routes.
func NewServerWithAuth(
	repo *fundrepo.SQLiteRepo,
	contribs *contribrepo.SQLiteContribRepo,
	webhookSecret string,
	cmd *commands.ContributeCommand,
	progress *queries.GetFundProgress,
	auth *authservice.SessionService,
) *Server {
	s := &Server{
		repo:          repo,
		contribs:      contribs,
		webhookSecret: webhookSecret,
		contribute:    cmd,
		progress:      progress,
		auth:          auth,
		mux:           http.NewServeMux(),
	}
	s.routes()
	return s
}
```

- Change `NewServer`, `NewServerWithWebhook`, `NewServerWithPayments` to delegate to `NewServerWithAuth` passing `nil` for auth:
  - `NewServer` → `return NewServerWithAuth(repo, nil, "", nil, nil, nil)`
  - `NewServerWithWebhook` → `return NewServerWithAuth(repo, contribs, webhookSecret, nil, nil, nil)`
  - `NewServerWithPayments(...)` → `return NewServerWithAuth(repo, contribs, webhookSecret, cmd, progress, nil)`

- Add to `routes()` (auth routes always registered when auth wired):
```go
if s.auth != nil {
	s.mux.HandleFunc("POST /auth/register", s.register)
	s.mux.HandleFunc("POST /auth/login", s.login)
	s.mux.HandleFunc("POST /auth/logout", s.logout)
	s.mux.HandleFunc("GET /auth/me", s.me)
	s.mux.HandleFunc("POST /auth/refresh", s.refresh)
}
```

- [ ] **Step 5: Add test DB helper for the http package**

Create `backend/internal/interfaces/http/testdb_helper_test.go`:
```go
package http

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

func openServerTestDB(dir string) (*sql.DB, error) {
	db, err := sql.Open("libsql", "file:"+filepath.Join(dir, "test.db"))
	if err != nil {
		return nil, fmt.Errorf("open test db: %w", err)
	}
	return db, nil
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run from `backend/`: `go test ./internal/interfaces/http/ -v`
Expected: PASS (all existing + new auth tests).

- [ ] **Step 7: Commit**

```bash
git add backend/internal/interfaces/http/
git commit -m "feat: auth HTTP endpoints (register/login/logout/me/refresh)"
```

---

### Task 6: Wire auth into main.go + rate limiting

**Files:**
- Modify: `backend/cmd/api/main.go`
- Create: `backend/cmd/api/ratelimit.go`
- Test: `backend/internal/interfaces/http/handler_auth.go` (review rate-limit integration in Task 7)

**Interfaces:**
- Consumes: `authrepo`, `authservice`, `Server.auth`.
- Produces: rate limiter wraps `/auth/login` and `/auth/register`.

- [ ] **Step 1: Write the failing test (rate limiter unit)**

Create `backend/cmd/api/ratelimit_test.go`:
```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitAllowsWithinBudget(t *testing.T) {
	rl := newRateLimiter(2, time.Minute)
	ok := false
	handler := rl.wrap(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		handler(w, httptest.NewRequest("POST", "/auth/login", nil))
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	_ = ok
}

func TestRateLimitExceedsBudget(t *testing.T) {
	rl := newRateLimiter(2, time.Minute)
	handler := rl.wrap(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	for i := 0; i < 2; i++ {
		handler(httptest.NewRecorder(), httptest.NewRequest("POST", "/auth/login", nil))
	}
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest("POST", "/auth/login", nil))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run from `backend/`: `go test ./cmd/api/ -run TestRateLimit -v`
Expected: FAIL (`newRateLimiter` undefined).

- [ ] **Step 3: Write minimal implementation**

Create `backend/cmd/api/ratelimit.go`:
```go
package main

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// rateLimiter is a simple in-memory per-IP token bucket used to slow brute-force
// attempts against auth endpoints. MVP scope: single instance, process-local.
type rateLimiter struct {
	mu      sync.Mutex
	budget  int
	window  time.Duration
	buckets map[string]*bucket
}

type bucket struct {
	count   int
	resetsAt time.Time
}

// newRateLimiter returns a limiter allowing `budget` requests per `window` per IP.
func newRateLimiter(budget int, window time.Duration) *rateLimiter {
	return &rateLimiter{budget: budget, window: window, buckets: map[string]*bucket{}}
}

func (rl *rateLimiter) wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		rl.mu.Lock()
		b, ok := rl.buckets[ip]
		now := time.Now()
		if !ok || now.After(b.resetsAt) {
			b = &bucket{count: 0, resetsAt: now.Add(rl.window)}
			rl.buckets[ip] = b
		}
		b.count++
		allowed := b.count <= rl.budget
		rl.mu.Unlock()
		if !allowed {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run from `backend/`: `go test ./cmd/api/ -v`
Expected: PASS (2 tests).

- [ ] **Step 5: Wire rate limiting + auth into main.go**

Edit `backend/cmd/api/main.go`:
- Add imports: `authrepo`, `authservice`, `time`.
- After `contribRepo`, add:
```go
authRepo := authrepo.NewSQLiteAuthRepo(db)
if err := authRepo.Migrate(); err != nil {
	log.Fatalf("api: migrate auth repo: %v", err)
}
```
- Build the auth service:
```go
jwtSecret := []byte(os.Getenv("AUTH_JWT_SECRET"))
if len(jwtSecret) < 32 {
	log.Fatalf("api: AUTH_JWT_SECRET must be set and at least 32 bytes")
}
authSvc := authservice.NewSessionService(authRepo, jwtSecret)
```
- Switch the server construction to `NewServerWithAuth`:
```go
srv := httptransport.NewServerWithAuth(
	fundRepo, contribRepo,
	os.Getenv("POLAR_WEBHOOK_SECRET"),
	contributeCmd, progressQuery,
	authSvc,
)
```
- Wrap the auth routes with the rate limiter. Because routes are registered inside the mux, add the limiter as a middleware on the Server's ServeHTTP for `/auth/login` and `/auth/register`. Implement in main by wrapping the whole server with a small router override in `ratelimit.go`:
```go
func withAuthRateLimit(srv http.Handler) http.Handler {
	rl := newRateLimiter(10, time.Minute)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login", "/auth/register":
			rl.wrap(func(_ http.ResponseWriter, _ *http.Request) {}).ServeHTTP(w, r) // placeholder no-op; real path below
		}
		srv.ServeHTTP(w, r)
	})
}
```
**Correction for Step 5:** the no-op above is wrong. Instead, apply the limiter by wrapping the auth handlers at construction. Since the limiter belongs in `internal/interfaces/http`, do this instead (see Task 7). For main.go, simply serve `srv` directly and let Task 7 wire the limiter as middleware in the http package.

```go
// Final main.go wiring:
// - db open, migrate fund+contrib+auth repos
// - polar client, contributeCmd, progressQuery, authSvc
// - srv := httptransport.NewServerWithAuth(...)
// - http.ListenAndServe(addr, httptransport.WithRateLimit(srv, authEndpoints))
```
Update `http.ListenAndServe(addr, srv)` to `http.ListenAndServe(addr, httptransport.WithRateLimit(srv))`.

- [ ] **Step 6: Commit**

```bash
git add backend/cmd/api/ main.go ratelimit.go ratelimit_test.go
git commit -m "feat: wire auth into main with rate limiting"
```

*(The `WithRateLimit` middleware is implemented in Task 7.)*

---

### Task 7: Rate-limit middleware in http package (finalize wiring)

**Files:**
- Modify: `backend/internal/interfaces/http/server.go`
- Create: `backend/internal/interfaces/http/middleware.go`
- Test: `backend/internal/interfaces/http/middleware_test.go`

**Interfaces:**
- Produces: `func WithRateLimit(next http.Handler) http.Handler` applying a 10 req/min per-IP limit on `/auth/login` and `/auth/register` only; all other paths pass through unchanged.

- [ ] **Step 1: Write the failing test**

Create `backend/internal/interfaces/http/middleware_test.go`:
```go
package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithRateLimit_BlocksAuthAfterBudget(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	h := WithRateLimit(inner)
	// 10 allowed on /auth/login, 11th blocked.
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/auth/login", nil))
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/auth/login", nil))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on 11th, got %d", w.Code)
	}
}

func TestWithRateLimit_PassesOtherPaths(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	h := WithRateLimit(inner)
	for i := 0; i < 30; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/funds/x", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("non-auth path should pass, got %d", w.Code)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run from `backend/`: `go test ./internal/interfaces/http/ -run TestWithRateLimit -v`
Expected: FAIL (`WithRateLimit` undefined).

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/interfaces/http/middleware.go`:
```go
package http

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// WithRateLimit applies a per-IP limit on auth endpoints (brute-force defense).
func WithRateLimit(next http.Handler) http.Handler {
	rl := &slidingLimiter{window: time.Minute, limit: 10, hits: map[string][]time.Time{}}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/login" && r.URL.Path != "/auth/register" {
			next.ServeHTTP(w, r)
			return
		}
		if !rl.allow(clientAddr(r)) {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type slidingLimiter struct {
	mu     sync.Mutex
	window time.Duration
	limit  int
	hits   map[string][]time.Time
}

func (l *slidingLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-l.window)
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	l.hits[key] = kept
	if len(kept) >= l.limit {
		return false
	}
	l.hits[key] = append(l.hits[key], now)
	return true
}

func clientAddr(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run from `backend/`: `go test ./internal/interfaces/http/ -run TestWithRateLimit -v`
Expected: PASS (2 tests).

- [ ] **Step 5: Update main.go to use WithRateLimit**

Edit `backend/cmd/api/main.go`:
- Change `http.ListenAndServe(addr, srv)` → `http.ListenAndServe(addr, httptransport.WithRateLimit(srv))`.

- [ ] **Step 6: Full verification**

Run from `backend/`:
```bash
gofmt -l .
go build ./...
go vet ./...
go test ./...
```
Expected: no diff from gofmt, build exit 0, vet exit 0, all tests pass.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/interfaces/http/middleware.go backend/internal/interfaces/http/middleware_test.go backend/cmd/api/main.go
git commit -m "feat: rate-limit auth endpoints and wire middleware"
```

---

### Task 8: Env documentation + end-to-end smoke verification

**Files:**
- Modify: `backend/.env.example`

**Interfaces:**
- Consumes: `AUTH_JWT_SECRET` env var.
- Produces: documented env contract.

- [ ] **Step 1: Add env docs**

Append to `backend/.env.example`:
```
# Auth — REQUIRED, at least 32 bytes (use: openssl rand -hex 32)
AUTH_JWT_SECRET=
```

- [ ] **Step 2: Write the end-to-end smoke test**

Create `backend/internal/interfaces/http/auth_e2e_test.go`:
```go
package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAuthE2E exercises register -> login -> me with a real server.
func TestAuthE2E(t *testing.T) {
	srv := newAuthServer(t)

	// register
	w := doJSON(t, srv, http.MethodPost, "/auth/register", `{"email":"e2e@b.com","password":"password123","name":"E2E"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}
	var reg map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&reg)
	access, _ := reg["access_token"].(string)
	if access == "" {
		t.Fatal("no access token")
	}

	// me with the access token
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("me: %d %s", w.Code, w.Body.String())
	}
	var me map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&me)
	if me["email"] != "e2e@b.com" {
		t.Fatalf("unexpected me: %v", me)
	}

	// refresh via cookie
	cookie, err := w.Result().CookiesExternal() // placeholder — see step 3 correction
	_ = cookie
	_ = err
}
```

- [ ] **Step 3: Complete the refresh step in the test (use the cookie helper)**

Edit the end of `TestAuthE2E`: capture the refresh cookie from a fresh login and POST `/auth/refresh` with it:
```go
	// login to get a refresh cookie
	w = doJSON(t, srv, http.MethodPost, "/auth/login", `{"email":"e2e@b.com","password":"password123"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	var refreshCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == refreshCookieName {
			refreshCookie = c
			break
		}
	}
	if refreshCookie == nil {
		t.Fatal("no refresh cookie")
	}
	// use it
	rreq := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(nil))
	rreq.AddCookie(refreshCookie)
	rrw := httptest.NewRecorder()
	srv.ServeHTTP(rrw, rreq)
	if rrw.Code != http.StatusOK {
		t.Fatalf("refresh: %d %s", rrw.Code, rrw.Body.String())
	}
```
Remove the placeholder lines from Step 2 and replace with the above.

- [ ] **Step 4: Run tests to verify they pass**

Run from `backend/`: `go test ./internal/interfaces/http/ -run TestAuthE2E -v`
Expected: PASS.

- [ ] **Step 5: Full verification**

Run from `backend/`: `gofmt -l . && go build ./... && go vet ./... && go test ./...`
Expected: all clean and green.

- [ ] **Step 6: Commit**

```bash
git add backend/.env.example backend/internal/interfaces/http/auth_e2e_test.go
git commit -m "test: auth end-to-end smoke + document AUTH_JWT_SECRET"
```
