package utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Scope memisahkan pintu sesi user dan admin. Token yang membawa scope salah
// ditolak oleh penjaga masing-masing (plans/03 §2).
const (
	ScopeUser  = "user"
	ScopeAdmin = "admin"
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

// Claims berisi identitas yang perlu ditanam di token.
type Claims struct {
	UserID  string
	Role    string
	Scope   string
	Session int
}

type claimsPayload struct {
	Role    string `json:"role"`
	Scope   string `json:"scope,omitempty"`
	Session int    `json:"sv,omitempty"`
	jwt.RegisteredClaims
}

// Issue membuat token untuk satu akun.
func (j *JWTIssuer) Issue(userID, role string) (string, time.Time, error) {
	return j.IssueWithClaims(Claims{UserID: userID, Role: role, Scope: ScopeUser})
}

// IssueWithClaims membuat token dengan scope dan session version eksplisit.
func (j *JWTIssuer) IssueWithClaims(c Claims) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(j.ttl)

	claims := claimsPayload{
		Role:    c.Role,
		Scope:   c.Scope,
		Session: c.Session,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   c.UserID,
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

	scope := claims.Scope
	if scope == "" {
		// Token lama sebelum scope diperkenalkan tetap dianggap sesi user.
		scope = ScopeUser
	}

	return TokenClaims{
		UserID:    claims.Subject,
		Role:      claims.Role,
		Scope:     scope,
		Session:   claims.Session,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}
