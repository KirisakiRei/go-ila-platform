package model

// Page adalah satu halaman publik yang isinya dikelola lewat panel admin.
//
// Halaman publik punya kerangka yang tetap (tata letak, urutan bagian, gaya
// tipografi), sedangkan isi tiap bagiannya berubah-ubah. Karena itu yang
// disimpan di sini bukan HTML utuh, melainkan data terstruktur: setiap bagian
// halaman adalah satu baris PageSection, dan setiap butir di dalamnya satu
// baris PageItem. Dengan begitu admin dapat mengubah kalimat tanpa bisa
// merusak tata letak, dan perubahan langsung tampil setelah disimpan.
type Page struct {
	Base
	// Slug adalah penanda halaman: tentu, fitur, harga, bantuan, legal.
	Slug string `gorm:"type:varchar(40);not null;uniqueIndex" json:"slug"`
	// Eyebrow adalah label kecil di atas judul, misalnya "Fitur".
	Eyebrow string `gorm:"type:varchar(80);not null;default:''" json:"eyebrow"`
	// Title adalah judul utama halaman.
	Title string `gorm:"type:varchar(200);not null;default:''" json:"title"`
	// Lead adalah paragraf pembuka di bawah judul.
	Lead string `gorm:"type:text;not null;default:''" json:"lead"`
	// MetaTitle dan MetaDescription dipakai mesin pencari.
	MetaTitle       string `gorm:"type:varchar(200);not null;default:''" json:"meta_title"`
	MetaDescription string `gorm:"type:varchar(320);not null;default:''" json:"meta_description"`
	// IsPublished menandai halaman boleh dibaca publik.
	IsPublished bool   `gorm:"not null;default:true" json:"is_published"`
	UpdatedBy   string `gorm:"type:varchar(36);not null;default:''" json:"updated_by"`
}

// TableName menetapkan nama tabel tanpa bentuk jamak.
func (Page) TableName() string {
	return "page"
}

// Jenis bagian halaman. Setiap jenis punya bentuk render yang sudah ditentukan
// di antarmuka, sehingga isi tidak pernah menentukan tata letak.
const (
	SectionProse    = "prose"    // paragraf bebas dan daftar
	SectionCards    = "cards"    // kartu berikon: judul + ringkasan + batasan
	SectionSteps    = "steps"    // langkah bernomor
	SectionFaq      = "faq"      // tanya-jawab
	SectionTable    = "table"    // tabel klasifikasi
	SectionCallout  = "callout"  // blok peringatan
	SectionFeatures = "features" // daftar kemampuan dengan batasan
	SectionStats    = "stats"    // angka ringkas: label + nilai
)

// KnownSectionKind menandai jenis bagian yang dikenali.
func KnownSectionKind(kind string) bool {
	switch kind {
	case SectionProse, SectionCards, SectionSteps, SectionFaq,
		SectionTable, SectionCallout, SectionFeatures, SectionStats:
		return true
	default:
		return false
	}
}

// PageSection adalah satu bagian terurut di dalam halaman.
//
// `Body` menampung markdown untuk bagian bergaya prosa dan callout. Jenis
// bagian lain memakai Butir di PageItem agar datanya tidak berbentuk teks
// bebas yang sulit disunting dan divalidasi.
type PageSection struct {
	Base
	PageSlug string `gorm:"type:varchar(40);not null;uniqueIndex:idx_page_section_key" json:"page_slug"`
	// Kunci unik per halaman, bukan global: dua halaman boleh sama-sama punya
	// bagian "faq" atau "catatan" tanpa saling bentrok. Inilah sebabnya
	// page_slug ikut menjadi bagian dari indeks unik yang sama.
	Key         string `gorm:"type:varchar(60);not null;uniqueIndex:idx_page_section_key" json:"key"`
	Kind        string `gorm:"type:varchar(20);not null" json:"kind"`
	Eyebrow     string `gorm:"type:varchar(80);not null;default:''" json:"eyebrow"`
	Title       string `gorm:"type:varchar(200);not null;default:''" json:"title"`
	Lead        string `gorm:"type:text;not null;default:''" json:"lead"`
	Body        string `gorm:"type:text;not null;default:''" json:"body"`
	Layout      string `gorm:"type:varchar(20);not null;default:''" json:"layout"`
	SortOrder   int    `gorm:"not null;default:0" json:"sort_order"`
	IsPublished bool   `gorm:"not null;default:true" json:"is_published"`
}

// TableName menetapkan nama tabel tanpa bentuk jamak.
func (PageSection) TableName() string {
	return "page_section"
}

// PageItem adalah satu butir di dalam bagian: satu kartu, satu langkah, satu
// pertanyaan, atau satu baris tabel.
type PageItem struct {
	Base
	SectionID string `gorm:"type:varchar(36);not null;index" json:"section_id"`
	// SectionKey bukan kolom: dipakai service untuk menautkan butir ke
	// bagiannya saat disimpan, lalu diisi SectionID oleh repository.
	SectionKey string `gorm:"-" json:"-"`
	// Label adalah judul singkat: nama kartu, pertanyaan FAQ, atau sel pertama.
	Label string `gorm:"type:varchar(255);not null" json:"label"`
	// Value adalah isi butir: ringkasan kartu, jawaban FAQ, atau sel tabel.
	Value string `gorm:"type:text;not null;default:''" json:"value"`
	// Note adalah keterangan tambahan berhuruf kecil, misalnya batasan fitur.
	Note string `gorm:"type:text;not null;default:''" json:"note"`
	// Icon adalah nama ikon dari daftar tertutup yang dikenali antarmuka.
	Icon      string `gorm:"type:varchar(40);not null;default:''" json:"icon"`
	Href      string `gorm:"type:varchar(255);not null;default:''" json:"href"`
	SortOrder int    `gorm:"not null;default:0" json:"sort_order"`
}

// TableName menetapkan nama tabel tanpa bentuk jamak.
func (PageItem) TableName() string {
	return "page_item"
}
