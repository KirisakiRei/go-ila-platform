// Package legal adalah katalog tetap jenis dokumen hukum beserta aturan kapan
// persetujuannya diminta. Satu sumber kebenaran untuk seed, validasi, dan
// pemeriksaan di jalur eksekusi.
package legal

// Slug dokumen yang dikenal sistem.
const (
	SlugTOS     = "tos"
	SlugPrivacy = "privacy"
	SlugRisk    = "risk"
	SlugRefund  = "refund"
	SlugFairUse = "fair_use"
)

// SignupSlugs adalah dokumen yang wajib disetujui saat mendaftar.
// Disetujui bersamaan agar user tidak diminta berulang kali di awal.
func SignupSlugs() []string {
	return []string{SlugTOS, SlugPrivacy, SlugFairUse}
}

// DemoAccountSlugs adalah dokumen wajib sebelum menghubungkan akun broker
// sungguhan. `risk` sengaja tidak diminta saat daftar: sebelum ada akun,
// user belum menanggung risiko finansial (plans/09 §2.2).
func DemoAccountSlugs() []string {
	return []string{SlugRisk}
}

// PaymentSlugs adalah dokumen wajib sebelum pembayaran pertama.
func PaymentSlugs() []string {
	return []string{SlugRefund}
}

// Definition adalah satu jenis dokumen beserta judul manusianya.
type Definition struct {
	Slug  string
	Title string
}

// Catalog memuat setiap jenis dokumen yang dikenali.
var Catalog = []Definition{
	{SlugTOS, "Syarat & Ketentuan"},
	{SlugPrivacy, "Kebijakan Privasi"},
	{SlugRisk, "Disclaimer Risiko Trading"},
	{SlugRefund, "Kebijakan Pengembalian Dana"},
	{SlugFairUse, "Kebijakan Penggunaan Wajar"},
}

// Known menandai slug yang terdaftar.
func Known(slug string) bool {
	for _, doc := range Catalog {
		if doc.Slug == slug {
			return true
		}
	}
	return false
}

// Title mengembalikan judul baku sebuah slug; kosong bila tidak dikenal.
func Title(slug string) string {
	for _, doc := range Catalog {
		if doc.Slug == slug {
			return doc.Title
		}
	}
	return ""
}
