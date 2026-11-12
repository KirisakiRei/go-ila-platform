package model

// Kategori laporan bug. Daftar tertutup supaya admin bisa menyaring dengan
// andal; teks bebas membuat pelaporan tidak bisa direkap.
const (
	BugKategoriTampilan   = "tampilan"
	BugKategoriSinyal     = "sinyal"
	BugKategoriEksekusi   = "eksekusi"
	BugKategoriAkunBroker = "akun-broker"
	BugKategoriLogin      = "login"
	BugKategoriKinerja    = "kinerja"
	BugKategoriLainnya    = "lainnya"
)

// BugKategori adalah daftar sah untuk divalidasi service.
var BugKategori = []string{
	BugKategoriTampilan, BugKategoriSinyal, BugKategoriEksekusi,
	BugKategoriAkunBroker, BugKategoriLogin, BugKategoriKinerja, BugKategoriLainnya,
}

// Status laporan bug.
const (
	BugStatusBaru     = "baru"
	BugStatusDitinjau = "ditinjau"
	BugStatusSelesai  = "selesai"
	BugStatusDitolak  = "ditolak"
)

// BugStatusSah adalah daftar status yang boleh dituju admin.
var BugStatusSah = []string{BugStatusBaru, BugStatusDitinjau, BugStatusSelesai, BugStatusDitolak}

// BugReport adalah laporan yang dikirim pengguna ke panel admin.
type BugReport struct {
	Base
	UserID string `gorm:"type:varchar(36);index;not null" json:"user_id"`
	// Category dibatasi BugKategori; divalidasi di service, bukan di DB, agar
	// menambah kategori baru tidak menuntut migrasi.
	Category string `gorm:"type:varchar(24);not null" json:"category"`
	Title    string `gorm:"type:varchar(120);not null" json:"title"`
	Body     string `gorm:"type:text;not null" json:"body"`
	// PageURL dan UserAgent diisi otomatis klien; membantu mereproduksi.
	PageURL   string `gorm:"type:varchar(255);not null;default:''" json:"page_url"`
	UserAgent string `gorm:"type:varchar(255);not null;default:''" json:"user_agent"`
	Status    string `gorm:"type:varchar(16);not null;default:'baru'" json:"status"`
	AdminNote string `gorm:"type:text;not null;default:''" json:"admin_note"`
}

// TableName menetapkan nama tabel.
func (BugReport) TableName() string {
	return "bug_report"
}

// BugReportAttachment adalah lampiran satu laporan.
//
// FileName hanya untuk tampilan dan unduhan; berkas disimpan memakai StoredAs
// (UUID) sehingga nama dari klien tidak pernah menyentuh jalur berkas.
type BugReportAttachment struct {
	Base
	ReportID  string `gorm:"type:varchar(36);index;not null" json:"report_id"`
	FileName  string `gorm:"type:varchar(160);not null" json:"file_name"`
	MimeType  string `gorm:"type:varchar(80);not null" json:"mime_type"`
	SizeBytes int64  `gorm:"not null" json:"size_bytes"`
	StoredAs  string `gorm:"type:varchar(160);not null" json:"-"`
}

// TableName menetapkan nama tabel.
func (BugReportAttachment) TableName() string {
	return "bug_report_attachment"
}
