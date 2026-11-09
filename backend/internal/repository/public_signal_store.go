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

// PreviewForSymbols mengambil sinyal terbaru per simbol.
//
// Sinyal yang DIBATALKAN, DITOLAK, atau DILEWATI tidak pernah diambil: di
// halaman publik, sinyal seperti itu terbaca sebagai arah pasar yang sah
// padahal sistem sudah menyatakannya tidak berlaku. Status yang boleh tampil
// hanya yang memang diterbitkan ke pengguna.
func (s *publicSignalStore) PreviewForSymbols(ctx context.Context, symbols []string, since time.Time) ([]model.Signal, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	statusTampil := []string{
		model.SignalStatusReadyToEntry,
		model.SignalStatusOpportunity,
		model.SignalStatusAcked,
		model.SignalStatusConsumed,
		model.SignalStatusExpired,
		model.SignalStatusInvalidated,
	}

	var rows []model.Signal
	err := s.db.WithContext(ctx).
		Model(&model.Signal{}).
		Where("symbol IN ?", symbols).
		Where("status IN ?", statusTampil).
		Where("published_at >= ?", since).
		Order("published_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca sinyal publik: %w", err)
	}

	// Satu sinyal terakhir per simbol; pengurutan sudah menurun sehingga
	// kemunculan pertama adalah yang terbaru. Selesai di Go supaya kueri tetap
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
