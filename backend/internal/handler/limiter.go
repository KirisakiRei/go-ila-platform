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
