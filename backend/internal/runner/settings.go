package runner

import (
	"context"
	"os"
	"strings"
	"sync/atomic"

	"ila/backend/internal/model"
)

// Settings adalah pengaturan platform yang dibaca dari environment.
// Nilainya bisa berubah tanpa deploy ulang agar ketersediaan strategi baru
// tidak menuntut rilis per tahap (plans/15 §6.1).
type Settings struct {
	// ILA3Availability: off | internal | beta | general.
	ila3 atomic.Value
}

// Nilai ketersediaan strategi ILA 3.0.
const (
	ILA3Off      = "off"
	ILA3Internal = "internal"
	ILA3Beta     = "beta"
	ILA3General  = "general"
)

// LoadSettings membaca pengaturan dari environment.
func LoadSettings() *Settings {
	s := &Settings{}
	s.ila3.Store(ILA3Off)

	switch strings.ToLower(strings.TrimSpace(os.Getenv("ILA3_AVAILABILITY"))) {
	case ILA3Internal:
		s.ila3.Store(ILA3Internal)
	case ILA3Beta:
		s.ila3.Store(ILA3Beta)
	case ILA3General:
		s.ila3.Store(ILA3General)
	}
	return s
}

// ILA3Avail mengembalikan tahap ketersediaan ILA 3.0.
func (s *Settings) ILA3Avail() string {
	if s == nil {
		return ILA3Off
	}
	v, _ := s.ila3.Load().(string)
	if v == "" {
		return ILA3Off
	}
	return v
}

// ILA3Running menandai apakah engine ILA 3.0 boleh dijalankan sama sekali.
func (s *Settings) ILA3Running() bool {
	return s.ILA3Avail() != ILA3Off
}

// StrategyStatus mengembalikan status strategi yang terlihat oleh user.
func (s *Settings) StrategyStatus(strategy string) string {
	if strategy != "ila-3.0" {
		return "active"
	}
	switch s.ILA3Avail() {
	case ILA3General:
		return "active"
	case ILA3Beta:
		return "beta"
	default:
		return "coming_soon"
	}
}

// StrategySelectable menandai strategi yang boleh dipilih user.
func (s *Settings) StrategySelectable(strategy string) bool {
	if strategy == "ila-3.0" {
		return s.ILA3Avail() == ILA3General || s.ILA3Avail() == ILA3Beta
	}
	return true
}

// NewSettings membuat pengaturan dengan tahap ILA 3.0 tertentu.
// Dipakai saat pengujian dan saat konfigurasi diberikan langsung oleh pemanggil.
func NewSettings(ila3 string) *Settings {
	s := &Settings{}
	s.ila3.Store(ILA3Off)
	switch ila3 {
	case ILA3Internal, ILA3Beta, ILA3General:
		s.ila3.Store(ila3)
	}
	return s
}

// ILA3StatusReader membaca tahap ILA 3.0 dari katalog paket.
// Katalog adalah sumber kebenaran yang juga dibaca panel admin dan
// dashboard, sehingga tiga tempat itu tidak bisa berbeda nilai.
type ILA3StatusReader interface {
	Feature(ctx context.Context, tier, key, fallback string) string
}

// Reload menurunkan tahap ILA 3.0 dari katalog paket.
// Semua paket yang menawarkan strategi ini harus menyetujui tahap yang sama;
// bila berbeda, tahap paling konservatif dipakai agar tidak ada paket yang
// mendapat fitur lebih cepat dari yang lain.
func (s *Settings) Reload(ctx context.Context, reader ILA3StatusReader) {
	if s == nil || reader == nil {
		return
	}

	stage := ""
	for _, tier := range []string{model.TierFree, model.TierBasic, model.TierPlus} {
		value := normalizeStage(reader.Feature(ctx, tier, model.FeatureILA3Status, ""))
		if value == "" {
			continue
		}
		if stage == "" || stageRank(value) < stageRank(stage) {
			stage = value
		}
	}
	if stage == "" {
		return
	}
	s.ila3.Store(stage)
}

// normalizeStage menerjemahkan nilai katalog paket ke tahap internal.
func normalizeStage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case ILA3Off, ILA3Internal, ILA3Beta, ILA3General:
		return strings.ToLower(strings.TrimSpace(value))
	case "coming_soon", "disabled", "":
		return ILA3Off
	default:
		return ILA3Off
	}
}

// stageRank mengurutkan tahap dari paling konservatif.
func stageRank(stage string) int {
	switch stage {
	case ILA3Off:
		return 0
	case ILA3Internal:
		return 1
	case ILA3Beta:
		return 2
	case ILA3General:
		return 3
	default:
		return 0
	}
}
