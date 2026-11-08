// Package pagecontent menyimpan isi awal halaman publik.
//
// Isi ini hanya dipakai satu kali saat migrasi: setelah tersimpan di basis
// data, seluruhnya dikelola admin lewat panel Konten Halaman. Karena itu teks
// di sini tidak boleh dirujuk langsung oleh antarmuka — antarmuka selalu
// membaca dari API.
package pagecontent

// Seed adalah satu halaman awal beserta bagiannya.
type Seed struct {
	Slug            string
	Eyebrow         string
	Title           string
	Lead            string
	MetaTitle       string
	MetaDescription string
	Sections        []Section
}

// Section adalah satu bagian halaman awal.
type Section struct {
	Key     string
	Kind    string
	Eyebrow string
	Title   string
	Lead    string
	Body    string
	Layout  string
	Items   []Item
}

// Item adalah satu butir bagian.
type Item struct {
	Label string
	Value string
	Note  string
	Icon  string
	Href  string
}

// Kind yang dipakai isi awal. Nilainya sama dengan yang dikenali model.
const (
	KindProse    = "prose"
	KindCards    = "cards"
	KindSteps    = "steps"
	KindFaq      = "faq"
	KindTable    = "table"
	KindCallout  = "callout"
	KindFeatures = "features"
	KindStats    = "stats"
)

// Slugs adalah halaman yang isinya dikelola lewat panel admin.
var Slugs = []string{"fitur", "harga", "tentang", "bantuan", "legal"}
