package pagination

import (
	"time"

	"ila/backend/internal/timezone"
)

// WIBLocation mengembalikan zona hari akuntansi untuk pemformatan tanggal.
func WIBLocation() *time.Location {
	return timezone.WIB
}
