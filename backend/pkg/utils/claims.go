package utils

import "time"

// TokenClaims adalah isi token sesi yang sudah diverifikasi.
// Didefinisikan di paket utils agar issuer dan konsumennya tidak saling
// bergantung dan tidak membentuk import cycle.
type TokenClaims struct {
	UserID    string
	Role      string
	Scope     string
	Session   int
	SessionID string
	ExpiresAt time.Time
}
