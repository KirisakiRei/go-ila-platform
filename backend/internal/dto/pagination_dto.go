package dto

// Batas paginasi standar. Nilai ini mengikat semua daftar: tanpa batas atas,
// satu permintaan bisa menarik seluruh tabel ke memori (plans/24 §2.1).
const (
	DefaultPerPage = 50
	MaxPerPage     = 200
)

// PaginationQuery adalah parameter query standar untuk semua endpoint listing.
type PaginationQuery struct {
	Page    int    `form:"page,default=1" binding:"omitempty,min=1"`
	PerPage int    `form:"per_page,default=50" binding:"omitempty,min=1,max=200"`
	Query   string `form:"q" binding:"omitempty,max=100"`
	Cursor  string `form:"cursor" binding:"omitempty,max=512"`
	SortBy  string `form:"sort" binding:"omitempty,max=40"`
	Order   string `form:"order,default=desc" binding:"omitempty,oneof=asc desc"`
	From    string `form:"from" binding:"omitempty"`
	To      string `form:"to" binding:"omitempty"`
}

// Normalized mengembalikan salinan dengan batas yang sudah dipaksa.
// Dipakai handler agar semua endpoint memakai aturan yang sama.
func (q PaginationQuery) Normalized() PaginationQuery {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PerPage < 1 {
		q.PerPage = DefaultPerPage
	}
	if q.PerPage > MaxPerPage {
		q.PerPage = MaxPerPage
	}
	if q.Order != "asc" && q.Order != "desc" {
		q.Order = "desc"
	}
	return q
}

// Offset menghitung posisi baris pertama untuk query berbasis halaman.
func (q PaginationQuery) Offset() int {
	return (q.Page - 1) * q.PerPage
}

// PaginationMeta adalah metadata pagination yang disertakan pada balasan list.
type PaginationMeta struct {
	CurrentPage int    `json:"current_page"`
	PerPage     int    `json:"per_page"`
	TotalItems  int64  `json:"total_items"`
	TotalPages  int    `json:"total_pages"`
	HasMore     bool   `json:"has_more"`
	Sort        string `json:"sort,omitempty"`
	Order       string `json:"order,omitempty"`
	// ApproxTotal menandai total sebagai perkiraan pada tabel besar, agar
	// antarmuka tidak menampilkan angka tepat yang sebetulnya tebakan.
	ApproxTotal bool `json:"approx_total"`
	// NextCursor diisi hanya pada daftar berbasis kursor.
	NextCursor string `json:"next_cursor,omitempty"`
}

// NewPaginationMeta menyusun metadata dari parameter dan jumlah total.
func NewPaginationMeta(q PaginationQuery, total int64) PaginationMeta {
	pages := 0
	if q.PerPage > 0 {
		pages = int((total + int64(q.PerPage) - 1) / int64(q.PerPage))
	}
	return PaginationMeta{
		CurrentPage: q.Page,
		PerPage:     q.PerPage,
		TotalItems:  total,
		TotalPages:  pages,
		HasMore:     int64(q.Page*q.PerPage) < total,
		Sort:        q.SortBy,
		Order:       q.Order,
	}
}

// SafeSortColumn memastikan kolom pengurutan hanya berasal dari daftar putih.
// Nama kolom dari klien tidak boleh masuk query SQL apa adanya (plans/24 §5).
func SafeSortColumn(requested string, allowed map[string]string, fallback string) string {
	if col, ok := allowed[requested]; ok {
		return col
	}
	return fallback
}
