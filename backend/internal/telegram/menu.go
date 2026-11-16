package telegram

import "strings"

// Prefix aksi pada callback_data.
//
// Telegram membatasi callback_data 64 byte, jadi setiap aksi memakai awalan
// pendek. Token approve berukuran 32 karakter, menyisakan ruang yang cukup
// untuk awalan dua karakter.
const (
	CallbackEksekusi = "a:"
	CallbackLewati   = "s:"
	CallbackMenu     = "m"
	CallbackPantau   = "p"
	// CallbackPantauHalaman membuka halaman tertentu daftar setup menunggu.
	CallbackPantauHalaman = "ph:"
	// CallbackPantauHalamanAksi adalah bentuk yang dikembalikan aksiCallback,
	// tanpa titik dua, karena handler membandingkan dengan nilai itu.
	CallbackPantauHalamanAksi = "ph"
	CallbackRiwayat           = "r:"
	CallbackDetail            = "d:"
	CallbackHariIni           = "h"
	CallbackPengaturan        = "g"
	CallbackSegarkan          = "px"
	CallbackHalaman           = "r:"
	CallbackBantuan           = "help"
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

// Label tombol inline pada halaman menu.
const (
	LabelPantau     = "📊 Pantau Saat Ini"
	LabelRiwayat    = "📋 Riwayat Posisi"
	LabelHariIni    = "📈 Ringkasan Hari Ini"
	LabelPengaturan = "⚙️ Pengaturan"
	LabelMenu       = "🏠 Menu"
	LabelSegarkan   = "🔄 Segarkan"
	// CabangDashboard adalah label tombol tautan ke dashboard web.
	CabangDashboard = "🌐 Buka Dashboard"
)

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

// aksiCallback memisahkan awalan dari isi callback_data.
func aksiCallback(data string) (string, string) {
	data = strings.TrimSpace(data)
	for _, prefix := range []string{CallbackEksekusi, CallbackLewati, CallbackRiwayat, CallbackPantauHalaman} {
		if strings.HasPrefix(data, prefix) {
			return strings.TrimSuffix(prefix, ":"), strings.TrimPrefix(data, prefix)
		}
	}
	return data, ""
}
