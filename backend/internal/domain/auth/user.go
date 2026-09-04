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
