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

var (
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrEmailTaken         = errors.New("authrepo: email already taken")
	ErrInvalidRefresh     = errors.New("auth: invalid refresh token")
	ErrRefreshReuse       = errors.New("auth: refresh token reuse detected")
)

const accessTokenTTL = 15 * time.Minute
const refreshTokenTTL = 30 * 24 * time.Hour

type TokenClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

type UserView struct {
	ID, Email, Name string
}

type SessionService struct {
	Users     *authrepo.SQLiteAuthRepo
	JWTSecret []byte
}

func NewSessionService(users *authrepo.SQLiteAuthRepo, jwtSecret []byte) *SessionService {
	return &SessionService{Users: users, JWTSecret: jwtSecret}
}

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
	// Create user using repo adapter
	if err := s.Users.CreateUser(ctx, u); err != nil {
		if err.Error() == "authrepo: email already taken" {
			return "", "", ErrEmailTaken
		}
		return "", "", err
	}
	return s.issueTokens(ctx, u.ID)
}

func (s *SessionService) Login(ctx context.Context, email, password string) (string, string, error) {
	ru, err := s.Users.FindUserByEmail(email)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}
	ok, err := auth.VerifyPassword(password, ru.PasswordHash)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", ErrInvalidCredentials
	}
	return s.issueTokens(ctx, ru.ID)
}

func (s *SessionService) Refresh(ctx context.Context, provided string) (string, string, error) {
	hash := auth.HashToken(provided)
	tok, err := s.Users.FindRefreshToken(hash)
	if err != nil {
		return "", "", ErrInvalidRefresh
	}
	if time.Now().After(tok.ExpiresAt) {
		return "", "", ErrInvalidRefresh
	}
	// replay detection: if token no longer exists after we tried to use it, it's reuse
	// For simplicity, we check if IsTokenUsed via existence
	used, _ := s.Users.IsTokenUsed(ctx, hash)
	if used {
		_ = s.Users.RevokeUserTokens(ctx, tok.UserID)
		return "", "", ErrRefreshReuse
	}
	if err := s.Users.MarkTokenUsed(ctx, hash); err != nil {
		return "", "", err
	}
	return s.issueTokens(ctx, tok.UserID)
}

func (s *SessionService) Logout(ctx context.Context, userID string) error {
	return s.Users.RevokeUserTokens(ctx, userID)
}

func (s *SessionService) Me(ctx context.Context, userID string) (*UserView, error) {
	// use FindByID adapter
	u, err := s.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &UserView{ID: u.ID, Email: u.Email, Name: u.Name}, nil
}

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
