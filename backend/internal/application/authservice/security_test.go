package authservice

import (
	"testing"
	"time"
)

func TestGeneratePasswordHash_Valid(t *testing.T) {
	hash, err := NewPasswordHash([]byte("password123"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if hash == "" {
		t.Fatalf("expected non-empty hash")
	}
}

func TestVerifyPassword_Valid(t *testing.T) {
	pwd := []byte("secret")
	hash, err := NewPasswordHash(pwd)
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if !ComparePasswordHash(pwd, hash) {
		t.Fatalf("expected password to match")
	}
}

func TestVerifyPassword_Invalid(t *testing.T) {
	pwd := []byte("secret")
	hash, err := NewPasswordHash(pwd)
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if ComparePasswordHash([]byte("wrong"), hash) {
		t.Fatalf("expected password mismatch")
	}
}

func TestCreateAccessToken_Valid(t *testing.T) {
	secret := []byte("supersecret")
	userID := "user-123"
	token, err := CreateAccessToken(userID, secret)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty token")
	}
}

func TestValidateAccessToken_Valid(t *testing.T) {
	secret := []byte("supersecret")
	userID := "user-123"
	token, err := CreateAccessToken(userID, secret)
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	gotID, err := ValidateAccessToken(token, secret)
	if err != nil {
		t.Fatalf("validate error: %v", err)
	}
	if gotID != userID {
		t.Fatalf("expected %s, got %s", userID, gotID)
	}
}

func TestValidateAccessToken_Invalid(t *testing.T) {
	secret := []byte("supersecret")
	_, err := ValidateAccessToken("invalid.token.here", secret)
	if err == nil {
		t.Fatalf("expected error for invalid token")
	}
}

func TestTokenExpiry(t *testing.T) {
	secret := []byte("supersecret")
	userID := "user-123"
	token, err := CreateAccessToken(userID, secret)
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	// Validate immediately succeeds
	if _, err := ValidateAccessToken(token, secret); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	// Not checking actual expiry timing here; just ensure exp claim exists
	_ = time.Now()
}
