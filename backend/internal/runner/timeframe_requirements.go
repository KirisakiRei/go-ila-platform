package runner

import (
	"context"
	"sort"

	"ila/backend/internal/repository"
)

// TimeframeRequirement adalah satu pasangan timeframe yang perlu dihitung
// engine untuk sebuah simbol pada sebuah metode.
type TimeframeRequirement struct {
	Htf string
	Ltf string
}

// ComputeTimeframeRequirements menyusun pasangan timeframe yang diminta ke
// engine per metode, yaitu gabungan pasangan yang dipantau siapa pun.
//
// Mengapa gabungan, bukan per akun: langganan engine bersifat global, satu
// stream melayani semua pengguna. Penyaringan per akun terjadi di Runner
// (putuskanTimeframe), sehingga engine hanya perlu tahu pasangan mana yang
// dipakai siapa pun.
//
// Bila flag timeframe mati, hasilnya kosong: pemanggil tetap memakai daftar
// simbol saja, persis perilaku sebelum fitur ini ada.
func (s *Service) ComputeTimeframeRequirements(ctx context.Context, symbol string) (map[string][]TimeframeRequirement, error) {
	if !s.timeframeSwitchActive(ctx) {
		return nil, nil
	}

	switches, err := s.store.TimeframeSwitches(ctx, symbol)
	if err != nil {
		return nil, err
	}

	// metode -> pasangan -> terpakai
	terpakai := make(map[string]map[repository.TimeframeKey]bool)
	for _, perMetode := range switches {
		for strategy, pasangan := range perMetode {
			for key, state := range pasangan {
				if !state.Watch {
					continue
				}
				if terpakai[strategy] == nil {
					terpakai[strategy] = make(map[repository.TimeframeKey]bool)
				}
				terpakai[strategy][key] = true
			}
		}
	}

	out := make(map[string][]TimeframeRequirement, len(terpakai))
	for strategy, pasangan := range terpakai {
		// Urutan deterministik: peta Go tidak berurutan, sedangkan daftar yang
		// dikirim ke engine harus stabil agar tidak memicu sinkronisasi ulang.
		keys := make([]repository.TimeframeKey, 0, len(pasangan))
		for key := range pasangan {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].Htf == keys[j].Htf {
				return keys[i].Ltf < keys[j].Ltf
			}
			return keys[i].Htf < keys[j].Htf
		})

		list := make([]TimeframeRequirement, 0, len(keys))
		for _, key := range keys {
			list = append(list, TimeframeRequirement{Htf: key.Htf, Ltf: key.Ltf})
		}
		out[strategy] = list
	}
	return out, nil
}
