package totp

import (
	"testing"
	"time"
)

// Vektor uji dari RFC 4226 Lampiran D: secret "12345678901234567890" dalam
// base32, dihitung dengan HMAC-SHA1.
const rfcSecret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

func TestHOTPVektorRFC4226(t *testing.T) {
	cases := []struct {
		counter uint64
		code    string
	}{
		{0, "755224"},
		{1, "287082"},
		{2, "359152"},
		{3, "969429"},
		{4, "338314"},
		{5, "254676"},
		{6, "287922"},
		{7, "162583"},
		{8, "399871"},
		{9, "520489"},
	}

	for _, c := range cases {
		got, err := hotp(rfcSecret, c.counter)
		if err != nil {
			t.Fatalf("hotp(%d) error: %v", c.counter, err)
		}
		if got != c.code {
			t.Errorf("hotp(%d) = %s; ingin %s", c.counter, got, c.code)
		}
	}
}

func TestValidateMenerimaKodeSaatIni(t *testing.T) {
	at := time.Unix(1_700_000_000, 0).UTC()
	code, err := hotp(rfcSecret, uint64(Step(at)))
	if err != nil {
		t.Fatalf("hotp error: %v", err)
	}

	ok, err := Validate(rfcSecret, code, at)
	if err != nil {
		t.Fatalf("Validate error: %v", err)
	}
	if !ok {
		t.Fatal("kode pada periode ini seharusnya diterima")
	}
}

func TestValidateToleransiSatuPeriode(t *testing.T) {
	at := time.Unix(1_700_000_000, 0).UTC()
	previous, _ := hotp(rfcSecret, uint64(Step(at)-1))
	next, _ := hotp(rfcSecret, uint64(Step(at)+1))

	if ok, _ := Validate(rfcSecret, previous, at); !ok {
		t.Fatal("kode periode sebelumnya seharusnya diterima")
	}
	if ok, _ := Validate(rfcSecret, next, at); !ok {
		t.Fatal("kode periode berikutnya seharusnya diterima")
	}
}

func TestValidateMenolakDiLuarToleransi(t *testing.T) {
	at := time.Unix(1_700_000_000, 0).UTC()
	tooOld, _ := hotp(rfcSecret, uint64(Step(at)-2))
	tooNew, _ := hotp(rfcSecret, uint64(Step(at)+2))

	if ok, _ := Validate(rfcSecret, tooOld, at); ok {
		t.Fatal("kode dua periode lalu seharusnya ditolak")
	}
	if ok, _ := Validate(rfcSecret, tooNew, at); ok {
		t.Fatal("kode dua periode ke depan seharusnya ditolak")
	}
}

func TestValidateMenolakBentukSalah(t *testing.T) {
	at := time.Unix(1_700_000_000, 0).UTC()

	for _, code := range []string{"", "12345", "1234567", "abcdef", "12 456", "-12345"} {
		ok, err := Validate(rfcSecret, code, at)
		if err != nil {
			t.Fatalf("Validate(%q) error: %v", code, err)
		}
		if ok {
			t.Errorf("Validate(%q) seharusnya false", code)
		}
	}
}

func TestNewSecretPanjangDanUnik(t *testing.T) {
	first, err := NewSecret()
	if err != nil {
		t.Fatalf("NewSecret error: %v", err)
	}
	second, err := NewSecret()
	if err != nil {
		t.Fatalf("NewSecret error: %v", err)
	}

	if len(first) != 32 {
		t.Fatalf("panjang secret = %d; ingin 32 (160 bit base32)", len(first))
	}
	if first == second {
		t.Fatal("dua secret tidak boleh sama")
	}

	// Secret harus bisa dipakai menghitung kode.
	at := time.Unix(1_700_000_000, 0).UTC()
	code, err := hotp(first, uint64(Step(at)))
	if err != nil {
		t.Fatalf("hotp error: %v", err)
	}
	if ok, _ := Validate(first, code, at); !ok {
		t.Fatal("secret baru seharusnya bisa dipakai")
	}
}

func TestProvisioningURL(t *testing.T) {
	got := ProvisioningURL("ABCDEF234567", "admin@ila.id", "")
	want := "otpauth://totp/Imaginary%20Liquidity:admin@ila.id"
	if len(got) < len(want) || got[:len(want)] != want {
		t.Fatalf("URL = %s; ingin berawalan %s", got, want)
	}
	for _, part := range []string{"secret=ABCDEF234567", "digits=6", "period=30", "algorithm=SHA1"} {
		if !contains(got, part) {
			t.Errorf("URL tidak memuat %q: %s", part, got)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
