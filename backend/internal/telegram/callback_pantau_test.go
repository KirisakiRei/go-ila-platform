package telegram

import "testing"

// TestAksiHalamanPantauCocokDenganHandler mengunci bentuk aksi yang dihasilkan
// aksiCallback. Handler membandingkan dengan bentuk tanpa titik dua; kalau
// keduanya berbeda, tombol halaman tidak pernah menghasilkan apa pun.
func TestAksiHalamanPantauCocokDenganHandler(t *testing.T) {
	aksi, isi := aksiCallback(CallbackPantauHalaman + "3")

	if aksi != CallbackPantauHalamanAksi {
		t.Fatalf("aksi = %q, ingin %q", aksi, CallbackPantauHalamanAksi)
	}
	if isi != "3" {
		t.Fatalf("isi = %q, ingin 3", isi)
	}
}

// TestCallbackPantauTetapTunggal memastikan callback lama tidak ikut terurai
// sebagai aksi berawalan, karena nilainya tanpa titik dua.
func TestCallbackPantauTetapTunggal(t *testing.T) {
	aksi, isi := aksiCallback(CallbackPantau)

	if aksi != CallbackPantau || isi != "" {
		t.Fatalf("aksi/isi = %q/%q, ingin %q/kosong", aksi, isi, CallbackPantau)
	}
}
