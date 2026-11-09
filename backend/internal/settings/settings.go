// Package settings menyatukan nilai pengaturan operasional.
//
// Urutannya selalu: nilai dari database menimpa nilai dari environment.
// Tabel `system_setting` murni override — bila kosong, env yang berlaku.
// Dengan begitu menghapus satu baris cukup untuk kembali ke bawaan
// (plans/11 Fase I §11.2).
package settings

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CacheTTL adalah umur cache. Perubahan flag berlaku paling lama selama ini,
// tanpa memaksa setiap request membaca database (plans/11 I2).
const CacheTTL = 10 * time.Second

// Kunci flag yang dikenali. Menambah kunci baru berarti menambah baris di sini
// agar tidak ada flag liar yang tersimpan tanpa diketahui kode.
const (
	KeyAutoExecute    = "AUTO_EXECUTE"
	KeyDryRunAll      = "DRY_RUN_ALL"
	KeyMaxDailyTrades = "MAX_DAILY_TRADES"
	KeyMaxDailyLosses = "MAX_DAILY_LOSSES"
	KeyStopAfterWin   = "STOP_AFTER_WIN"
	KeyMarketOpenOnly = "MARKET_OPEN_ONLY"
	KeyLegalGate      = "LEGAL_GATE"
	// KeyPairSwitch mengaktifkan saklar metode per akun × pair di Runner.
	// Bawaan mati agar perilaku lama tidak berubah sampai data saklar lengkap.
	KeyPairSwitch = "PAIR_SWITCH_ENABLED"
)

// Kind menjelaskan cara nilai dibaca dan ditampilkan.
type Kind string

const (
	KindBool Kind = "bool"
	KindInt  Kind = "int"
)

// Definition adalah satu flag yang boleh diubah dari panel admin.
type Definition struct {
	Key   string
	Label string
	Help  string
	Kind  Kind
	// Min dan Max membatasi nilai angka; diabaikan untuk bool.
	Min int
	Max int
	// Danger menandai flag yang mengubah perilaku trading secara luas.
	Danger bool
}

// Catalog adalah daftar lengkap flag yang boleh diubah.
var Catalog = []Definition{
	{KeyAutoExecute, "Eksekusi otomatis", "Bila mati, seluruh eksekusi ditahan meski akun ber-mode auto.", KindBool, 0, 0, true},
	{KeyDryRunAll, "Paksa simulasi", "Semua order menjadi simulasi. Aman untuk uji coba.", KindBool, 0, 0, true},
	{KeyMaxDailyTrades, "Maksimal trade harian", "Batas jumlah entry per hari akuntansi (WIB).", KindInt, 1, 20, false},
	{KeyMaxDailyLosses, "Maksimal kerugian harian", "Setelah batas ini tercapai, entry baru ditahan.", KindInt, 1, 20, false},
	{KeyStopAfterWin, "Berhenti setelah menang", "Satu kemenangan mengakhiri hari perdagangan.", KindBool, 0, 0, false},
	{KeyMarketOpenOnly, "Hanya saat pasar buka", "Menolak entry ketika pasar tutup.", KindBool, 0, 0, false},
	{KeyLegalGate, "Tahan bila dokumen belum disetujui", "Entry baru ditolak sampai dokumen wajib disetujui.", KindBool, 0, 0, false},
	{KeyPairSwitch, "Saklar metode per akun", "Bila nyala, Runner hanya menjalankan metode yang dinyalakan akun pada pair tersebut.", KindBool, 0, 0, true},
}

// Definition mencari flag berdasarkan kunci.
func Lookup(key string) (Definition, bool) {
	for _, item := range Catalog {
		if item.Key == key {
			return item, true
		}
	}
	return Definition{}, false
}

// OverrideStore membaca dan menyimpan nilai dari database.
type OverrideStore interface {
	Load(ctx context.Context) (map[string]string, error)
	Save(ctx context.Context, key, value, reason, actorID string) error
	Delete(ctx context.Context, key string) error
}

// Service membaca nilai efektif sebuah flag.
type Service struct {
	store OverrideStore
	env   func(key string) string
	ttl   time.Duration
	now   func() time.Time

	mu      sync.RWMutex
	cached  map[string]string
	expires time.Time
}

// NewService merangkai pembaca pengaturan.
func NewService(store OverrideStore, env func(key string) string) *Service {
	if env == nil {
		env = func(string) string { return "" }
	}
	return &Service{
		store: store,
		env:   env,
		ttl:   CacheTTL,
		now:   func() time.Time { return time.Now().UTC() },
	}
}

// SetTTL mengganti umur cache; dipakai pengujian.
func (s *Service) SetTTL(ttl time.Duration) {
	s.ttl = ttl
}

// Invalidate memaksa pembacaan ulang pada permintaan berikutnya.
func (s *Service) Invalidate() {
	s.mu.Lock()
	s.cached = nil
	s.expires = time.Time{}
	s.mu.Unlock()
}

// raw mengembalikan nilai mentah efektif, sudah memperhitungkan override.
func (s *Service) raw(ctx context.Context, key string) string {
	overrides := s.overrides(ctx)
	if value, ok := overrides[key]; ok && strings.TrimSpace(value) != "" {
		return value
	}
	return s.env(key)
}

func (s *Service) overrides(ctx context.Context) map[string]string {
	s.mu.RLock()
	if s.cached != nil && s.now().Before(s.expires) {
		cached := s.cached
		s.mu.RUnlock()
		return cached
	}
	s.mu.RUnlock()

	loaded, err := s.store.Load(ctx)
	if err != nil {
		// Gagal membaca bukan alasan menghentikan sistem: pakai nilai terakhir
		// yang diketahui, atau env bila belum pernah berhasil.
		s.mu.RLock()
		last := s.cached
		s.mu.RUnlock()
		if last != nil {
			return last
		}
		return map[string]string{}
	}

	s.mu.Lock()
	s.cached = loaded
	s.expires = s.now().Add(s.ttl)
	s.mu.Unlock()
	return loaded
}

// Bool membaca flag sebagai boolean. Nilai tak dikenal memakai fallback.
func (s *Service) Bool(ctx context.Context, key string, fallback bool) bool {
	value, err := strconv.ParseBool(strings.TrimSpace(s.raw(ctx, key)))
	if err != nil {
		return fallback
	}
	return value
}

// Int membaca flag sebagai bilangan dengan batas dari katalog.
func (s *Service) Int(ctx context.Context, key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(s.raw(ctx, key)))
	if err != nil {
		return fallback
	}
	if def, ok := Lookup(key); ok && def.Kind == KindInt {
		if value < def.Min {
			return def.Min
		}
		if value > def.Max {
			return def.Max
		}
	}
	return value
}

// Source menandai asal nilai: "database" bila ada override, "env" bila tidak.
func (s *Service) Source(ctx context.Context, key string) string {
	overrides := s.overrides(ctx)
	if value, ok := overrides[key]; ok && strings.TrimSpace(value) != "" {
		return "database"
	}
	return "env"
}

// Value mengembalikan nilai efektif dalam bentuk teks.
func (s *Service) Value(ctx context.Context, key string) string {
	return strings.TrimSpace(s.raw(ctx, key))
}

// Write menyimpan override lalu membuang cache agar berlaku seketika di proses
// ini. Proses lain menunggu CacheTTL (plans/11 I2).
func (s *Service) Write(ctx context.Context, key, value, reason, actorID string) error {
	if err := Validate(key, value); err != nil {
		return err
	}
	if strings.TrimSpace(reason) == "" {
		return ErrEmptyValue
	}
	actorID = strings.TrimSpace(actorID)
	if err := s.store.Save(ctx, key, strings.TrimSpace(value), strings.TrimSpace(reason), actorID); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}

// Snapshot mengembalikan seluruh flag beserta asalnya untuk tampilan admin.
func (s *Service) Snapshot(ctx context.Context) []Snapshot {
	out := make([]Snapshot, 0, len(Catalog))
	for _, def := range Catalog {
		out = append(out, Snapshot{
			Key:    def.Key,
			Label:  def.Label,
			Help:   def.Help,
			Kind:   string(def.Kind),
			Value:  s.Value(ctx, def.Key),
			Source: s.Source(ctx, def.Key),
			Danger: def.Danger,
			Min:    def.Min,
			Max:    def.Max,
		})
	}
	return out
}

// Snapshot adalah satu baris tampilan pengaturan.
type Snapshot struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Help   string `json:"help"`
	Kind   string `json:"kind"`
	Value  string `json:"value"`
	Source string `json:"source"`
	Danger bool   `json:"danger"`
	Min    int    `json:"min,omitempty"`
	Max    int    `json:"max,omitempty"`
}

// Validate memastikan nilai sesuai jenis flag sebelum disimpan.
func Validate(key, value string) error {
	def, ok := Lookup(key)
	if !ok {
		return ErrUnknownKey
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return ErrEmptyValue
	}

	switch def.Kind {
	case KindBool:
		if _, err := strconv.ParseBool(value); err != nil {
			return ErrBadValue
		}
	case KindInt:
		number, err := strconv.Atoi(value)
		if err != nil {
			return ErrBadValue
		}
		if number < def.Min || number > def.Max {
			return ErrOutOfRange
		}
	}
	return nil
}
