package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/utils"
)

// adminLoginFixture menyiapkan pintu admin dengan 2FA yang bisa dinyalakan.
type adminLoginFixture struct {
	svc        *adminAuthService
	store      *fakeChallengeStore
	sealerGate TwoFactorGate
	clock      *time.Time
}

func newAdminLoginFixture(t *testing.T) *adminLoginFixture {
	t.Helper()

	base := newAdminAuthFixture(t)
	store := newFakeChallengeStore()

	// Gate sederhana: 2FA aktif bila di-set, kode "123456" dianggap sah.
	gate := &stubGate{}

	svc := NewAdminAuthService(AdminAuthDeps{
		Users:      base.users,
		Roles:      base.store,
		Perms:      &stubPermReader{users: base.users},
		Issuer:     utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour),
		TwoFactor:  gate.gate(),
		Challenges: store,
	}).(*adminAuthService)
	// Jam dipakai bersama oleh layanan dan pengelola tantangan.
	clock := time.Unix(1_700_000_000, 0).UTC()
	svc.now = func() time.Time { return clock }
	svc.challenges.setClock(svc.now)

	return &adminLoginFixture{svc: svc, store: store, sealerGate: gate.gate(), clock: &clock}
}

// stubGate adalah TwoFactorGate yang bisa diatur dari tes.
type stubGate struct {
	enabled bool
	code    string
}

func (g *stubGate) Enabled(context.Context, string) (bool, error) { return g.enabled, nil }

func (g *stubGate) Verify(_ context.Context, _ string, code string) error {
	if code == g.code {
		return nil
	}
	return validation("kode 2FA tidak sesuai")
}

func (g *stubGate) ConsumeRecovery(_ context.Context, _ string, code string) (bool, error) {
	return false, nil
}

// Implementasi agar stubGate bisa dipakai challangeService.
func (g *stubGate) gate() TwoFactorGate { return g }

// Tanpa 2FA, login langsung menerbitkan sesi seperti sebelumnya.
func TestLoginAdminTanpa2FALangsungSelesai(t *testing.T) {
	f := newAdminLoginFixture(t)

	result, err := f.svc.Login(context.Background(), dto.LoginRequest{
		Identifier: "admin@ila.id", Password: "rahasia-panjang-123",
	}, LoginContext{})
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if result.Step != string(StepComplete) {
		t.Fatalf("step = %q; ingin complete", result.Step)
	}
	if result.Token == "" {
		t.Fatal("sesi seharusnya diterbitkan")
	}
}

// Dengan 2FA, login berhenti di tantangan dan belum menerbitkan sesi.
func TestLoginAdminDengan2FAMenuntutKode(t *testing.T) {
	f := newAdminLoginFixture(t)
	f.sealerGate.(*stubGate).enabled = true
	f.sealerGate.(*stubGate).code = "123456"

	result, err := f.svc.Login(context.Background(), dto.LoginRequest{
		Identifier: "admin@ila.id", Password: "rahasia-panjang-123",
	}, LoginContext{})
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}

	if result.Step != string(StepTwoFactor) {
		t.Fatalf("step = %q; ingin two_factor_required", result.Step)
	}
	if result.Token != "" {
		t.Fatal("sesi tidak boleh terbit sebelum kode diverifikasi")
	}
	if result.ChallengeID == "" {
		t.Fatal("challenge id seharusnya ada")
	}
}

// Kode yang benar menuntaskan login; kode salah ditolak.
func TestVerifikasi2FAMenuntaskanLogin(t *testing.T) {
	f := newAdminLoginFixture(t)
	f.sealerGate.(*stubGate).enabled = true
	f.sealerGate.(*stubGate).code = "123456"
	ctx := context.Background()

	first, err := f.svc.Login(ctx, dto.LoginRequest{
		Identifier: "admin@ila.id", Password: "rahasia-panjang-123",
	}, LoginContext{})
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}

	// Kode salah.
	if _, err := f.svc.VerifyTwoFactor(ctx, dto.VerifyTwoFactorRequest{
		ChallengeID: first.ChallengeID, Code: "000000",
	}, LoginContext{}); err == nil {
		t.Fatal("kode salah seharusnya ditolak")
	}

	// Kode benar.
	result, err := f.svc.VerifyTwoFactor(ctx, dto.VerifyTwoFactorRequest{
		ChallengeID: first.ChallengeID, Code: "123456",
	}, LoginContext{})
	if err != nil {
		t.Fatalf("VerifyTwoFactor error: %v", err)
	}
	if result.Token == "" {
		t.Fatal("sesi seharusnya terbit setelah kode sah")
	}
	if result.Step != string(StepComplete) {
		t.Fatalf("step = %q; ingin complete", result.Step)
	}
}

// Tantangan sekali pakai: kode benar tidak bisa dipakai dua kali.
func TestTantanganSekaliPakai(t *testing.T) {
	f := newAdminLoginFixture(t)
	f.sealerGate.(*stubGate).enabled = true
	f.sealerGate.(*stubGate).code = "123456"
	ctx := context.Background()

	first, _ := f.svc.Login(ctx, dto.LoginRequest{
		Identifier: "admin@ila.id", Password: "rahasia-panjang-123",
	}, LoginContext{})

	if _, err := f.svc.VerifyTwoFactor(ctx, dto.VerifyTwoFactorRequest{
		ChallengeID: first.ChallengeID, Code: "123456",
	}, LoginContext{}); err != nil {
		t.Fatalf("pemakaian pertama seharusnya berhasil: %v", err)
	}

	if _, err := f.svc.VerifyTwoFactor(ctx, dto.VerifyTwoFactorRequest{
		ChallengeID: first.ChallengeID, Code: "123456",
	}, LoginContext{}); err == nil {
		t.Fatal("tantangan yang sudah dipakai seharusnya ditolak")
	}
}

// Tantangan hangus setelah 5 kode salah.
func TestTantanganHangusSetelahLimaSalah(t *testing.T) {
	f := newAdminLoginFixture(t)
	f.sealerGate.(*stubGate).enabled = true
	f.sealerGate.(*stubGate).code = "123456"
	ctx := context.Background()

	first, _ := f.svc.Login(ctx, dto.LoginRequest{
		Identifier: "admin@ila.id", Password: "rahasia-panjang-123",
	}, LoginContext{})

	for i := 0; i < MaxChallengeAttempts; i++ {
		if _, err := f.svc.VerifyTwoFactor(ctx, dto.VerifyTwoFactorRequest{
			ChallengeID: first.ChallengeID, Code: "000000",
		}, LoginContext{}); err == nil {
			t.Fatal("kode salah seharusnya ditolak")
		}
	}

	// Kode benar pun tidak lagi diterima: harus mulai dari kata sandi.
	_, err := f.svc.VerifyTwoFactor(ctx, dto.VerifyTwoFactorRequest{
		ChallengeID: first.ChallengeID, Code: "123456",
	}, LoginContext{})
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Fatalf("error = %v; ingin UNAUTHORIZED setelah tantangan hangus", err)
	}
}

// Tantangan kedaluwarsa ditolak meski kodenya benar.
func TestTantanganKedaluwarsaDitolak(t *testing.T) {
	f := newAdminLoginFixture(t)
	f.sealerGate.(*stubGate).enabled = true
	f.sealerGate.(*stubGate).code = "123456"
	ctx := context.Background()

	first, _ := f.svc.Login(ctx, dto.LoginRequest{
		Identifier: "admin@ila.id", Password: "rahasia-panjang-123",
	}, LoginContext{})

	// Majukan jam melewati umur tantangan.
	*f.clock = f.clock.Add(ChallengeTTL + time.Minute)

	if _, err := f.svc.VerifyTwoFactor(ctx, dto.VerifyTwoFactorRequest{
		ChallengeID: first.ChallengeID, Code: "123456",
	}, LoginContext{}); !errors.Is(err, apperror.ErrUnauthorized) {
		t.Fatalf("error = %v; ingin UNAUTHORIZED", err)
	}
}

// Tantangan pintu user tidak bisa dipakai di pintu admin.
func TestTantanganPurposeSalahDitolak(t *testing.T) {
	f := newAdminLoginFixture(t)
	f.sealerGate.(*stubGate).enabled = true
	f.sealerGate.(*stubGate).code = "123456"
	ctx := context.Background()

	first, _ := f.svc.Login(ctx, dto.LoginRequest{
		Identifier: "admin@ila.id", Password: "rahasia-panjang-123",
	}, LoginContext{})

	// Tandai tantangan sebagai milik pintu user.
	f.store.rows[first.ChallengeID].Purpose = PurposeUser

	if _, err := f.svc.VerifyTwoFactor(ctx, dto.VerifyTwoFactorRequest{
		ChallengeID: first.ChallengeID, Code: "123456",
	}, LoginContext{}); !errors.Is(err, apperror.ErrUnauthorized) {
		t.Fatalf("error = %v; ingin UNAUTHORIZED", err)
	}
}

// Challenge ID acak tidak bisa ditebak.
func TestChallengeIDTidakDikenalDitolak(t *testing.T) {
	f := newAdminLoginFixture(t)

	if _, err := f.svc.VerifyTwoFactor(context.Background(), dto.VerifyTwoFactorRequest{
		ChallengeID: "tidak-ada", Code: "123456",
	}, LoginContext{}); !errors.Is(err, apperror.ErrUnauthorized) {
		t.Fatalf("error = %v; ingin UNAUTHORIZED", err)
	}
}
