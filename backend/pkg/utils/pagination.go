package utils

import (
	"math"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/dto"
)

const (
	defaultPage    = 1
	defaultPerPage = 10
	maxPerPage     = 100
)

// Normalize membatasi nilai pagination ke rentang aman.
// Query tanpa batas adalah sumber memory leak di produksi, jadi nilai selalu dijepit.
func Normalize(q *dto.PaginationQuery) {
	if q.Page < 1 {
		q.Page = defaultPage
	}
	if q.PerPage < 1 {
		q.PerPage = defaultPerPage
	}
	if q.PerPage > maxPerPage {
		q.PerPage = maxPerPage
	}

	switch q.Order {
	case "asc", "desc":
	default:
		q.Order = "desc"
	}
}

// Offset menghitung offset SQL dari nomor halaman.
func Offset(page, perPage int) int {
	return (page - 1) * perPage
}

// BuildMeta menyusun metadata pagination untuk balasan API.
func BuildMeta(page, perPage int, totalItems int64) dto.PaginationMeta {
	totalPages := 0
	if perPage > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(perPage)))
	}

	return dto.PaginationMeta{
		CurrentPage: page,
		PerPage:     perPage,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}
}

// SafeSortColumn memastikan kolom pengurutan hanya berasal dari daftar putih,
// agar nama kolom dari klien tidak bisa disisipkan ke query SQL.
func SafeSortColumn(requested string, allowed map[string]string, fallback string) string {
	if col, ok := allowed[requested]; ok {
		return col
	}
	return fallback
}
