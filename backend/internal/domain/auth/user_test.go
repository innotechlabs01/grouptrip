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
