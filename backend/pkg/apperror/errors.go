package apperror

import (
	"errors"
	"net/http"
)

// ErrNotFound: data yang diminta tidak ada di penyimpanan.
var ErrNotFound = errors.New("resource not found")

// ErrAlreadyExists: data bentrok dengan yang sudah tersimpan.
var ErrAlreadyExists = errors.New("resource already exists")

// ErrUnauthorized: identitas tidak terbukti.
var ErrUnauthorized = errors.New("unauthorized access")

// ErrForbidden: identitas sah tetapi tidak berwenang.
var ErrForbidden = errors.New("forbidden resource")

// ErrBadRequest: payload atau parameter tidak sah.
var ErrBadRequest = errors.New("invalid request payload")

// ErrConflict: keadaan sistem menolak operasi pada saat ini.
var ErrConflict = errors.New("operation conflicts with current state")

// ErrInternalServer: kegagalan di luar kendali pemanggil.
var ErrInternalServer = errors.New("internal server error")

// ErrServiceUnavailable: dependensi eksternal tidak bisa dihubungi.
var ErrServiceUnavailable = errors.New("service temporarily unavailable")

// ErrTooManyRequests: terlalu banyak percobaan dalam jendela waktu.
var ErrTooManyRequests = errors.New("too many requests")

// MapHTTPStatus menerjemahkan domain error ke pasangan status HTTP dan kode error.
func MapHTTPStatus(err error) (int, string) {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, ErrAlreadyExists):
		return http.StatusConflict, "ALREADY_EXISTS"
	case errors.Is(err, ErrConflict):
		return http.StatusConflict, "CONFLICT"
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized, "UNAUTHORIZED"
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden, "FORBIDDEN"
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest, "BAD_REQUEST"
	case errors.Is(err, ErrTooManyRequests):
		return http.StatusTooManyRequests, "TOO_MANY_REQUESTS"
	case errors.Is(err, ErrServiceUnavailable):
		return http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"
	default:
		return http.StatusInternalServerError, "INTERNAL_SERVER_ERROR"
	}
}
