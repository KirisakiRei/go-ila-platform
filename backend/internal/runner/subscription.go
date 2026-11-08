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
