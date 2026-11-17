package httpx

// Pengujian penerjemah galat validasi.
//
// Sasaran: pengguna tidak pernah membaca nama struct Go, nama tag validator,
// atau istilah internal seperti "invalid request payload" di layar.

import (
	"strings"
	"testing"
)

// Keluaran validator diterjemahkan menjadi kalimat biasa.
func TestPesanValidasiMenerjemahkanTag(t *testing.T) {
	kasus := []struct {
		nama     string
		masukan  string
		harusAda string
	}{
		{
			nama:     "alphanum",
			masukan:  "Key: 'RegisterRequest.Username' Error:Field validation for 'Username' failed on the 'alphanum' tag",
			harusAda: "username hanya boleh huruf dan angka",
		},
		{
			nama:     "required",
			masukan:  "Key: 'RegisterRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag",
			harusAda: "email wajib diisi",
		},
		{
			nama:     "email",
			masukan:  "Key: 'RegisterRequest.Email' Error:Field validation for 'Email' failed on the 'email' tag",
			harusAda: "format email tidak sah",
		},
		{
			nama:     "min",
			masukan:  "Key: 'RegisterRequest.Password' Error:Field validation for 'Password' failed on the 'min' tag",
			harusAda: "password terlalu pendek",
		},
		{
			nama:     "field bersarang",
			masukan:  "Key: 'A.B.Phone' Error:Field validation for 'Phone' failed on the 'max' tag",
			harusAda: "nomor telepon terlalu panjang",
		},
		{
			nama:     "aturan belum dikenal",
			masukan:  "Key: 'X.Username' Error:Field validation for 'Username' failed on the 'tidakdikenal' tag",
			harusAda: "username tidak sah",
		},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			hasil := PesanValidasi(k.masukan)
			if !strings.Contains(hasil, k.harusAda) {
				t.Fatalf("harapan memuat %q, dapat %q", k.harusAda, hasil)
			}
			// Tidak boleh ada sisa istilah internal.
			for _, terlarang := range []string{"Key:", "Field validation", "tag", "RegisterRequest"} {
				if strings.Contains(hasil, terlarang) {
					t.Errorf("masih memuat %q: %q", terlarang, hasil)
				}
			}
		})
	}
}

// Beberapa galat sekaligus digabung, bukan hanya yang pertama.
func TestPesanValidasiBeberapaGalat(t *testing.T) {
	masukan := "Key: 'R.Name' Error:Field validation for 'Name' failed on the 'required' tag\n" +
		"Key: 'R.Email' Error:Field validation for 'Email' failed on the 'email' tag"

	hasil := PesanValidasi(masukan)
	if !strings.Contains(hasil, "nama wajib diisi") || !strings.Contains(hasil, "format email tidak sah") {
		t.Fatalf("kedua galat harus muncul, dapat %q", hasil)
	}
}

// Pesan yang sudah ditulis manusia TIDAK diubah. Ini menjaga 13+ pemanggil
// WrapValidation yang mengirim kalimat sendiri.
func TestPesanValidasiTidakMengubahPesanManusia(t *testing.T) {
	pesan := "token ini sudah dipakai akun broker lain; setiap akun memakai token berbeda"
	if hasil := PesanValidasi(pesan); hasil != pesan {
		t.Fatalf("pesan manusia berubah: %q", hasil)
	}
}

// WrapValidation tidak boleh lagi menghasilkan teks mentah validator.
func TestWrapValidationTidakBocorIstilahInternal(t *testing.T) {
	err := WrapValidation("Key: 'RegisterRequest.Username' Error:Field validation for 'Username' failed on the 'alphanum' tag")
	if strings.Contains(err.Error(), "Field validation") {
		t.Fatalf("pesan mentah validator masih bocor: %q", err.Error())
	}
	if strings.Contains(err.Error(), "alphanum") {
		t.Fatalf("nama tag validator masih bocor: %q", err.Error())
	}
}

// Pesan kosong tidak memicu panic.
func TestPesanValidasiKosong(t *testing.T) {
	if hasil := PesanValidasi(""); hasil != "" {
		t.Fatalf("pesan kosong harus tetap kosong, dapat %q", hasil)
	}
}
