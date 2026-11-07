package service

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
)

func TestStatusAwalNonaktif(t *testing.T) {
	f := newTwoFactorFixture(t)

	status, err := f.svc.Status(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if status.Enabled {
		t.Fatal("akun baru seharusnya belum mengaktifkan 2FA")
	}
}

// Alur lengkap: setup → kode dari aplikasi → aktif + 8 kode pemulihan.
func TestAktivasiBerhasilDanMemberiDelapanKode(t *testing.T) {
	f := newTwoFactorFixture(t)
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	*f.clock = now

	setup, err := f.svc.Setup(ctx, "u1", "user@ila.id")
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}
	if setup.Secret == "" || setup.ProvisioningURL == "" {
		t.Fatal("setup tidak memberi secret atau tautan otentikator")
	}

	// Sebelum aktivasi, 2FA belum berlaku.
	status, _ := f.svc.Status(ctx, "u1")
	if status.Enabled {
		t.Fatal("2FA seharusnya belum aktif sebelum kode diverifikasi")
	}

	codes, err := f.svc.Activate(ctx, "u1", codeFor(t, setup.Secret, now))
	if err != nil {
		t.Fatalf("Activate error: %v", err)
	}
	if len(codes) != RecoveryCodeCount {
		t.Fatalf("kode pemulihan = %d; ingin %d", len(codes), RecoveryCodeCount)
	}

	status, _ = f.svc.Status(ctx, "u1")
	if !status.Enabled {
		t.Fatal("2FA seharusnya aktif setelah aktivasi")
	}
	if status.RecoveryRemaining != RecoveryCodeCount {
		t.Fatalf("kode pemulihan tersisa = %d; ingin %d", status.RecoveryRemaining, RecoveryCodeCount)
	}
}

// Kode salah tidak boleh mengaktifkan 2FA.
func TestAktivasiKodeSalahDitolak(t *testing.T) {
	f := newTwoFactorFixture(t)
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	*f.clock = now

	if _, err := f.svc.Setup(ctx, "u1", "user@ila.id"); err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	if _, err := f.svc.Activate(ctx, "u1", "000000"); err == nil {
		t.Fatal("kode salah seharusnya ditolak")
	}

	status, _ := f.svc.Status(ctx, "u1")
	if status.Enabled {
		t.Fatal("2FA tidak boleh aktif setelah kegagalan")
	}
}

// Aktivasi tanpa penyiapan lebih dulu ditolak.
func TestAktivasiTanpaSetupDitolak(t *testing.T) {
	f := newTwoFactorFixture(t)

	if _, err := f.svc.Activate(context.Background(), "u1", "123456"); err == nil {
		t.Fatal("aktivasi tanpa penyiapan seharusnya ditolak")
	}
}

// Kode yang sama tidak boleh dipakai dua kali (replay).
func TestVerifyMenolakPemakaianUlangKode(t *testing.T) {
	f := newTwoFactorFixture(t)
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	*f.clock = now

	setup, _ := f.svc.Setup(ctx, "u1", "user@ila.id")
	code := codeFor(t, setup.Secret, now)
	if _, err := f.svc.Activate(ctx, "u1", code); err != nil {
		t.Fatalf("Activate error: %v", err)
	}

	// Kode yang dipakai saat aktivasi tidak boleh dipakai lagi.
	if err := f.svc.Verify(ctx, "u1", code); err == nil {
		t.Fatal("kode yang sudah dipakai seharusnya ditolak")
	}

	// Kode periode berikutnya diterima.
	*f.clock = now.Add(30 * time.Second)
	next := codeFor(t, setup.Secret, *f.clock)
	if err := f.svc.Verify(ctx, "u1", next); err != nil {
		t.Fatalf("kode periode berikutnya seharusnya diterima: %v", err)
	}
}

// Kode pemulihan hanya bisa dipakai sekali.
func TestKodePemulihanSekaliPakai(t *testing.T) {
	f := newTwoFactorFixture(t)
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	*f.clock = now

	setup, _ := f.svc.Setup(ctx, "u1", "user@ila.id")
	codes, err := f.svc.Activate(ctx, "u1", codeFor(t, setup.Secret, now))
	if err != nil {
		t.Fatalf("Activate error: %v", err)
	}

	used, err := f.svc.ConsumeRecovery(ctx, "u1", codes[0])
	if err != nil || !used {
		t.Fatalf("pemakaian pertama = %v, %v; ingin true", used, err)
	}

	used, err = f.svc.ConsumeRecovery(ctx, "u1", codes[0])
	if err != nil {
		t.Fatalf("ConsumeRecovery error: %v", err)
	}
	if used {
		t.Fatal("kode pemulihan yang sama tidak boleh dipakai dua kali")
	}
}

// Kode pemulihan dinormalkan: huruf besar dan tanpa tanda hubung tetap sah.
func TestKodePemulihanFleksibelPenulisan(t *testing.T) {
	f := newTwoFactorFixture(t)
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	*f.clock = now

	setup, _ := f.svc.Setup(ctx, "u1", "user@ila.id")
	codes, _ := f.svc.Activate(ctx, "u1", codeFor(t, setup.Secret, now))

	// Pemakaian dengan huruf besar dan tanpa tanda hubung.
	plain := codes[1]
	var bare []byte
	for i := 0; i < len(plain); i++ {
		if plain[i] == '-' {
			continue
		}
		c := plain[i]
		if c >= 'a' && c <= 'z' {
			c = c - 'a' + 'A'
		}
		bare = append(bare, c)
	}

	used, err := f.svc.ConsumeRecovery(ctx, "u1", string(bare))
	if err != nil {
		t.Fatalf("ConsumeRecovery error: %v", err)
	}
	if !used {
		t.Fatal("kode pemulihan seharusnya tetap sah meski ditulis berbeda")
	}
}

// Menonaktifkan 2FA wajib membuktikan kode.
func TestDisableWajibKodeSah(t *testing.T) {
	f := newTwoFactorFixture(t)
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	*f.clock = now

	setup, _ := f.svc.Setup(ctx, "u1", "user@ila.id")
	if _, err := f.svc.Activate(ctx, "u1", codeFor(t, setup.Secret, now)); err != nil {
		t.Fatalf("Activate error: %v", err)
	}

	if err := f.svc.Disable(ctx, "u1", "000000"); err == nil {
		t.Fatal("penonaktifan dengan kode salah seharusnya ditolak")
	}
	if status, _ := f.svc.Status(ctx, "u1"); !status.Enabled {
		t.Fatal("2FA tidak boleh mati karena percobaan gagal")
	}

	*f.clock = now.Add(30 * time.Second)
	if err := f.svc.Disable(ctx, "u1", codeFor(t, setup.Secret, *f.clock)); err != nil {
		t.Fatalf("Disable error: %v", err)
	}
	if status, _ := f.svc.Status(ctx, "u1"); status.Enabled {
		t.Fatal("2FA seharusnya mati setelah kode sah")
	}
}

// Membuat ulang kode pemulihan menghanguskan kode lama.
func TestRegenerateRecoveryMenghanguskanKodeLama(t *testing.T) {
	f := newTwoFactorFixture(t)
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	*f.clock = now

	setup, _ := f.svc.Setup(ctx, "u1", "user@ila.id")
	oldCodes, _ := f.svc.Activate(ctx, "u1", codeFor(t, setup.Secret, now))

	*f.clock = now.Add(30 * time.Second)
	newCodes, err := f.svc.RegenerateRecovery(ctx, "u1", codeFor(t, setup.Secret, *f.clock))
	if err != nil {
		t.Fatalf("RegenerateRecovery error: %v", err)
	}
	if len(newCodes) != RecoveryCodeCount {
		t.Fatalf("kode baru = %d; ingin %d", len(newCodes), RecoveryCodeCount)
	}

	used, _ := f.svc.ConsumeRecovery(ctx, "u1", oldCodes[0])
	if used {
		t.Fatal("kode pemulihan lama seharusnya hangus")
	}

	used, _ = f.svc.ConsumeRecovery(ctx, "u1", newCodes[0])
	if !used {
		t.Fatal("kode pemulihan baru seharusnya sah")
	}
}

// Secret disimpan terenkripsi: bocornya database tidak membocorkan secret.
func TestSecretTersimpanTerenkripsi(t *testing.T) {
	f := newTwoFactorFixture(t)
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	*f.clock = now

	setup, _ := f.svc.Setup(ctx, "u1", "user@ila.id")

	stored := f.store.settings["u1"]
	if stored == nil {
		t.Fatal("pengaturan 2FA tidak tersimpan")
	}
	if stored.SecretEnc == setup.Secret {
		t.Fatal("secret tersimpan dalam bentuk polos")
	}
	if len(stored.SecretEnc) == 0 {
		t.Fatal("secret tersimpan kosong")
	}
}

// Secret satu akun tidak bisa dibuka sebagai milik akun lain.
func TestSecretTerikatPemilik(t *testing.T) {
	f := newTwoFactorFixture(t)
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	*f.clock = now

	if _, err := f.svc.Setup(ctx, "u1", "satu@ila.id"); err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	// Ciphertext milik u1 dipindahkan menjadi milik u2 tanpa kunci u1.
	stolen := f.store.settings["u1"].SecretEnc
	f.store.settings["u2"] = &model.TwoFactor{
		UserID:        "u2",
		SecretEnc:     stolen,
		SecretVersion: 1,
	}

	if _, err := f.svc.Activate(ctx, "u2", "123456"); err == nil {
		t.Fatal("secret milik akun lain seharusnya tidak bisa dipakai")
	}
}
