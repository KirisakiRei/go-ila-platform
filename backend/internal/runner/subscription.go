package runner

import (
	"context"
	"sort"
	"time"
)

// StrategyLocation menyatakan di mana engine untuk satu strategi berada.
type StrategyLocation struct {
	Strategy string
	Addr     string
}

// Requirement adalah kebutuhan satu simbol pada satu strategi.
type Requirement struct {
	Symbol   string
	Strategy string
}

// smtStrategies adalah strategi yang evaluasinya membandingkan instrumen
// korelasi (Smart Money Technique). Hanya strategi ini yang membuat Runner
// meminta feed tambahan ke engine; strategi lain cukup simbol yang dilanggan.
var smtStrategies = map[string]bool{
	"ila-3.0": true,
}

// ComputeRequirements menyusun daftar (symbol, strategy) yang perlu dihitung
// engine, berdasarkan user aktif, tier, dan pilihan strategi mereka.
//
// Inilah satu-satunya tempat user diterjemahkan menjadi kebutuhan engine,
// sehingga engine tetap buta terhadap siapa pun (plans/15 §2.2).
func (s *Service) ComputeRequirements(ctx context.Context, available []StrategyLocation) (map[string][]string, error) {
	subs, err := s.store.ListActiveSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	now := s.opts.Clock()
	byStrategy := make(map[string]map[string]bool)

	// PairGate memastikan simbol yang dinonaktifkan admin tidak pernah diminta
	// ke engine. Kegagalan membaca gate tidak boleh mematikan trading: simbol
	// tetap dianggap aktif agar tidak ada posisi yang kehilangan pengawasan.
	var enabled map[string]bool
	if s.opts.PairGate != nil {
		gate, gateErr := s.opts.PairGate.EnabledSymbols(ctx)
		if gateErr != nil {
			s.logger.Warn("gagal membaca katalog pair aktif", "error", gateErr.Error())
		} else {
			enabled = gate
		}
	}

	for _, sub := range subs {
		if !sub.IsActive {
			continue
		}

		user, userErr := s.store.FindUserByID(ctx, sub.UserID)
		if userErr != nil {
			continue
		}
		if user.Status != "" && user.Status != "active" {
			continue
		}

		tier := EffectiveTier(user.Tier, user.TrialExpiresAt, user.TierExpiresAt, now)
		if !StrategyAllowed(tier, sub.Strategy) {
			continue
		}
		// Strategi yang belum dirilis tidak boleh membuat engine dijalankan.
		if sub.Strategy == "ila-3.0" && !s.opts.Settings.StrategySelectable("ila-3.0") {
			continue
		}

		if byStrategy[sub.Strategy] == nil {
			byStrategy[sub.Strategy] = make(map[string]bool)
		}
		if enabled != nil && !enabled[sub.Symbol] {
			// Simbol dinonaktifkan admin: tidak diminta ke engine. Posisi yang
			// sudah terbuka tetap dikelola lewat pemantauan posisi.
			continue
		}
		byStrategy[sub.Strategy][sub.Symbol] = true
	}

	// Strategi SMT butuh lilin instrumen pembanding. Tanpa feed itu engine
	// tidak dapat menghitung divergence, walau user hanya melanggan simbol
	// utamanya.
	s.expandCorrelations(ctx, byStrategy, enabled)

	out := make(map[string][]string, len(byStrategy))
	for _, location := range available {
		symbols, ok := byStrategy[location.Strategy]
		if !ok {
			// Tidak ada user yang membutuhkan strategi ini: engine tidak jalan.
			continue
		}
		list := make([]string, 0, len(symbols))
		for symbol := range symbols {
			list = append(list, symbol)
		}
		sort.Strings(list)
		out[location.Strategy] = list
	}

	return out, nil
}

// expandCorrelations menambahkan instrumen pembanding SMT ke kebutuhan engine.
//
// Ekspansi hanya satu tingkat: pembanding ditambahkan apa adanya, tanpa ikut
// dipecah lagi, sehingga rantai korelasi tidak pernah melebar tanpa batas.
// Instrumen yang dinonaktifkan admin tetap dilewati, sama seperti simbol utama.
func (s *Service) expandCorrelations(ctx context.Context, byStrategy map[string]map[string]bool, enabled map[string]bool) {
	if s.opts.CorrelationGate == nil {
		return
	}

	needsSMT := false
	for strategy := range byStrategy {
		if smtStrategies[strategy] {
			needsSMT = true
			break
		}
	}
	if !needsSMT {
		return
	}

	correlations, err := s.opts.CorrelationGate.Correlations(ctx)
	if err != nil {
		// Kegagalan membaca korelasi tidak boleh mematikan trading: engine tetap
		// menerima simbol utama, hanya evaluasi SMT yang kehilangan pembanding.
		s.logger.Warn("gagal membaca korelasi pair", "error", err.Error())
		return
	}
	if len(correlations) == 0 {
		return
	}

	for strategy, symbols := range byStrategy {
		if !smtStrategies[strategy] {
			continue
		}
		// Salin kunci lebih dulu: penambahan saat iterasi peta membuat hasil
		// bergantung urutan, padahal kebutuhan engine harus deterministik.
		base := make([]string, 0, len(symbols))
		for symbol := range symbols {
			base = append(base, symbol)
		}
		for _, symbol := range base {
			for _, correlated := range correlations[symbol] {
				if correlated == "" {
					continue
				}
				if enabled != nil && !enabled[correlated] {
					continue
				}
				symbols[correlated] = true
			}
		}
	}
}

// DiffRequirements membandingkan kebutuhan lama dan baru.
// Hasilnya dipakai untuk memanggil Subscribe/Unsubscribe tanpa memutus
// simbol yang tidak berubah.
func DiffRequirements(old, current []string) (added, removed []string) {
	oldSet := make(map[string]bool, len(old))
	for _, v := range old {
		oldSet[v] = true
	}

	currentSet := make(map[string]bool, len(current))
	for _, v := range current {
		currentSet[v] = true
	}

	// Dibandingkan lewat set, sehingga simbol berulang dihitung sekali.
	for v := range currentSet {
		if !oldSet[v] {
			added = append(added, v)
		}
	}

	for v := range oldSet {
		if !currentSet[v] {
			removed = append(removed, v)
		}
	}

	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

// RefreshInterval adalah jeda pemeriksaan ulang kebutuhan engine.
// Perubahan tier atau simbol baru terasa dalam hitungan detik tanpa restart.
const RefreshInterval = 60 * time.Second
