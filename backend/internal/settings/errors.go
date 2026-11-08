package settings

import "errors"

// Kesalahan validasi nilai flag. Dipetakan ke 400 oleh lapisan service.
var (
	ErrUnknownKey = errors.New("flag tidak dikenal")
	ErrEmptyValue = errors.New("nilai flag tidak boleh kosong")
	ErrBadValue   = errors.New("nilai flag tidak sesuai jenisnya")
	ErrOutOfRange = errors.New("nilai flag di luar rentang yang diizinkan")
)
