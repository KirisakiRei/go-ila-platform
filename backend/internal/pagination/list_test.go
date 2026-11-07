package pagination

import (
	"errors"
	"strings"
	"testing"
	"time"

	"ila/backend/pkg/apperror"
)

var testSort = map[string]string{"created_at": "created_at", "name": "name"}

func TestPerPageCappedAt200(t *testing.T) {
	p, err := Build(Request{PerPage: 9999, AllowedSort: testSort, DefaultSort: "created_at"})
	if err != nil {
		t.Fatalf("tidak boleh gagal: %v", err)
	}
	if p.PerPage != MaxPerPage {
		t.Fatalf("per_page harus dipaksa %d, dapat %d", MaxPerPage, p.PerPage)
	}
}

func TestPageZeroBecomesOne(t *testing.T) {
	p, err := Build(Request{Page: 0, AllowedSort: testSort, DefaultSort: "created_at"})
	if err != nil {
		t.Fatalf("tidak boleh gagal: %v", err)
	}
	if p.Page != 1 {
		t.Fatalf("halaman 0 harus menjadi 1, dapat %d", p.Page)
	}
}

func TestUnknownSortRejected(t *testing.T) {
	_, err := Build(Request{Sort: "id;DROP TABLE", AllowedSort: testSort, DefaultSort: "created_at"})
	if !errors.Is(err, ErrSortNotAllowed) {
		t.Fatalf("sort tak dikenal harus ditolak, dapat %v", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatal("error harus bertipe BAD_REQUEST")
	}
}

func TestWildcardEscaped(t *testing.T) {
	escaped := EscapeLike("100%_x")
	if strings.Contains(escaped, "%") && !strings.Contains(escaped, `\%`) {
		t.Fatal("tanda persen harus di-escape")
	}
	if !strings.Contains(escaped, `\_`) {
		t.Fatal("garis bawah harus di-escape")
	}
}

func TestQueryTruncated(t *testing.T) {
	p, err := Build(Request{Query: strings.Repeat("a", 500), AllowedSort: testSort, DefaultSort: "created_at"})
	if err != nil {
		t.Fatalf("tidak boleh gagal: %v", err)
	}
	if len(p.Query) != MaxQueryLength {
		t.Fatalf("q harus dipotong %d karakter, dapat %d", MaxQueryLength, len(p.Query))
	}
}

func TestSameDayRangeIs24HoursWIB(t *testing.T) {
	p, err := Build(Request{From: "2026-11-05", To: "2026-11-05", AllowedSort: testSort, DefaultSort: "created_at"})
	if err != nil {
		t.Fatalf("tidak boleh gagal: %v", err)
	}
	if p.From == nil || p.To == nil {
		t.Fatal("rentang harus terisi")
	}
	// 00:00 WIB = 17:00 UTC hari sebelumnya.
	if p.From.UTC().Format("15:04") != "17:00" {
		t.Fatalf("awal rentang harus 17:00 UTC, dapat %s", p.From.UTC().Format("15:04"))
	}
	span := p.To.Sub(*p.From)
	if span != 24*time.Hour {
		t.Fatalf("rentang sehari harus 24 jam, dapat %v", span)
	}
}

func TestRangeOver366DaysRejected(t *testing.T) {
	_, err := Build(Request{From: "2025-01-01", To: "2026-11-05", AllowedSort: testSort, DefaultSort: "created_at"})
	if !errors.Is(err, ErrRangeTooLong) {
		t.Fatalf("rentang panjang harus ditolak, dapat %v", err)
	}
}

func TestAuditRequiresRangeDefaultsTo7Days(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	p, err := Build(Request{AllowedSort: testSort, DefaultSort: "created_at", RequireRange: true, Now: now})
	if err != nil {
		t.Fatalf("tidak boleh gagal: %v", err)
	}
	if p.From == nil {
		t.Fatal("rentang bawaan harus terisi")
	}
	// Bawaan 7 hari: hari ini dan enam hari sebelumnya.
	expected := time.Date(2026, 10, 30, 0, 0, 0, 0, time.FixedZone("WIB", 7*3600)).UTC()
	if !p.From.Equal(expected) {
		t.Fatalf("awal rentang harus %s, dapat %s", expected, p.From)
	}
}

func TestCursorMismatchRejected(t *testing.T) {
	cur := Cursor{Value: "2026-11-05T00:00:00Z", ID: "abc", Sort: "created_at", Order: "desc"}.Encode()
	_, err := Build(Request{Cursor: cur, Sort: "name", Order: "desc", AllowedSort: testSort, DefaultSort: "created_at"})
	if !errors.Is(err, ErrCursorMismatch) {
		t.Fatalf("kursor dengan sort berbeda harus ditolak, dapat %v", err)
	}
}

func TestCursorRoundTrip(t *testing.T) {
	cur := Cursor{Value: "2026-11-05T00:00:00Z", ID: "abc", Sort: "created_at", Order: "desc"}
	decoded, err := DecodeCursor(cur.Encode())
	if err != nil {
		t.Fatalf("kursor sah harus terbaca: %v", err)
	}
	if decoded.ID != "abc" || decoded.Value != cur.Value {
		t.Fatal("isi kursor harus sama setelah encode-decode")
	}
}

func TestStatusFilterRejectsUnknown(t *testing.T) {
	_, err := ParseStatusFilter("active,ngawur", []string{"active", "banned"})
	if err == nil {
		t.Fatal("status tak dikenal harus ditolak")
	}
	values, err := ParseStatusFilter("active,banned,active", []string{"active", "banned"})
	if err != nil {
		t.Fatalf("status sah harus diterima: %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("duplikat harus dibuang, dapat %d", len(values))
	}
}
