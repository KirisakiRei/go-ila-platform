package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ila/backend/internal/model"
)

/*
Pembaca ringkasan sinyal publik.

Yang dijaga:
  - Nama yang dikenal pengunjung (NAS100, GOLD) menemukan simbol kanoniknya.
  - Sinyal yang dibatalkan, ditolak, atau dilewati tidak pernah diambil: di
    halaman publik sinyal seperti itu terbaca sebagai arah pasar yang sah.
  - Yang diambil hanya sinyal terbaru per simbol dan hanya yang masih segar.
*/

// newPublicSignalTestDB menyiapkan basis data sementara berisi tabel katalog
// dan sinyal. Berkas dipakai agar tidak berbagi keadaan dengan tes lain.
func newPublicSignalTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := filepath.ToSlash(filepath.Join(t.TempDir(), "public.db"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("membuka basis data uji: %v", err)
	}
	if err := db.AutoMigrate(&model.PairCategory{}, &model.TradingPair{}, &model.Signal{}); err != nil {
		t.Fatalf("menyiapkan skema: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("mengambil koneksi: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	kategori := []model.PairCategory{
		{Base: model.Base{ID: "cat-forex"}, Key: "forex", Label: "Forex", IsEnabled: true},
		{Base: model.Base{ID: "cat-indices"}, Key: "indices", Label: "Indices", IsEnabled: true},
		{Base: model.Base{ID: "cat-commodity"}, Key: "commodity", Label: "Commodity", IsEnabled: true},
		// Kategori yang dimatikan admin: isinya tidak boleh tampil.
		{Base: model.Base{ID: "cat-crypto"}, Key: "crypto", Label: "Crypto", IsEnabled: false},
	}
	if err := db.Create(&kategori).Error; err != nil {
		t.Fatalf("membuat kategori: %v", err)
	}

	pair := []model.TradingPair{
		{Base: model.Base{ID: "p-xau"}, Symbol: "XAUUSD", DisplayName: "Gold vs US Dollar", CategoryID: "cat-commodity", Aliases: "GOLD,XAU/USD", IsEnabled: true},
		{Base: model.Base{ID: "p-eur"}, Symbol: "EURUSD", DisplayName: "Euro vs US Dollar", CategoryID: "cat-forex", Aliases: "EUR/USD", IsEnabled: true},
		// NAS100 dan US100 adalah alias; kunci kanonik engine adalah USTEC.
		{Base: model.Base{ID: "p-ustec"}, Symbol: "USTEC", DisplayName: "US Tech 100", CategoryID: "cat-indices", Aliases: "NAS100,US100,NDX", IsEnabled: true},
		// Instrumen yang dimatikan admin tidak boleh muncul lagi, walaupun
		// kategorinya sendiri masih aktif.
		{Base: model.Base{ID: "p-btc"}, Symbol: "BTCUSD", DisplayName: "Bitcoin vs US Dollar", CategoryID: "cat-forex", Aliases: "BTC/USD", IsEnabled: false},
		// Instrumen aktif di kategori yang dimatikan juga tidak boleh tampil.
		{Base: model.Base{ID: "p-eth"}, Symbol: "ETHUSD", DisplayName: "Ethereum vs US Dollar", CategoryID: "cat-crypto", Aliases: "ETH/USD", IsEnabled: true},
	}
	if err := db.Create(&pair).Error; err != nil {
		t.Fatalf("membuat pair: %v", err)
	}

	// Kolom is_enabled bertipe bool dengan nilai bawaan true, sehingga GORM
	// memperlakukan false sebagai "tidak diisi" saat Create. Nonaktifkan secara
	// eksplisit agar keadaan uji benar-benar seperti data yang dimatikan admin.
	if err := db.Model(&model.TradingPair{}).Where("id = ?", "p-btc").
		Update("is_enabled", false).Error; err != nil {
		t.Fatalf("menonaktifkan pair: %v", err)
	}
	if err := db.Model(&model.PairCategory{}).Where("id = ?", "cat-crypto").
		Update("is_enabled", false).Error; err != nil {
		t.Fatalf("menonaktifkan kategori: %v", err)
	}
	return db
}

func TestResolveSymbolsMengenaliAlias(t *testing.T) {
	db := newPublicSignalTestDB(t)
	store := NewPublicSignalStore(db)

	refs, err := store.ResolveSymbols(context.Background(), []string{"NAS100", "GOLD", "EURUSD"})
	if err != nil {
		t.Fatalf("ResolveSymbols error: %v", err)
	}
	if len(refs) != 3 {
		t.Fatalf("refs = %d; ingin 3", len(refs))
	}

	harapan := map[string]string{
		"NAS100": "USTEC",
		"GOLD":   "XAUUSD",
		"EURUSD": "EURUSD",
	}
	for i, ref := range refs {
		masukan := []string{"NAS100", "GOLD", "EURUSD"}[i]
		if ref.Symbol != harapan[masukan] {
			t.Errorf("%s -> %s; ingin %s", masukan, ref.Symbol, harapan[masukan])
		}
	}

	// Kategori ikut agar kartu bisa memberi label yang benar.
	if refs[0].CategoryKey != "indices" {
		t.Errorf("kategori USTEC = %q; ingin indices", refs[0].CategoryKey)
	}
	if refs[1].CategoryKey != "commodity" {
		t.Errorf("kategori XAUUSD = %q; ingin commodity", refs[1].CategoryKey)
	}
}

func TestResolveSymbolsMengabaikanInstrumenNonaktif(t *testing.T) {
	db := newPublicSignalTestDB(t)
	store := NewPublicSignalStore(db)

	refs, err := store.ResolveSymbols(context.Background(), []string{"BTCUSD", "BTC/USD", "ETHUSD"})
	if err != nil {
		t.Fatalf("ResolveSymbols error: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("instrumen nonaktif ikut terbaca: %+v", refs)
	}
}

// Sinyal yang dibatalkan atau ditolak tidak boleh sampai ke halaman publik.
func TestPreviewForSymbolsMengabaikanStatusBatal(t *testing.T) {
	db := newPublicSignalTestDB(t)
	store := NewPublicSignalStore(db)

	sekarang := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	signals := []model.Signal{
		{SignalID: "s-batal", Symbol: "XAUUSD", Direction: "BULLISH", Status: model.SignalStatusCancelled, PublishedAt: sekarang.Add(-time.Hour)},
		{SignalID: "s-tolak", Symbol: "EURUSD", Direction: "BEARISH", Status: model.SignalStatusRejected, PublishedAt: sekarang.Add(-time.Hour)},
		{SignalID: "s-lewat", Symbol: "USTEC", Direction: "BULLISH", Status: model.SignalStatusSkipped, PublishedAt: sekarang.Add(-time.Hour)},
	}
	for i := range signals {
		signals[i].Htf, signals[i].Ltf = "H4", "M15"
		if err := db.Create(&signals[i]).Error; err != nil {
			t.Fatalf("membuat sinyal: %v", err)
		}
	}

	rows, err := store.PreviewForSymbols(context.Background(), []string{"XAUUSD", "EURUSD", "USTEC"}, sekarang.Add(-7*24*time.Hour))
	if err != nil {
		t.Fatalf("PreviewForSymbols error: %v", err)
	}
	if len(rows) != 0 {
		for _, row := range rows {
			t.Errorf("sinyal %q berstatus %s ikut terambil", row.Symbol, row.Status)
		}
	}
}

// Hanya sinyal terbaru per simbol yang dipakai, dan sinyal lama diabaikan.
func TestPreviewForSymbolsMengambilTerbaruPerSimbol(t *testing.T) {
	db := newPublicSignalTestDB(t)
	store := NewPublicSignalStore(db)

	sekarang := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	signals := []model.Signal{
		{SignalID: "x-lama", Symbol: "XAUUSD", Direction: "BEARISH", Status: model.SignalStatusExpired, PublishedAt: sekarang.Add(-3 * time.Hour)},
		{SignalID: "x-baru", Symbol: "XAUUSD", Direction: "BULLISH", Status: model.SignalStatusOpportunity, PublishedAt: sekarang.Add(-30 * time.Minute)},
		{SignalID: "e-baru", Symbol: "EURUSD", Direction: "BEARISH", Status: model.SignalStatusReadyToEntry, PublishedAt: sekarang.Add(-10 * time.Minute)},
		// Terlalu tua untuk ringkasan halaman depan.
		{SignalID: "tua", Symbol: "USTEC", Direction: "BULLISH", Status: model.SignalStatusExpired, PublishedAt: sekarang.Add(-30 * 24 * time.Hour)},
	}
	for i := range signals {
		signals[i].Htf, signals[i].Ltf = "H4", "M15"
		if err := db.Create(&signals[i]).Error; err != nil {
			t.Fatalf("membuat sinyal: %v", err)
		}
	}

	rows, err := store.PreviewForSymbols(context.Background(), []string{"XAUUSD", "EURUSD", "USTEC"}, sekarang.Add(-7*24*time.Hour))
	if err != nil {
		t.Fatalf("PreviewForSymbols error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d; ingin 2 (XAUUSD terbaru dan EURUSD)", len(rows))
	}

	perSimbol := make(map[string]model.Signal, len(rows))
	for _, row := range rows {
		perSimbol[row.Symbol] = row
	}
	if got := perSimbol["XAUUSD"].SignalID; got != "x-baru" {
		t.Errorf("XAUUSD memakai sinyal %q; ingin x-baru", got)
	}
	if got := perSimbol["EURUSD"].Direction; got != "BEARISH" {
		t.Errorf("EURUSD arah = %q; ingin BEARISH", got)
	}
	if _, ada := perSimbol["USTEC"]; ada {
		t.Error("sinyal berusia 30 hari masih ikut ringkasan")
	}
}

// Tanpa simbol, kueri tidak dijalankan sama sekali.
func TestPreviewForSymbolsDaftarKosong(t *testing.T) {
	db := newPublicSignalTestDB(t)
	store := NewPublicSignalStore(db)

	rows, err := store.PreviewForSymbols(context.Background(), nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("PreviewForSymbols error: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %d; ingin 0", len(rows))
	}
}
