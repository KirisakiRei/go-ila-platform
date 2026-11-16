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
	CallbackBantuan    = "help"
)

// Perintah yang didaftarkan ke Telegram agar muncul di menu "/".
const (
	CommandMulai      = "start"
	CommandMenu       = "menu"
	CommandPantau     = "pantau"
	CommandRiwayat    = "riwayat"
	CommandHari       = "hari"
	CommandPengaturan = "pengaturan"
	CommandBantuan    = "bantuan"
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

// ReplyKeyboard adalah keyboard tetap di bawah layar chat.
//
// Keyboard menempel pada pesan, bukan pada chat: setiap balasan bot perlu
// membawanya ulang supaya tombolnya tetap terlihat. Pengguna tetap bisa
// menyembunyikannya lewat tombol tutup bawaan Telegram.
func KeyboardUtama() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]KeyboardButton{
			{{Text: LabelPantau}, {Text: LabelRiwayat}},
			{{Text: LabelHariIni}, {Text: LabelPengaturan}},
		},
		ResizeKeyboard: true,
		IsPersistent:   true,
		// Tanpa placeholder, kolom ketik tetap muncul di atas keyboard.
		InputFieldPlaceholder: "Tulis pesan, atau pilih menu di bawah",
	}
}

// PerintahTelegram adalah daftar perintah yang muncul saat pengguna mengetik
// "/" di kolom pesan.
//
// Deskripsinya pendek karena Telegram memotong teks panjang di daftar.
func PerintahTelegram() []BotCommand {
	return []BotCommand{
		{Command: CommandMenu, Description: "Buka menu utama"},
		{Command: CommandPantau, Description: "Posisi berjalan dan setup menunggu"},
		{Command: CommandRiwayat, Description: "Riwayat posisi yang sudah tutup"},
		{Command: CommandHari, Description: "Ringkasan hari ini"},
		{Command: CommandPengaturan, Description: "Apa saja yang diatur lewat website"},
		{Command: CommandBantuan, Description: "Panduan singkat pemakaian"},
	}
}

// AksiDariPerintah mengubah perintah chat menjadi aksi menu yang sama.
func AksiDariPerintah(text string) (string, bool) {
	perintah, _, _ := strings.Cut(strings.TrimSpace(text), " ")
	perintah = strings.TrimPrefix(perintah, "/")
	// Telegram mengirim "/pantau@NamaBot" di grup.
	perintah, _, _ = strings.Cut(perintah, "@")
	switch strings.ToLower(perintah) {
	case CommandMenu:
		return CallbackMenu, true
	case CommandPantau:
		return CallbackPantau, true
	case CommandRiwayat:
		return CallbackRiwayat + "0", true
	case CommandHari:
		return CallbackHariIni, true
	case CommandPengaturan:
		return CallbackPengaturan, true
	case CommandBantuan:
		return CallbackBantuan, true
	}
	return "", false
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
