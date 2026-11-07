// Package authstate menyimpan bentuk ringkas akun yang dipakai untuk
// memvalidasi sesi tiap permintaan. Diletakkan di paket terpisah agar
// repository, middleware, dan service bisa memakainya tanpa import cycle.
package authstate

import (
	"context"

	"ila/backend/internal/model"
)

// User adalah data akun yang dibutuhkan untuk memvalidasi sesi.
type User struct {
	ID             string
	Role           string
	Status         string
	SessionVersion int
}

// Reader membaca akun berdasarkan ID.
type Reader interface {
	FindSessionUser(ctx context.Context, id string) (*User, error)
}

// Status akun yang dikenal platform.
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusBanned    = "banned"
)

// CanUsePlatform menandai status yang diizinkan masuk. Suspended boleh melihat
// data tetapi tidak bisa eksekusi; pembatasan itu dilakukan Runner.
func CanUsePlatform(status string) bool {
	return status == StatusActive || status == StatusSuspended
}

// FromUser memetakan model user ke bentuk ringkas.
func FromUser(u *model.User) *User {
	if u == nil {
		return nil
	}
	return &User{
		ID:             u.ID,
		Role:           u.Role,
		Status:         u.Status,
		SessionVersion: u.SessionVersion,
	}
}
