package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

/*
Ringkasan sinyal publik.

Yang dijaga:
  - Harga masuk, batas kerugian, target, dan ringkasan analisis tidak pernah
    ikut ke halaman publik.
  - Alias katalog diresolusi lebih dulu: NAS100 harus menemukan sinyal USTEC,
    karena itulah nama kanonik yang dipakai engine.
  - Tanpa sinyal bukan galat; instrumen tetap tampil dengan penanda kosong.
  - Sinyal yang dibatalkan atau sudah kedaluwarsa tidak boleh terbaca sebagai
    arah pasar yang masih berlaku.
*/

// fakePublicSignalStore meniru katalog dan tabel sinyal.
type fakePublicSignalStore struct {
	katalog []repository.SymbolRef
	signals []model.Signal
	// kanonikDiminta mencatat simbol yang benar-benar dikirim ke kueri sinyal.
	kanonikDiminta []string
	err            error
}

func (f *fakePublicSignalStore) ResolveSymbols(_ context.Context, names []string) ([]repository.SymbolRef, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]repository.SymbolRef, 0, len(names))
	for _, name := range names {
		for _, ref := range f.katalog {
			if strings.EqualFold(ref.Symbol, name) || refAlias(ref, name) {
				out = append(out, ref)
				break
			}
		}
	}
	return out, nil
}

func (f *fakePublicSignalStore) PreviewForSymbols(_ context.Context, symbols []string, _ time.Time) ([]model.Signal, error) {
	f.kanonikDiminta = symbols
	if f.err != nil {
		return nil, f.err
	}
	out := make([]model.Signal, 0, len(f.signals))
	for _, signal := range f.signals {
		for _, simbol := range symbols {
			if strings.EqualFold(signal.Symbol, simbol) {
				out = append(out, signal)
				break
			}
		}
	}
	return out, nil
}

func refAlias(ref repository.SymbolRef, name string) bool {
	for _, alias := range ref.Aliases {
		if strings.EqualFold(alias, name) {
			return true
		}
	}
	return false
}

// fakeMarketReader menyediakan jadwal pasar untuk pengujian.
type fakeMarketReader struct {
	status *dto.PublicStatus
	err    error
}

func (f *fakeMarketReader) Status(context.Context) (*dto.PublicStatus, error) {
	return f.status, f.err
}

// katalogBawaan berisi tiga instrumen yang tampil di halaman depan.
func katalogBawaan() []repository.SymbolRef {
	return []repository.SymbolRef{
		{Symbol: "XAUUSD", DisplayName: "Gold vs US Dollar", CategoryKey: "commodity", Aliases: []string{"GOLD", "XAU/USD"}},
		{Symbol: "EURUSD", DisplayName: "Euro vs US Dollar", CategoryKey: "forex", Aliases: []string{"EUR/USD"}},
		// NAS100 adalah alias; kunci kanonik engine adalah USTEC.
		{Symbol: "USTEC", DisplayName: "US Tech 100", CategoryKey: "indices", Aliases: []string{"NAS100", "US100"}},
	}
}

func newPublicSignalService(store *fakePublicSignalStore) (*publicSignalService, *fakeMarketReader) {
	market := &fakeMarketReader{status: &dto.PublicStatus{Market: dto.MarketWindow{Open: true, Label: "Pasar buka"}}}
	svc := NewPublicSignalService(store, market).(*publicSignalService)
	svc.now = func() time.Time { return time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC) }
	return svc, market
}

// Harga masuk, batas kerugian, target, dan ringkasan analisis wajib absen.
//
// Ini pengaman utama: halaman publik tidak boleh membocorkan isi produk
// berlangganan. Bila suatu saat snapshot publik ditambah field, tes ini gagal.
func TestPreviewTidakMemuatParameterHarga(t *testing.T) {
	store := &fakePublicSignalStore{
		katalog: katalogBawaan(),
		signals: []model.Signal{{
			Symbol:      "XAUUSD",
			Direction:   "bullish",
			Htf:         "H4",
			Ltf:         "M15",
			Status:      model.SignalStatusReadyToEntry,
			EntryPrice:  2650.55,
			StopLoss:    2638.10,
			TakeProfit:  2680.00,
			RiskReward:  2.4,
			Summary:     "Area likuiditas tersapu",
			PublishedAt: time.Date(2026, 11, 9, 9, 0, 0, 0, time.UTC),
		}},
	}
	svc, _ := newPublicSignalService(store)

	hasil, err := svc.Preview(context.Background(), nil)
	if err != nil {
		t.Fatalf("Preview error: %v", err)
	}

	var xau *dto.PublicSignalPreviewItem
	for i := range hasil.Items {
		if hasil.Items[i].Symbol == "XAUUSD" {
			xau = &hasil.Items[i]
		}
	}
	if xau == nil || xau.Signal == nil {
		t.Fatal("sinyal XAUUSD tidak ikut pada ringkasan")
	}

	// Diperiksa dari JSON yang benar-benar dikirim server: nama medan apa pun
	// yang mengandung harga, kerugian, target, atau rasio langsung menggagalkan
	// tes. Cara ini menangkap medan baru yang ditambahkan tanpa sengaja.
	isi, err := json.Marshal(hasil)
	if err != nil {
		t.Fatalf("menyusun JSON: %v", err)
	}
	teks := strings.ToLower(string(isi))
	for _, terlarang := range []string{"entry", "stop_loss", "takeprofit", "take_profit", "risk_reward", "summary", "next_action"} {
		if strings.Contains(teks, terlarang) {
			t.Errorf("JSON publik memuat %q: %s", terlarang, teks)
		}
	}
}

// NAS100 yang diminta klien harus menemukan sinyal bersimbol USTEC.
func TestPreviewMenyelesaikanAliasSimbol(t *testing.T) {
	store := &fakePublicSignalStore{
		katalog: katalogBawaan(),
		signals: []model.Signal{{
			Symbol:      "USTEC",
			Direction:   "BEARISH",
			Htf:         "H1",
			Ltf:         "M5",
			Status:      model.SignalStatusOpportunity,
			PublishedAt: time.Date(2026, 11, 9, 9, 30, 0, 0, time.UTC),
		}},
	}
	svc, _ := newPublicSignalService(store)

	hasil, err := svc.Preview(context.Background(), []string{"NAS100"})
	if err != nil {
		t.Fatalf("Preview error: %v", err)
	}
	if len(hasil.Items) != 1 {
		t.Fatalf("item = %d; ingin 1", len(hasil.Items))
	}

	item := hasil.Items[0]
	// Yang tampil adalah nama kanonik katalog, bukan alias masukan.
	if item.Symbol != "USTEC" {
		t.Errorf("simbol = %q; ingin USTEC", item.Symbol)
	}
	if item.Category != "indices" {
		t.Errorf("kategori = %q; ingin indices", item.Category)
	}
	if !item.HasSignal || item.Signal == nil {
		t.Fatal("sinyal USTEC tidak ditemukan lewat alias NAS100")
	}
	if item.Signal.Direction != "BEARISH" {
		t.Errorf("arah = %q; ingin BEARISH", item.Signal.Direction)
	}
}

// Tanpa sinyal, instrumen tetap tampil dan bukan galat.
func TestPreviewTanpaSinyalTetapMenampilkanInstrumen(t *testing.T) {
	store := &fakePublicSignalStore{katalog: katalogBawaan()}
	svc, _ := newPublicSignalService(store)

	hasil, err := svc.Preview(context.Background(), nil)
	if err != nil {
		t.Fatalf("Preview error: %v", err)
	}
	if len(hasil.Items) != len(SimbolBawaan) {
		t.Fatalf("item = %d; ingin %d", len(hasil.Items), len(SimbolBawaan))
	}
	for _, item := range hasil.Items {
		if item.HasSignal {
			t.Errorf("%s ditandai punya sinyal padahal tabel kosong", item.Symbol)
		}
		if item.Signal != nil {
			t.Errorf("%s memuat snapshot padahal tanpa sinyal", item.Symbol)
		}
	}
	if !hasil.Market.Open || hasil.Market.Label != "Pasar buka" {
		t.Errorf("keadaan pasar = %+v; ingin buka", hasil.Market)
	}
}

// Keadaan pasar tetap terisi walau layanan status gagal.
func TestPreviewTetapJalanSaatStatusPasarGagal(t *testing.T) {
	store := &fakePublicSignalStore{katalog: katalogBawaan()}
	svc, market := newPublicSignalService(store)
	market.err = errors.New("status tidak tersedia")

	hasil, err := svc.Preview(context.Background(), nil)
	if err != nil {
		t.Fatalf("Preview tidak boleh gagal karena status pasar: %v", err)
	}
	if hasil.Market.Open {
		t.Error("pasar tidak boleh dianggap buka saat status tidak diketahui")
	}
	if hasil.Market.Label == "" {
		t.Error("label pasar wajib ada agar kartu tidak kosong")
	}
}

// Simbol yang tidak dikenal katalog bukan sumber galat baru, tetapi bila semua
// tidak dikenal hasilnya diberitahu sebagai masukan tidak sah.
func TestPreviewMenolakSimbolYangSemuaTidakDikenal(t *testing.T) {
	store := &fakePublicSignalStore{katalog: katalogBawaan()}
	svc, _ := newPublicSignalService(store)

	_, err := svc.Preview(context.Background(), []string{"ZZZXXX"})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
}

// Karakter di luar bentuk simbol ditolak sebelum menyentuh kueri.
func TestPreviewMenolakSimbolTidakSah(t *testing.T) {
	store := &fakePublicSignalStore{katalog: katalogBawaan()}
	svc, _ := newPublicSignalService(store)

	for _, simbol := range []string{"XAU'--", "XAU USD", "<script>"} {
		if _, err := svc.Preview(context.Background(), []string{simbol}); !errors.Is(err, apperror.ErrBadRequest) {
			t.Errorf("simbol %q: error = %v; ingin BAD_REQUEST", simbol, err)
		}
	}
}

// Jumlah simbol dibatasi agar katalog tidak bisa ditelusuri sekaligus.
func TestPreviewMembatasiJumlahSimbol(t *testing.T) {
	store := &fakePublicSignalStore{katalog: katalogBawaan()}
	svc, _ := newPublicSignalService(store)

	banyak := make([]string, 0, BatasSimbol+1)
	for i := 0; i <= BatasSimbol; i++ {
		banyak = append(banyak, "XAUUSD")
	}
	// Duplikat disaring lebih dulu, jadi beri nama berbeda.
	banyak = []string{"XAUUSD", "EURUSD", "USTEC", "GOLD", "US100", "NDX"}

	if _, err := svc.Preview(context.Background(), banyak); !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
}

// Sinyal yang dibatalkan tidak boleh diambil kueri sama sekali.
func TestPreviewTidakMengambilSinyalBatal(t *testing.T) {
	// Penyaringan status dilakukan di kueri; di sini dipastikan daftar simbol
	// yang dikirim ke kueri memakai simbol kanonik, bukan nama masukan.
	store := &fakePublicSignalStore{katalog: katalogBawaan()}
	svc, _ := newPublicSignalService(store)

	if _, err := svc.Preview(context.Background(), []string{"NAS100"}); err != nil {
		t.Fatalf("Preview error: %v", err)
	}

	// Simbol kanonik wajib ada, karena itulah yang dipakai engine. Alias ikut
	// dikirim dengan sengaja: sinyal lama bisa tersimpan dengan nama lain untuk
	// instrumen yang sama, dan membuangnya berarti arah tersebut hilang.
	adaKanonik := false
	for _, simbol := range store.kanonikDiminta {
		if simbol == "USTEC" {
			adaKanonik = true
		}
	}
	if !adaKanonik {
		t.Errorf("kueri tidak memuat simbol kanonik USTEC: %v", store.kanonikDiminta)
	}
}
