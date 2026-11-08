package service

import (
	"context"
	"strings"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
)

// onboardingSteps adalah urutan langkah yang ditampilkan ke pengguna baru.
// Satu sumber untuk backend dan frontend, sehingga tidak ada dua versi.
var onboardingSteps = []string{
	"verifikasi_email",
	"hubungkan_broker",
	"pilih_strategi",
	"hubungkan_telegram",
	"setel_risiko",
}

// graceBannerDays adalah jarak hari saat peringatan mulai muncul.
const graceBannerDays = 7

// Onboarding mengembalikan posisi alur mulai pengguna.
func (s *emailAccountService) Onboarding(ctx context.Context, userID string) (*dto.OnboardingState, error) {
	user, err := s.store.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.onboardingState(ctx, user)
}

// SaveOnboarding menyimpan langkah yang sudah selesai agar alur bisa
// dilanjutkan setelah keluar-tengah-jalan (plans/17 §5).
func (s *emailAccountService) SaveOnboarding(ctx context.Context, userID string, req dto.OnboardingRequest) (*dto.OnboardingState, error) {
	user, err := s.store.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if req.Step > len(onboardingSteps) {
		return nil, validation("langkah onboarding tidak dikenal")
	}
	// Langkah tidak pernah mundur: kembali ke halaman sebelumnya bukan berarti
	// membatalkan yang sudah diselesaikan.
	if req.Step > user.OnboardingStep {
		user.OnboardingStep = req.Step
	}
	if req.Done {
		user.OnboardingDone = true
		user.OnboardingStep = len(onboardingSteps)
	}
	if err := s.store.Update(ctx, user); err != nil {
		return nil, err
	}
	return s.onboardingState(ctx, user)
}

func (s *emailAccountService) onboardingState(ctx context.Context, user *model.User) (*dto.OnboardingState, error) {
	return &dto.OnboardingState{
		Step:       user.OnboardingStep,
		Done:       user.OnboardingDone,
		TotalSteps: len(onboardingSteps),
		Steps:      onboardingSteps,
		Verified:   user.EmailVerifiedAt != nil,
		TelegramOK: user.TelegramChatID != nil,
		NextAction: nextOnboardingAction(user),
	}, nil
}

// nextOnboardingAction memilih satu langkah berikutnya agar tampilan tidak
// memaksa pengguna menebak apa yang harus dilakukan.
func nextOnboardingAction(user *model.User) string {
	if user.EmailVerifiedAt == nil {
		return onboardingSteps[0]
	}
	if user.OnboardingStep >= len(onboardingSteps) {
		return ""
	}
	return onboardingSteps[user.OnboardingStep]
}

// Banner merangkum keadaan langganan untuk banner dashboard.
func (s *emailAccountService) Banner(ctx context.Context, userID string) (*dto.SubscriptionBanner, error) {
	user, err := s.store.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := s.now()

	// Uji coba berlaku lebih dulu selama periodenya belum habis.
	expiry := user.TierExpiresAt
	if user.TrialExpiresAt != nil && now.Before(*user.TrialExpiresAt) {
		expiry = user.TrialExpiresAt
	}

	if expiry == nil {
		return &dto.SubscriptionBanner{
			State:   "normal",
			Title:   "Paket " + tierLabel(user.Tier),
			Message: "Tidak ada batas waktu pada paket ini.",
			Level:   "tier",
		}, nil
	}

	daysLeft := int(time.Until(*expiry).Hours() / 24)
	if daysLeft < 0 {
		daysLeft = 0
	}

	switch {
	case now.After(*expiry):
		// Masa tenggang tetap memberi akses baca; entry baru ditahan Runner.
		if user.GraceUntil != nil && now.Before(*user.GraceUntil) {
			return &dto.SubscriptionBanner{
				State:     "grace",
				Title:     "Langganan berakhir, masa tenggang aktif",
				Message:   "Entry baru ditahan sampai pembayaran diterima. Posisi berjalan tetap dikelola.",
				ExpiresAt: user.GraceUntil,
				DaysLeft:  daysUntil(now, *user.GraceUntil),
				Level:     "warning",
			}, nil
		}
		return &dto.SubscriptionBanner{
			State:     "expired",
			Title:     "Langganan berakhir",
			Message:   "Perpanjang langganan untuk membuka entry baru.",
			ExpiresAt: expiry,
			Level:     "critical",
		}, nil
	case daysLeft <= graceBannerDays:
		return &dto.SubscriptionBanner{
			State:     "ending",
			Title:     "Langganan akan berakhir",
			Message:   "Perpanjang sekarang agar eksekusi tidak tertahan.",
			ExpiresAt: expiry,
			DaysLeft:  daysLeft,
			Level:     "warning",
		}, nil
	default:
		return &dto.SubscriptionBanner{
			State:     "normal",
			Title:     "Paket " + tierLabel(user.Tier),
			Message:   "Aktif sampai tanggal berikut.",
			ExpiresAt: expiry,
			DaysLeft:  daysLeft,
			Level:     "info",
		}, nil
	}
}

// tierLabel menampilkan nama paket dengan huruf awal besar.
func tierLabel(tier string) string {
	if tier == "" {
		return "Free"
	}
	return strings.ToUpper(tier[:1]) + tier[1:]
}

func daysUntil(now, target time.Time) int {
	days := int(target.Sub(now).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
