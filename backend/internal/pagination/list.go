// Package pagination menyatukan aturan daftar, pencarian, filter, dan kursor.
//
// Satu tempat agar batas per_page, escape wildcard, konversi tanggal WIB, dan
// penyusunan kursor tidak bisa berbeda antar endpoint (plans/24 §8.1).
package pagination

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"ila/backend/internal/timezone"
	"ila/backend/pkg/apperror"
)

// Batas keras daftar. Nilai ini mengikat semua endpoint.
const (
	DefaultPerPage = 50
	MaxPerPage     = 200
	MaxQueryLength = 100
	MaxRangeDays   = 366
	ExportMaxRows  = 10000
)

// Kesalahan yang bisa dibedakan pemanggil.
var (
	ErrCursorMismatch = errors.New("KURSOR_TIDAK_COCOK")
	ErrRangeTooLong   = errors.New("RENTANG_TERLALU_PANJANG")
	ErrSortNotAllowed = errors.New("URUTAN_TIDAK_DIKENAL")
)

// Params adalah parameter daftar yang sudah divalidasi.
type Params struct {
	Page    int
	PerPage int
	Query   string
	Cursor  Cursor
	Sort    string
	Order   string
	// From dan To adalah batas UTC hasil konversi tanggal WIB. Nil bila kosong.
	From *time.Time
	To   *time.Time
	// CursorMode menandai daftar berbasis kursor, bukan nomor halaman.
	CursorMode bool
}

// Offset menghitung posisi baris pertama. Hanya berlaku pada mode halaman.
func (p Params) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// Cursor menyimpan posisi terakhir agar baris baru tidak menggeser halaman.
type Cursor struct {
	Value string
	ID    string
	Sort  string
	Order string
}

// Present menandai kursor terisi.
func (c Cursor) Present() bool {
	return c.ID != "" || c.Value != ""
}

// Encode menyusun kursor opaque: base64url("<sort_value>|<id>|<sort>|<order>").
// sort dan order ikut dibawa agar kursor tidak bisa dipakai lintas pengurutan.
func (c Cursor) Encode() string {
	if c.ID == "" {
		return ""
	}
	raw := c.Value + "|" + c.ID + "|" + c.Sort + "|" + c.Order
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor membaca kursor. Kursor tidak sah ditolak sebagai 400.
func DecodeCursor(encoded string) (Cursor, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return Cursor{}, nil
	}
	if len(encoded) > 512 {
		return Cursor{}, validation("kursor terlalu panjang")
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Cursor{}, validation("kursor tidak sah")
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 4 {
		return Cursor{}, validation("kursor tidak sah")
	}
	return Cursor{Value: parts[0], ID: parts[1], Sort: parts[2], Order: parts[3]}, nil
}

// Request adalah masukan mentah dari query string.
type Request struct {
	Page    int
	PerPage int
	Query   string
	Cursor  string
	Sort    string
	Order   string
	From    string
	To      string
	// AllowedSort adalah daftar putih kolom untuk endpoint ini.
	AllowedSort map[string]string
	// DefaultSort dan CursorMode ditentukan per endpoint.
	DefaultSort string
	CursorMode  bool
	// RequireRange menolak permintaan tanpa rentang tanggal (dipakai audit).
	RequireRange bool
	// DefaultRangeDays dipakai bila RequireRange dan pengguna mengosongkan.
	DefaultRangeDays int
	// Now dipakai bila pemanggil perlu waktu tetap, misalnya pada pengujian.
	Now time.Time
}

// Build memvalidasi masukan menjadi Params yang siap dipakai repository.
func Build(req Request) (Params, error) {
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	perPage := req.PerPage
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	page := req.Page
	if page < 1 {
		page = 1
	}

	sort := strings.TrimSpace(req.Sort)
	if sort == "" {
		sort = req.DefaultSort
	}
	if sort == "" {
		sort = "created_at"
	}
	if _, ok := req.AllowedSort[sort]; !ok {
		return Params{}, errors.Join(apperror.ErrBadRequest, ErrSortNotAllowed)
	}

	order := strings.ToLower(strings.TrimSpace(req.Order))
	if order != "asc" {
		order = "desc"
	}

	query := strings.TrimSpace(req.Query)
	if len(query) > MaxQueryLength {
		query = query[:MaxQueryLength]
	}

	cursor, err := DecodeCursor(req.Cursor)
	if err != nil {
		return Params{}, err
	}
	if cursor.Present() {
		// Kursor membawa pengurutan aslinya; memakainya dengan urutan lain
		// menghasilkan halaman yang tidak bermakna.
		if cursor.Sort != sort || cursor.Order != order {
			return Params{}, errors.Join(apperror.ErrBadRequest, ErrCursorMismatch)
		}
		req.CursorMode = true
	}

	from, to, err := resolveRange(req, now)
	if err != nil {
		return Params{}, err
	}

	return Params{
		Page:       page,
		PerPage:    perPage,
		Query:      query,
		Cursor:     cursor,
		Sort:       sort,
		Order:      order,
		From:       from,
		To:         to,
		CursorMode: req.CursorMode,
	}, nil
}

// resolveRange mengubah tanggal WIB menjadi batas UTC.
//
// Hari akuntansi memakai WIB (plans/22), jadi "from=2026-11-05" berarti
// 00:00 WIB hari itu, bukan 00:00 UTC.
func resolveRange(req Request, now time.Time) (*time.Time, *time.Time, error) {
	fromText := strings.TrimSpace(req.From)
	toText := strings.TrimSpace(req.To)

	if fromText == "" && toText == "" {
		if !req.RequireRange {
			return nil, nil, nil
		}
		days := req.DefaultRangeDays
		if days <= 0 {
			days = 7
		}
		start := startOfDayWIB(now.AddDate(0, 0, -(days - 1)))
		return &start, nil, nil
	}

	var from, to *time.Time
	if fromText != "" {
		day, err := parseDayWIB(fromText)
		if err != nil {
			return nil, nil, err
		}
		from = &day
	}
	if toText != "" {
		day, err := parseDayWIB(toText)
		if err != nil {
			return nil, nil, err
		}
		// Hari akhir inklusif: batasnya awal hari berikutnya.
		end := day.AddDate(0, 0, 1)
		to = &end
	}

	if from != nil && to != nil {
		span := to.Sub(*from)
		if span <= 0 {
			return nil, nil, validation("tanggal akhir harus setelah tanggal awal")
		}
		if span > MaxRangeDays*24*time.Hour {
			return nil, nil, errors.Join(apperror.ErrBadRequest, ErrRangeTooLong)
		}
	}
	return from, to, nil
}

// parseDayWIB membaca tanggal YYYY-MM-DD sebagai 00:00 WIB lalu ke UTC.
func parseDayWIB(text string) (time.Time, error) {
	day, err := time.ParseInLocation("2006-01-02", text, timezone.WIB)
	if err != nil {
		return time.Time{}, validation("format tanggal harus YYYY-MM-DD")
	}
	return day.UTC(), nil
}

func startOfDayWIB(t time.Time) time.Time {
	local := t.In(timezone.WIB)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, timezone.WIB).UTC()
}

// ParseStatusFilter membaca "active,banned" menjadi daftar yang sudah divalidasi.
// Nilai di luar daftar putih ditolak agar salah ketik tidak diam-diam
// menampilkan semua baris (plans/24 §4.1).
func ParseStatusFilter(raw string, allowed []string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	permitted := make(map[string]struct{}, len(allowed))
	for _, item := range allowed {
		permitted[item] = struct{}{}
	}

	seen := make(map[string]struct{})
	out := make([]string, 0, 4)
	for _, part := range strings.Split(raw, ",") {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		if _, ok := permitted[value]; !ok {
			return nil, validation("nilai filter tidak dikenal: " + value)
		}
		if _, dup := seen[value]; dup {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out, nil
}

// EscapeLike menetralkan wildcard pada pencarian agar "%" dicari apa adanya.
func EscapeLike(text string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(text)
}

// Meta adalah ringkasan daftar yang dikirim ke klien.
type Meta struct {
	Page        int    `json:"page"`
	PerPage     int    `json:"per_page"`
	Total       int64  `json:"total"`
	TotalPages  int    `json:"total_pages"`
	HasMore     bool   `json:"has_more"`
	Sort        string `json:"sort"`
	Order       string `json:"order"`
	ApproxTotal bool   `json:"approx_total"`
	NextCursor  string `json:"next_cursor,omitempty"`
}

// NewMeta menyusun metadata untuk daftar berbasis halaman.
func NewMeta(p Params, total int64) Meta {
	pages := 0
	if p.PerPage > 0 {
		pages = int((total + int64(p.PerPage) - 1) / int64(p.PerPage))
	}
	return Meta{
		Page:       p.Page,
		PerPage:    p.PerPage,
		Total:      total,
		TotalPages: pages,
		HasMore:    int64(p.Page*p.PerPage) < total,
		Sort:       p.Sort,
		Order:      p.Order,
	}
}

// NewCursorMeta menyusun metadata untuk daftar berbasis kursor.
// Total hanya perkiraan sehingga ditandai agar antarmuka menampilkan "~".
func NewCursorMeta(p Params, approxTotal int64, next Cursor) Meta {
	return Meta{
		Page:        1,
		PerPage:     p.PerPage,
		Total:       approxTotal,
		TotalPages:  0,
		HasMore:     next.Present(),
		Sort:        p.Sort,
		Order:       p.Order,
		ApproxTotal: true,
		NextCursor:  next.Encode(),
	}
}

// BigTableThreshold adalah batas kapan COUNT(*) diganti perkiraan.
const BigTableThreshold = 100000

// NumOf memformat angka tanpa pemisah ribuan; dipakai kolom kursor numerik.
func NumOf(value int64) string {
	return strconv.FormatInt(value, 10)
}

// createdAtCursor membuat kursor dari nilai waktu dan id baris terakhir.
func createdAtCursor(t time.Time, id, sort, order string) Cursor {
	return Cursor{
		Value: t.UTC().Format(time.RFC3339Nano),
		ID:    id,
		Sort:  sort,
		Order: order,
	}
}

// CursorOfCreatedAt mengekspor pembentuk kursor waktu untuk repository.
func CursorOfCreatedAt(t time.Time, id, sort, order string) Cursor {
	return createdAtCursor(t, id, sort, order)
}

// validation membungkus pesan menjadi BAD_REQUEST.
func validation(message string) error {
	return errors.Join(apperror.ErrBadRequest, errors.New(message))
}
