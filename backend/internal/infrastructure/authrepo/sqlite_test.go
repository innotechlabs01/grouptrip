package authrepo

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	// Use a unique temp file per test.
	dbPath := "file:" + t.TempDir() + "/auth_test.db"
	db, err := sql.Open("libsql", dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSQLiteAuthRepo_Migrate(t *testing.T) {
	db := openTestDB(t)
	repo := NewSQLiteAuthRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	// Verify tables exist by querying sqlite_master
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count); err != nil {
		t.Fatalf("check users table: %v", err)
	}
	if count != 1 {
		t.Fatalf("users table missing")
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='refresh_tokens'").Scan(&count); err != nil {
		t.Fatalf("check refresh_tokens table: %v", err)
	}
	if count != 1 {
		t.Fatalf("refresh_tokens table missing")
	}
}

func TestSQLiteAuthRepo_SaveAndFindUser(t *testing.T) {
	db := openTestDB(t)
	repo := NewSQLiteAuthRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	user := &User{
		ID:           "u1",
		Email:        "a@b.com",
		Name:         "Ana",
		PasswordHash: "hash",
		CreatedAt:    time.Now().UTC(),
	}
	if err := repo.SaveUser(user); err != nil {
		t.Fatalf("save user: %v", err)
	}
	found, err := repo.FindUserByEmail("a@b.com")
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if found.ID != user.ID || found.Email != user.Email {
		t.Fatalf("unexpected user: %+v", found)
	}
}

func TestSQLiteAuthRepo_SaveAndFindRefreshToken(t *testing.T) {
	db := openTestDB(t)
	repo := NewSQLiteAuthRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	token := &RefreshToken{
		ID:        "rt1",
		UserID:    "u1",
		TokenHash: "hash123",
		ExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
		CreatedAt: time.Now().UTC(),
	}
	if err := repo.SaveRefreshToken(token); err != nil {
		t.Fatalf("save token: %v", err)
	}
	found, err := repo.FindRefreshToken("hash123")
	if err != nil {
		t.Fatalf("find token: %v", err)
	}
	if found.ID != token.ID || found.UserID != token.UserID {
		t.Fatalf("unexpected token: %+v", found)
	}
}

func TestSQLiteAuthRepo_DeleteRefreshToken(t *testing.T) {
	db := openTestDB(t)
	repo := NewSQLiteAuthRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	token := &RefreshToken{
		ID:        "rt2",
		UserID:    "u1",
		TokenHash: "hashdel",
		ExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
		CreatedAt: time.Now().UTC(),
	}
	if err := repo.SaveRefreshToken(token); err != nil {
		t.Fatalf("save token: %v", err)
	}
	if err := repo.DeleteRefreshToken("rt2"); err != nil {
		t.Fatalf("delete token: %v", err)
	}
	found, err := repo.FindRefreshToken("hashdel")
	if err == nil {
		t.Fatalf("expected not found, got %+v", found)
	}
}

func TestSQLiteAuthRepo_ListRefreshTokens(t *testing.T) {
	db := openTestDB(t)
	repo := NewSQLiteAuthRepo(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	tokens := []*RefreshToken{
		{ID: "rt3", UserID: "u1", TokenHash: "h1", ExpiresAt: time.Now().Add(24 * time.Hour).UTC(), CreatedAt: time.Now().UTC()},
		{ID: "rt4", UserID: "u1", TokenHash: "h2", ExpiresAt: time.Now().Add(24 * time.Hour).UTC(), CreatedAt: time.Now().UTC()},
		{ID: "rt5", UserID: "u2", TokenHash: "h3", ExpiresAt: time.Now().Add(24 * time.Hour).UTC(), CreatedAt: time.Now().UTC()},
	}
	for _, tk := range tokens {
		if err := repo.SaveRefreshToken(tk); err != nil {
			t.Fatalf("save token: %v", err)
		}
	}
	list, err := repo.ListRefreshTokens("u1")
	if err != nil {
		t.Fatalf("list tokens: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 tokens for u1, got %d", len(list))
	}
}
