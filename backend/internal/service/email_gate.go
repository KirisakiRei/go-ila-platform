package service

import (
	"errors"

	"ila/backend/pkg/apperror"
)

// ErrEmailNotVerified menolak langkah yang menuntut alamat surel terbukti.
// Login tetap boleh; yang dikunci hanya trial, broker, dan checkout
// (plans/17 §2.5).
var ErrEmailNotVerified = errors.Join(apperror.ErrForbidden,
	errors.New("Verifikasi email Anda terlebih dahulu. Buka Pengaturan → Keamanan untuk mengirim ulang tautan."))
