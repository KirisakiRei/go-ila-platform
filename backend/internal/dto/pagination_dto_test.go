package dto_test

import (
	"testing"

	"ila/backend/internal/dto"
)

// Tanpa batas atas, satu permintaan bisa menarik seluruh tabel (plans/24 §2.1).
func TestPerPageDipaksaBatasAtas(t *testing.T) {
	q := dto.PaginationQuery{Page: 1, PerPage: 9999}.Normalized()
	if q.PerPage != dto.MaxPerPage {
		t.Fatalf("per_page = %d, ingin %d", q.PerPage, dto.MaxPerPage)
	}
}

// Nilai nol atau negatif diganti bawaan, bukan menghasilkan query aneh.
func TestNilaiTidakSahDigantiBawaan(t *testing.T) {
	q := dto.PaginationQuery{Page: 0, PerPage: 0, Order: "acak"}.Normalized()

	if q.Page != 1 {
		t.Fatalf("page = %d, ingin 1", q.Page)
	}
	if q.PerPage != dto.DefaultPerPage {
		t.Fatalf("per_page = %d, ingin %d", q.PerPage, dto.DefaultPerPage)
	}
	if q.Order != "desc" {
		t.Fatalf("order = %q, ingin desc", q.Order)
	}
}

// Offset dihitung dari halaman, bukan dari nomor baris langsung.
func TestOffsetPaginasi(t *testing.T) {
	q := dto.PaginationQuery{Page: 3, PerPage: 50}.Normalized()
	if got := q.Offset(); got != 100 {
		t.Fatalf("offset = %d, ingin 100", got)
	}
}

// Meta harus memberi tahu apakah masih ada halaman berikutnya.
func TestMetaHasMore(t *testing.T) {
	q := dto.PaginationQuery{Page: 1, PerPage: 10}.Normalized()

	first := dto.NewPaginationMeta(q, 25)
	if !first.HasMore {
		t.Fatal("halaman 1 dari 25 baris harus punya halaman berikutnya")
	}
	if first.TotalPages != 3 {
		t.Fatalf("total_pages = %d, ingin 3", first.TotalPages)
	}

	last := dto.NewPaginationMeta(dto.PaginationQuery{Page: 3, PerPage: 10}.Normalized(), 25)
	if last.HasMore {
		t.Fatal("halaman terakhir tidak boleh menandai has_more")
	}
}

// Hasil kosong tetap menghasilkan meta yang masuk akal, bukan bagi nol.
func TestMetaTanpaData(t *testing.T) {
	meta := dto.NewPaginationMeta(dto.PaginationQuery{Page: 1, PerPage: 50}.Normalized(), 0)
	if meta.TotalPages != 0 || meta.HasMore {
		t.Fatalf("meta kosong salah: %+v", meta)
	}
}

// Nama kolom dari klien hanya boleh lewat daftar putih.
func TestSafeSortColumnMenolakKolomAsing(t *testing.T) {
	allowed := map[string]string{"created": "created_at"}

	if got := dto.SafeSortColumn("created", allowed, "id"); got != "created_at" {
		t.Fatalf("kolom sah = %q, ingin created_at", got)
	}
	if got := dto.SafeSortColumn("password_hash", allowed, "id"); got != "id" {
		t.Fatalf("kolom asing = %q, ingin fallback id", got)
	}
	if got := dto.SafeSortColumn("id; DROP TABLE user", allowed, "id"); got != "id" {
		t.Fatalf("percobaan injeksi = %q, ingin fallback id", got)
	}
}
