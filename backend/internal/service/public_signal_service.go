package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
)

// PublicSignalService menyusun ringkasan sinyal untuk halaman publik.
//
// Batasan yang dipegang:
//   - Simbol tidak pernah datang bebas dari klien. Yang diterima hanya nama
//     yang ada di daftar izin; sisanya diabaikan, bukan diteruskan ke kueri.
//   - Hasilnya tidak memuat harga masuk, batas kerugian, target, maupun
//     ringkasan analisis.
//   - Tanpa sinyal bukan kesalahan. Pengunjung tetap menerima daftar
//     instrumen dengan penanda "belum ada arah".
type PublicSignalService interface {
	Preview(ctx context.Context, symbols []string) (*dto.PublicSignalPreview, error)
}

// SimbolBawaan adalah instrumen yang ditampilkan saat klien tidak meminta apa pun.
var SimbolBawaan = []string{"XAUUSD", "EURUSD", "NAS100"}

// BatasSimbol membatasi jumlah instrumen per permintaan agar satu panggilan
// tidak bisa dipakai menelusuri seluruh katalog.
const BatasSimbol = 5

// UsiaSinyalTerbaru membatasi seberapa lama sebuah arah masih dianggap relevan
// untuk halaman depan. Lewat dari ini, kartu kembali ke keadaan "sedang
// dipantau" daripada menampilkan arah yang sudah basi.
const UsiaSinyalTerbaru = 7 * 24 * time.Hour

type publicSignalService struct {
	store  repository.PublicSignalStore
	market PublicMarketReader
	now    func() time.Time
}

// PublicMarketReader menyediakan keadaan pasar terkini. Dipenuhi layanan
// status publik yang sudah ada agar jadwal buka/tutup dihitung di satu tempat.
type PublicMarketReader interface {
	Status(ctx context.Context) (*dto.PublicStatus, error)
}

// NewPublicSignalService merangkai penyusun ringkasan sinyal publik.
func NewPublicSignalService(store repository.PublicSignalStore, market PublicMarketReader) PublicSignalService {
	return &publicSignalService{
		store:  store,
		market: market,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// Preview menyusun ringkasan untuk simbol yang diminta.
func (s *publicSignalService) Preview(ctx context.Context, symbols []string) (*dto.PublicSignalPreview, error) {
	diminta, err := s.normalizeSymbols(symbols)
	if err != nil {
		return nil, err
	}

	refs, err := s.store.ResolveSymbols(ctx, diminta)
	if err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return nil, validation("simbol tidak dikenali katalog")
	}

	now := s.now()

	// Kueri memakai simbol kanonik hasil resolusi, bukan nama yang dikirim
	// klien, sehingga alias seperti NAS100 tetap menemukan sinyal USTEC.
	kanonik := make([]string, 0, len(refs)*2)
	for _, ref := range refs {
		kanonik = append(kanonik, ref.Symbol)
		kanonik = append(kanonik, ref.Aliases...)
	}

	terbaru, err := s.store.PreviewForSymbols(ctx, kanonik, now.Add(-UsiaSinyalTerbaru))
	if err != nil {
		return nil, err
	}
	perSimbol := make(map[string]model.Signal, len(terbaru))
	for _, signal := range terbaru {
		perSimbol[strings.ToUpper(signal.Symbol)] = signal
	}

	items := make([]dto.PublicSignalPreviewItem, 0, len(refs))
	for _, ref := range refs {
		item := dto.PublicSignalPreviewItem{
			Symbol:      ref.Symbol,
			DisplayName: ref.DisplayName,
			Category:    ref.CategoryKey,
		}

		// Simbol kanonik dicoba lebih dulu; alias dipakai bila engine mengirim
		// nama lain untuk instrumen yang sama.
		signal, ada := perSimbol[strings.ToUpper(ref.Symbol)]
		if !ada {
			for _, alias := range ref.Aliases {
				if ditemukan, ok := perSimbol[strings.ToUpper(alias)]; ok {
					signal, ada = ditemukan, true
					break
				}
			}
		}

		if ada {
			item.HasSignal = true
			item.Signal = &dto.PublicSignalSnapshot{
				Direction:   strings.ToUpper(signal.Direction),
				Htf:         signal.Htf,
				Ltf:         signal.Ltf,
				Status:      publicSignalStatus(signal.Status),
				PublishedAt: signal.PublishedAt,
			}
		}
		items = append(items, item)
	}

	return &dto.PublicSignalPreview{
		GeneratedAt: now,
		Market:      s.marketState(ctx),
		Items:       items,
	}, nil
}

// normalizeSymbols membersihkan masukan klien dan menegakkan batas jumlah.
func (s *publicSignalService) normalizeSymbols(symbols []string) ([]string, error) {
	if len(symbols) == 0 {
		return append([]string(nil), SimbolBawaan...), nil
	}

	bersih := make([]string, 0, len(symbols))
	terlihat := make(map[string]struct{}, len(symbols))
	for _, raw := range symbols {
		nama := strings.ToUpper(strings.TrimSpace(raw))
		if nama == "" {
			continue
		}
		if len(nama) > 20 || !simbolSah(nama) {
			return nil, validation("simbol tidak sah")
		}
		if _, dup := terlihat[nama]; dup {
			continue
		}
		terlihat[nama] = struct{}{}
		bersih = append(bersih, nama)
	}

	if len(bersih) == 0 {
		return append([]string(nil), SimbolBawaan...), nil
	}
	if len(bersih) > BatasSimbol {
		return nil, validation("maksimal " + strconv.Itoa(BatasSimbol) + " simbol per permintaan")
	}
	return bersih, nil
}

// marketState membaca jadwal pasar; kegagalan tidak menggagalkan ringkasan.
func (s *publicSignalService) marketState(ctx context.Context) dto.PublicMarketState {
	if s.market == nil {
		return dto.PublicMarketState{Open: false, Label: "Jadwal pasar belum tersedia"}
	}
	status, err := s.market.Status(ctx)
	if err != nil || status == nil {
		return dto.PublicMarketState{Open: false, Label: "Jadwal pasar belum tersedia"}
	}
	return dto.PublicMarketState{Open: status.Market.Open, Label: status.Market.Label}
}

// publicSignalStatus menyederhanakan status internal menjadi dua keadaan yang
// berarti bagi pengunjung: masih berlaku atau sudah lewat.
func publicSignalStatus(status string) string {
	if status == model.SignalStatusExpired || status == model.SignalStatusInvalidated {
		return "EXPIRED"
	}
	return "ACTIVE"
}

// simbolSah menerima huruf, angka, titik, dan garis bawah — bentuk yang dipakai
// simbol instrumen. Karakter lain ditolak agar nilai klien tidak pernah masuk
// ke kueri dalam bentuk tak terduga.
func simbolSah(nama string) bool {
	for _, r := range nama {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}
