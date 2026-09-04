// Package authrepo provides authentication repository contracts and domain types.
package authrepo

import (
	"time"
)

// User represents an authenticated account.
type User struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	CreatedAt    time.Time
}

// RefreshToken represents a long-lived refresh token for a user.
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// AuthRepository defines persistence operations for users and refresh tokens.
type AuthRepository interface {
	SaveUser(user *User) error
	FindUserByEmail(email string) (*User, error)
	SaveRefreshToken(token *RefreshToken) error
	FindRefreshToken(tokenHash string) (*RefreshToken, error)
	DeleteRefreshToken(id string) error
	DeleteRefreshTokensForUser(userID string) error
	ListRefreshTokens(userID string) ([]*RefreshToken, error)
}
