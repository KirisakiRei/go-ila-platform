package runner_test

import (
	"context"
	"testing"

	"ila/backend/internal/model"
	"ila/backend/internal/runner"
)

// fakeKatalog meniru tabel plan_feature.
type fakeKatalog map[string]map[string]string

func (f fakeKatalog) Feature(_ context.Context, tier, key, fallback string) string {
	if baris, ok := f[tier]; ok {
		if nilai, ok := baris[key]; ok && nilai != "" {
			return nilai
		}
	}
	return fallback
}

// TestReloadMengabaikanPaketYangTidakMenawarkan memastikan paket Free/Basic
// dengan "coming_soon" tidak lagi menahan tahap dari paket Plus. Sebelum
// perbaikan ini seluruh platform selalu jatuh ke tahap off.
func TestReloadMengabaikanPaketYangTidakMenawarkan(t *testing.T) {
	katalog := fakeKatalog{
		model.TierFree: {
			model.FeatureStrategies: "ila-2.0",
			model.FeatureILA3Status: "coming_soon",
		},
		model.TierBasic: {
			model.FeatureStrategies: "ila-2.0",
			model.FeatureILA3Status: "coming_soon",
		},
		model.TierPlus: {
			model.FeatureStrategies: "ila-2.0,ila-3.0",
			model.FeatureILA3Status: "general",
		},
	}

	s := runner.NewSettings(runner.ILA3Off)
	s.Reload(context.Background(), katalog)

	if got := s.ILA3Avail(); got != runner.ILA3General {
		t.Fatalf("tahap = %q, ingin %q", got, runner.ILA3General)
	}
	if !s.StrategySelectable("ila-3.0") {
		t.Error("ILA 3.0 harus bisa dipilih setelah tahap general")
	}
}

// TestReloadMemakaiTahapPalingKonservatif menjaga aturan lama: bila beberapa
// paket menawarkan strategi ini dengan tahap berbeda, tahap paling konservatif
// yang dipakai.
func TestReloadMemakaiTahapPalingKonservatif(t *testing.T) {
	katalog := fakeKatalog{
		model.TierBasic: {
			model.FeatureStrategies: "ila-2.0,ila-3.0",
			model.FeatureILA3Status: "beta",
		},
		model.TierPlus: {
			model.FeatureStrategies: "ila-2.0,ila-3.0",
			model.FeatureILA3Status: "general",
		},
	}

	s := runner.NewSettings(runner.ILA3Off)
	s.Reload(context.Background(), katalog)

	if got := s.ILA3Avail(); got != runner.ILA3Beta {
		t.Fatalf("tahap = %q, ingin %q", got, runner.ILA3Beta)
	}
}

// TestReloadTanpaPaketPenyediaTidakMengubahTahap memastikan nilai env tetap
// berlaku ketika katalog belum diisi.
func TestReloadTanpaPaketPenyediaTidakMengubahTahap(t *testing.T) {
	katalog := fakeKatalog{
		model.TierPlus: {
			model.FeatureStrategies: "ila-2.0",
			model.FeatureILA3Status: "coming_soon",
		},
	}

	s := runner.NewSettings(runner.ILA3General)
	s.Reload(context.Background(), katalog)

	if got := s.ILA3Avail(); got != runner.ILA3General {
		t.Fatalf("tahap = %q, ingin tetap %q", got, runner.ILA3General)
	}
}
