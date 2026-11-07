package utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTIssuer menerbitkan dan memverifikasi token sesi berbasis HS256.
type JWTIssuer struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTIssuer membuat penerbit token dengan masa berlaku tertentu.
func NewJWTIssuer(secret string, ttl time.Duration) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret), ttl: ttl}
}

type claimsPayload struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// Issue membuat token untuk satu akun.
func (j *JWTIssuer) Issue(userID, role string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(j.ttl)

	claims := claimsPayload{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    "ila-platform",
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("menandatangani token: %w", err)
	}
	return token, expiresAt, nil
}

// Verify memeriksa tanda tangan dan masa berlaku token.
func (j *JWTIssuer) Verify(raw string) (TokenClaims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	var claims claimsPayload
	_, err := parser.ParseWithClaims(raw, &claims, func(*jwt.Token) (any, error) {
		return j.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return TokenClaims{}, fmt.Errorf("sesi berakhir: %w", err)
		}
		return TokenClaims{}, fmt.Errorf("token tidak sah: %w", err)
	}

	return TokenClaims{
		UserID:    claims.Subject,
		Role:      claims.Role,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}
