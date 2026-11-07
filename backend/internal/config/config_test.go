package config

import "testing"

func TestLegacyMasterKeysMenguraiBeberapaVersi(t *testing.T) {
	cfg := &Config{MasterKeysLegacy: "1:a2V5LWxhbWEtc2F0dQ==,2:a2V5LWxhbWEtZHVh"}

	got := cfg.LegacyMasterKeys()

	if len(got) != 2 {
		t.Fatalf("ingin 2 kunci, dapat %d", len(got))
	}
	if got[1] != "a2V5LWxhbWEtc2F0dQ==" {
		t.Fatalf("kunci versi 1 salah: %q", got[1])
	}
	if got[2] != "a2V5LWxhbWEtZHVh" {
		t.Fatalf("kunci versi 2 salah: %q", got[2])
	}
}

func TestLegacyMasterKeysKosong(t *testing.T) {
	if got := (&Config{}).LegacyMasterKeys(); got != nil {
		t.Fatalf("ingin nil, dapat %v", got)
	}
}

func TestLegacyMasterKeysMelewatiEntriRusak(t *testing.T) {
	cfg := &Config{MasterKeysLegacy: "rusak,abc:bukan-angka,3:a2V5LW9r"}

	got := cfg.LegacyMasterKeys()

	if len(got) != 1 {
		t.Fatalf("ingin 1 kunci sah, dapat %d (%v)", len(got), got)
	}
	if got[3] != "a2V5LW9r" {
		t.Fatalf("kunci versi 3 salah: %q", got[3])
	}
}
