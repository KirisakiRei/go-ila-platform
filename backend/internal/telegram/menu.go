package telegram

import "strings"

// Prefix aksi pada callback_data.
//
// Telegram membatasi callback_data 64 byte, jadi setiap aksi memakai awalan
// pendek. Token approve berukuran 32 karakter, menyisakan ruang yang cukup
// untuk awalan dua karakter.
const (
	CallbackEksekusi   = "a:"
	CallbackLewati     = "s:"
	CallbackMenu       = "m"
	CallbackPantau     = "p"
	CallbackRiwayat    = "r:"
	CallbackDetail     = "d:"
	CallbackHariIni    = "h"
	CallbackPengaturan = "g"
	CallbackSegarkan   = "px"
	CallbackHalaman    = "r:"
)

// Label tombol keyboard tetap. Nilainya juga dipakai webhook untuk mengenali
// menu mana yang dibuka, karena keyboard tetap mengirim teks biasa.
const (
	LabelPantau     = "📊 Pantau Saat Ini"
	LabelRiwayat    = "📋 Riwayat Posisi"
	LabelHariIni    = "📈 Ringkasan Hari Ini"
	LabelPengaturan = "⚙️ Pengaturan"
	LabelMenu       = "🏠 Menu"
)

// KeyboardUtama menyusun keyboard tetap yang selalu terlihat di bawah layar.
//
// Dua tombol per baris supaya teksnya tidak terpotong di layar sempit.
func KeyboardUtama() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]KeyboardButton{
			{{Text: LabelPantau}, {Text: LabelRiwayat}},
			{{Text: LabelHariIni}, {Text: LabelPengaturan}},
		},
		ResizeKeyboard: true,
		IsPersistent:   true,
	}
}

// AksiDariTeks mengenali teks tombol keyboard tetap dan mengubahnya menjadi
// aksi callback yang sama.
func AksiDariTeks(text string) (string, bool) {
	switch strings.TrimSpace(text) {
	case LabelPantau:
		return CallbackPantau, true
	case LabelRiwayat:
		return CallbackRiwayat + "0", true
	case LabelHariIni:
		return CallbackHariIni, true
	case LabelPengaturan:
		return CallbackPengaturan, true
	case LabelMenu:
		return CallbackMenu, true
	}
	return "", false
}

// aksiCallback memisahkan awalan dari isi callback_data.
func aksiCallback(data string) (string, string) {
	data = strings.TrimSpace(data)
	for _, prefix := range []string{CallbackEksekusi, CallbackLewati, CallbackRiwayat} {
		if strings.HasPrefix(data, prefix) {
			return strings.TrimSuffix(prefix, ":"), strings.TrimPrefix(data, prefix)
		}
	}
	return data, ""
}
