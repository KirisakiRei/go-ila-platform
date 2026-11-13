package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
)

// PublicSignalStore membaca data katalog dan sinyal untuk halaman publik.
//
// Pembacaannya sengaja sempit: hanya kolom yang memang boleh tampil ke
// pengunjung. Tidak ada metode di sini yang mengembalikan harga masuk, batas
// kerugian, atau target.
type PublicSignalStore interface {
	// PreviewForSymbols mengembalikan sinyal terakhir per simbol yang diminta.
	// `since` membatasi usia sinyal agar ringkasan tidak menampilkan arah yang
	// sudah lama tidak relevan.
	PreviewForSymbols(ctx context.Context, symbols []string, since time.Time) ([]model.Signal, error)
	// ResolveSymbols memetakan nama yang dikirim klien ke simbol kanonik
	// beserta nama tampilan dan kategori katalognya.
	ResolveSymbols(ctx context.Context, names []string) ([]SymbolRef, error)
}

// SymbolRef adalah satu instrumen yang dikenali katalog.
type SymbolRef struct {
	// Symbol adalah kunci kanonik yang dipakai engine, mis. USTEC.
	Symbol      string
	DisplayName string
	CategoryKey string
	// Aliases adalah nama lain yang diterima, sudah dipisah dan dibersihkan.
	Aliases []string
}

type publicSignalStore struct {
	db *gorm.DB
}

// NewPublicSignalStore membuat pembaca ringkasan sinyal publik.
func NewPublicSignalStore(db *gorm.DB) PublicSignalStore {
	return &publicSignalStore{db: db}
}

// ResolveSymbols mencari instrumen menurut simbol kanonik ATAU aliasnya.
//
// Ini penting karena nama yang dikenal pengunjung belum tentu sama dengan
// kunci kanonik engine: NAS100 adalah alias dari USTEC, dan GOLD alias dari
// XAUUSD. Pencocokan dilakukan di Go, bukan di SQL, karena alias disimpan
// sebagai teks satu kolom — jumlah instrumennya kecil sehingga biayanya tidak
// berarti.
//
// Hanya instrumen yang aktif di katalog yang dikembalikan; instrumen yang
// dimatikan admin tidak lagi tampil di halaman publik.
func (s *publicSignalStore) ResolveSymbols(ctx context.Context, names []string) ([]SymbolRef, error) {
	if len(names) == 0 {
		return nil, nil
	}

	var rows []struct {
		Symbol      string
		DisplayName string
		Aliases     string
		CategoryKey string
	}
	err := s.db.WithContext(ctx).
		Table("trading_pair tp").
		Select("tp.symbol, tp.display_name, tp.aliases, c.key AS category_key").
		Joins("JOIN pair_category c ON c.id = tp.category_id").
		Where("tp.deleted_at IS NULL").
		Where("tp.is_enabled = ?", true).
		Where("c.is_enabled = ?", true).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca katalog pair publik: %w", err)
	}

	out := make([]SymbolRef, 0, len(names))
	for _, name := range names {
		needle := strings.TrimSpace(strings.ToUpper(name))
		if needle == "" {
			continue
		}
		for _, row := range rows {
			if !symbolCocok(row.Symbol, row.Aliases, needle) {
				continue
			}
			out = append(out, SymbolRef{
				Symbol:      row.Symbol,
				DisplayName: row.DisplayName,
				CategoryKey: row.CategoryKey,
				Aliases:     splitAliases(row.Aliases),
			})
			break
		}
	}
	return out, nil
}

// PreviewForSymbols mengambil status radar terbaru per simbol.
//
// Yang dibaca adalah baris radar (`signal_id` berawalan `status-`), sumber yang
// sama dengan yang dipakai dashboard member. Sinyal riwayat sengaja tidak ikut:
// di halaman publik, riwayat lama terbaca sebagai arah pasar yang berlaku
// sekarang, padahal bisa berumur belasan jam dan sudah bertentangan dengan
// status radar terkini.
//
// Baris radar memakai `published_at` beku sejak slot lahir, sedangkan
// `updated_at` bergerak setiap kali engine mengirim kemajuan. Pengurutan karena
// itu memakai `COALESCE(updated_at, published_at)`, sama seperti dashboard.
func (s *publicSignalStore) PreviewForSymbols(ctx context.Context, symbols []string, since time.Time) ([]model.Signal, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	// Slot yang dibatalkan atau ditolak tidak ditampilkan: di halaman publik,
	// arah seperti itu terbaca sah padahal sistem sudah menyatakannya tidak
	// berlaku.
	statusDisembunyikan := []string{
		model.SignalStatusCancelled,
		model.SignalStatusRejected,
		model.SignalStatusSkipped,
	}

	var rows []model.Signal
	err := s.db.WithContext(ctx).
		Model(&model.Signal{}).
		Where("symbol IN ?", symbols).
		Where("signal_id LIKE ?", "status-%").
		Where("status NOT IN ?", statusDisembunyikan).
		Where("COALESCE(updated_at, published_at) >= ?", since).
		Order("COALESCE(updated_at, published_at) DESC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca status radar publik: %w", err)
	}

	// Satu status terbaru per simbol. Pengurutan sudah menurun, jadi kemunculan
	// pertama adalah yang paling segar. Dikerjakan di Go supaya kueri tetap
	// sederhana dan berlaku sama di Postgres maupun SQLite.
	terpilih := make(map[string]model.Signal, len(symbols))
	urutan := make([]string, 0, len(symbols))
	for _, row := range rows {
		if _, sudah := terpilih[row.Symbol]; sudah {
			continue
		}
		terpilih[row.Symbol] = row
		urutan = append(urutan, row.Symbol)
	}

	out := make([]model.Signal, 0, len(urutan))
	for _, symbol := range urutan {
		out = append(out, terpilih[symbol])
	}
	return out, nil
}

// symbolCocok mencocokkan simbol kanonik atau salah satu aliasnya.
func symbolCocok(symbol, aliases, needle string) bool {
	if strings.EqualFold(symbol, needle) {
		return true
	}
	for _, alias := range splitAliases(aliases) {
		if strings.EqualFold(alias, needle) {
			return true
		}
	}
	return false
}

// splitAliases memecah daftar alias dan membuang yang kosong.
func splitAliases(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	bagian := strings.Split(raw, ",")
	out := make([]string, 0, len(bagian))
	for _, item := range bagian {
		if nama := strings.TrimSpace(item); nama != "" {
			out = append(out, nama)
		}
	}
	return out
}
