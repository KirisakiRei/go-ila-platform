package handler

import "time"

// LoginLimiter membatasi percobaan masuk yang gagal.
// Didefinisikan di sisi pemakai agar handler tidak terikat implementasi.
type LoginLimiter interface {
	Blocked(key string) bool
	Fail(key string)
	Reset(key string)
	RetryAfter(key string) time.Duration
}

// RegisterLimiter membatasi pendaftaran per IP dan per alamat surel.
// Dideklarasikan di sisi pemakai agar handler tidak terikat implementasi.
type RegisterLimiter interface {
	Allow(ip, email string) (bool, int)
	Record(ip, email string)
}
