package model_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ila/backend/internal/model"
)

// Kunci bagian hanya wajib unik di dalam satu halaman.
//
// Dua halaman memang memakai kunci yang sama — `faq`, `catatan`, `kontak` —
// karena setiap halaman punya bagiannya sendiri. Indeks unik yang hanya
// memuat kolom `key` pernah membuat seed berhenti di halaman kedua, jadi
// perilaku ini dijaga oleh uji.
func TestKunciBagianUnikPerHalaman(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("membuka sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Page{}, &model.PageSection{}, &model.PageItem{}); err != nil {
		t.Fatalf("migrasi gagal: %v", err)
	}

	pertama := &model.PageSection{PageSlug: "fitur", Key: "faq", Kind: model.SectionFaq}
	if err := db.Create(pertama).Error; err != nil {
		t.Fatalf("bagian pertama harus tersimpan: %v", err)
	}

	// Kunci sama di halaman berbeda: sah.
	kedua := &model.PageSection{PageSlug: "harga", Key: "faq", Kind: model.SectionFaq}
	if err := db.Create(kedua).Error; err != nil {
		t.Fatalf("kunci sama di halaman lain harus sah: %v", err)
	}

	// Kunci sama di halaman yang sama: ditolak.
	ganda := &model.PageSection{PageSlug: "fitur", Key: "faq", Kind: model.SectionProse}
	if err := db.Create(ganda).Error; err == nil {
		t.Fatal("kunci ganda dalam satu halaman seharusnya ditolak")
	}
}
