package httpx

// Penerjemah galat validasi.
//
// Latar belakang: handler memanggil c.ShouldBindJSON lalu meneruskan
// err.Error() ke WrapValidation. Pesan validator mentah berbentuk
//
//	Key: 'RegisterRequest.Username' Error:Field validation for 'Username'
//	failed on the 'alphanum' tag
//
// dan itu ikut terkirim ke layar pengguna. Pesan seperti itu tidak dapat
// ditindaklanjuti: ia menyebut nama struct Go dan tag validator, bukan apa
// yang perlu diperbaiki pengguna.
//
// Fungsi di sini menerjemahkan pesan tersebut menjadi kalimat biasa. Pesan
// yang sudah ditulis manusia (mis. "token ini sudah dipakai akun broker lain")
// TIDAK diubah, karena pemanggil WrapValidation memang boleh mengirim pesan
// sendiri yang sudah rapi.

import (
	"regexp"
	"strings"
)

// polaValidator mengenali keluaran validator go-playground. Contoh yang
// ditangkap:
//
//	Key: 'A.B.Username' Error:Field validation for 'Username' failed on the 'required' tag
//
// Regex sengaja longgar terhadap nama key agar tetap jalan meski ada struct
// bersarang (dotted key), sekaligus cukup ketat untuk tidak menelan pesan
// manusia biasa.
var polaValidator = regexp.MustCompile(
	`Key: '[^']*' Error:Field validation for '([^']*)' failed on the '([^']*)' tag`,
)

// labelKolom memetakan nama field Go menjadi sebutan yang dikenal pengguna.
// Field yang tidak terdaftar memakai namanya apa adanya (huruf kecil).
var labelKolom = map[string]string{
	"Name":        "nama",
	"Username":    "username",
	"Email":       "email",
	"Phone":       "nomor telepon",
	"Password":    "password",
	"AcceptTerms": "persetujuan syarat dan ketentuan",
	"InviteCode":  "kode undangan",
	"Identifier":  "email atau username",
	"Reason":      "alasan",
	"Tier":        "paket langganan",
	"Code":        "kode",
	"Token":       "token",
}

// pesanAturan menerjemahkan tag validator menjadi kalimat yang bisa dibaca.
// %s diisi nama kolom.
var pesanAturan = map[string]string{
	"required": "%s wajib diisi",
	"min":      "%s terlalu pendek",
	"max":      "%s terlalu panjang",
	"len":      "panjang %s tidak sesuai",
	"email":    "format %s tidak sah",
	"alphanum": "%s hanya boleh huruf dan angka",
	"numeric":  "%s hanya boleh angka",
	"uuid":     "%s tidak sah",
	"oneof":    "pilihan %s tidak dikenal",
	"url":      "alamat %s tidak sah",
	"gt":       "%s harus lebih besar dari batas minimum",
	"gte":      "%s harus lebih besar atau sama dengan batas minimum",
	"lt":       "%s harus lebih kecil dari batas maksimum",
	"lte":      "%s harus lebih kecil atau sama dengan batas maksimum",
	"boolean":  "%s harus bernilai benar atau salah",
	"datetime": "format tanggal %s tidak sah",
	"eqfield":  "%s harus sama dengan kolom pembanding",
	"nefield":  "%s tidak boleh sama dengan kolom pembanding",
}

// PesanValidasi mengubah pesan galat validator menjadi kalimat yang dapat
// dibaca pengguna. Pesan yang bukan keluaran validator dikembalikan apa adanya.
func PesanValidasi(pesan string) string {
	if pesan == "" {
		return ""
	}
	if !polaValidator.MatchString(pesan) {
		return pesan
	}

	// Satu payload bisa memuat beberapa galat sekaligus; terjemahkan tiap
	// kemunculan, lalu gabungkan supaya pengguna tahu semua yang perlu dibetulkan.
	pertemuan := polaValidator.FindAllStringSubmatch(pesan, -1)
	hasil := make([]string, 0, len(pertemuan))
	for _, m := range pertemuan {
		kolom := labelKolomUntuk(m[1])
		aturan := m[2]
		if kalimat, ada := pesanAturan[aturan]; ada {
			hasil = append(hasil, sprintfSederhana(kalimat, kolom))
			continue
		}
		// Aturan yang belum diterjemahkan: sebut kolomnya saja tanpa
		// membocorkan nama tag validator.
		hasil = append(hasil, kolom+" tidak sah")
	}

	if len(hasil) == 0 {
		return pesan
	}
	return strings.Join(hasil, "; ")
}

// labelKolomUntuk mengembalikan sebutan kolom yang ramah pengguna.
func labelKolomUntuk(field string) string {
	if label, ada := labelKolom[field]; ada {
		return label
	}
	return strings.ToLower(field)
}

// sprintfSederhana mengganti satu penanda %s tanpa menarik paket fmt, karena
// pola kalimat di pesanAturan hanya memuat satu penanda.
func sprintfSederhana(kalimat, isi string) string {
	return strings.Replace(kalimat, "%s", isi, 1)
}
