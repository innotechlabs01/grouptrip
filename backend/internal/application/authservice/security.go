package authservice

import (
	"errors"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
)

func NewPasswordHash(password []byte) (string, error) {
	hash, err := argon2id.CreateHash(string(password), argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hash, nil
}

func ComparePasswordHash(password []byte, hash string) bool {
	match, err := argon2id.ComparePasswordAndHash(string(password), hash)
	if err != nil {
		return false
	}
	return match
}

type claims struct {
	UserID string `json:"sub"`
	jwt.RegisteredClaims
}

func CreateAccessToken(userID string, secret []byte) (string, error) {
	now := time.Now()
	claims := claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}
	return signed, nil
}

func ValidateAccessToken(tokenStr string, secret []byte) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(*claims); ok && token.Valid {
		if claims.UserID == "" {
			return "", errors.New("missing sub claim")
		}
		return claims.UserID, nil
	}
	return "", errors.New("invalid token")
}
