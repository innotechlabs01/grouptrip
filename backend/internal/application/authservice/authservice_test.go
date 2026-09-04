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

	_, newRT, err := svc.Refresh(context.Background(), rt)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if newRT == rt {
		t.Fatal("refresh must rotate")
	}

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
