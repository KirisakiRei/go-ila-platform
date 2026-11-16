package telegram

import (
	"fmt"
	"strings"
	"time"
)

// BarisRiwayat adalah satu posisi pada daftar riwayat.
type BarisRiwayat struct {
	PositionID string
	Symbol     string
	Direction  string
	OutcomeR   float64
	ProfitLoss float64
	ClosedAt   time.Time
}

// RiwayatParams memuat satu halaman riwayat posisi.
type RiwayatParams struct {
	Baris        []BarisRiwayat
	Halaman      int
	TotalHalaman int
	TotalPosisi  int
}

// FormatRiwayatMessage menyusun satu halaman riwayat posisi.
//
// Lima posisi per halaman: cukup untuk melihat pola, tidak sampai harus
// menggulir jauh di layar ponsel.
func FormatRiwayatMessage(p RiwayatParams) Message {
	var b strings.Builder

	if p.TotalPosisi == 0 {
		b.WriteString("📋 Riwayat Posisi\n\nBelum ada posisi yang tercatat.\n")
		b.WriteString("Setiap entry yang diambil akan tampil di sini.\n")
		return Message{Text: b.String(), Buttons: tombolRiwayat(0, 0)}
	}

	fmt.Fprintf(&b, "📋 Riwayat Posisi\nHalaman %d dari %d · %d posisi\n", p.Halaman, p.TotalHalaman, p.TotalPosisi)

	hariTerakhir := ""
	for _, row := range p.Baris {
		hari := row.ClosedAt.In(time.FixedZone("WIB", 7*3600)).Format("02 Jan")
		if hari != hariTerakhir {
			fmt.Fprintf(&b, "\n%s\n", hari)
			hariTerakhir = hari
		}
		jam := row.ClosedAt.In(time.FixedZone("WIB", 7*3600)).Format("15:04")
		fmt.Fprintf(&b, "%s %s %s · %s · %s\n",
			simbolHasil(row.OutcomeR),
			arahTampil(row.Direction),
			row.Symbol,
			hasilR(row.OutcomeR),
			jam,
		)
	}

	return Message{Text: b.String(), Buttons: tombolRiwayat(p.Halaman, p.TotalHalaman)}
}

// tombolRiwayat menyusun navigasi halaman.
//
// Tombol yang tidak mungkin ditekan dikirim sebagai tombol tanpa aksi, supaya
// tata letaknya tidak bergeser antar halaman.
func tombolRiwayat(halaman, total int) [][]InlineButton {
	if total <= 1 {
		return [][]InlineButton{{{Text: LabelMenu, CallbackData: CallbackMenu}}}
	}

	sebelum := InlineButton{Text: "‹ Sebelumnya"}
	if halaman > 1 {
		sebelum.CallbackData = fmt.Sprintf("%s%d", CallbackRiwayat, (halaman-2)*RiwayatPerHalaman)
	}

	berikut := InlineButton{Text: "Berikutnya ›"}
	if halaman < total {
		berikut.CallbackData = fmt.Sprintf("%s%d", CallbackRiwayat, halaman*RiwayatPerHalaman)
	}

	return [][]InlineButton{
		{sebelum, {Text: fmt.Sprintf("%d/%d", halaman, total), CallbackData: CallbackMenu}, berikut},
		{{Text: LabelMenu, CallbackData: CallbackMenu}},
	}
}

// RiwayatPerHalaman adalah jumlah posisi pada satu halaman riwayat.
const RiwayatPerHalaman = 5

// HariParams memuat ringkasan satu hari dan bulan berjalan.
type HariParams struct {
	Hari          string
	TradeTerpakai int
	TradeMaks     int
	Win           int
	Loss          int
	ProfitLoss    float64
	PosisiJalan   int
	BulanTotal    int
	BulanWinRate  float64
	BulanNet      float64
	DashboardLink string
}

// FormatHariMessage menyusun ringkasan hari ini.
func FormatHariMessage(p HariParams) Message {
	var b strings.Builder

	b.WriteString("📈 Ringkasan Hari Ini")
	if p.Hari != "" {
		b.WriteString(" · " + p.Hari)
	}
	b.WriteString("\n\n")
	b.WriteString(daftar([][2]string{
		{"Trade", kuotaTampil(p.TradeTerpakai, p.TradeMaks)},
		{"Menang", itoaOpsional(p.Win)},
		{"Kalah", itoaOpsional(p.Loss)},
		{"Untung/Rugi", formatProfitHarian(p.ProfitLoss)},
		{"Posisi jalan", itoaOpsional(p.PosisiJalan)},
	}))
	if p.BulanTotal > 0 {
		b.WriteString("\nBulan ini\n")
		b.WriteString(daftar([][2]string{
			{"Total posisi", fmt.Sprintf("%d", p.BulanTotal)},
			{"Rasio menang", fmt.Sprintf("%s%%", formatFrom(p.BulanWinRate))},
			{"Untung/Rugi", formatMoney(p.BulanNet)},
		}))
	}

	var tombol [][]InlineButton
	tombol = append(tombol, []InlineButton{
		{Text: LabelRiwayat, CallbackData: CallbackRiwayat + "0"},
		{Text: LabelMenu, CallbackData: CallbackMenu},
	})
	if p.DashboardLink != "" {
		tombol = append(tombol, []InlineButton{{Text: "Buka di dashboard", URL: p.DashboardLink}})
	}

	return Message{Text: b.String(), Buttons: tombol}
}

// FormatPengaturanMessage menjelaskan apa yang bisa dan tidak bisa diatur
// dari Telegram.
//
// Kalimatnya sengaja menjelaskan alasannya, bukan sekadar melarang. Trader
// perlu tahu bahwa urusan akun tetap aman di tempat yang semestinya.
func FormatPengaturanMessage(dashboardLink string) Message {
	var b strings.Builder

	b.WriteString("⚙️ PENGATURAN\n\n")
	b.WriteString("Sebagian besar pengaturan tetap dikelola di website, karena menyangkut akun dan uang Anda.\n\n")
	b.WriteString("Yang bisa Anda atur di website\n")
	b.WriteString(daftar([][2]string{
		{"Akun broker", "kredensial dan koneksi"},
		{"Risiko", "per transaksi dan ukuran lot"},
		{"Pantauan", "pair dan timeframe"},
		{"Mode", "otomatis atau manual"},
		{"Langganan", "notifikasi dan paket"},
	}))
	b.WriteString("\nLewat Telegram Anda cukup memantau dan memutuskan entry.\n")

	var tombol [][]InlineButton
	if dashboardLink != "" {
		tombol = append(tombol, []InlineButton{{Text: "Buka Pengaturan", URL: dashboardLink}})
	}
	tombol = append(tombol, []InlineButton{{Text: LabelMenu, CallbackData: CallbackMenu}})
	return Message{Text: b.String(), Buttons: tombol}
}

// FormatMenuMessage menyusun halaman menu utama.
func FormatMenuMessage(nama, paket string) Message {
	var b strings.Builder

	b.WriteString("🤖 BOT Trading ILA\n")
	if nama != "" {
		b.WriteString(nama)
		if paket != "" {
			b.WriteString(" · " + paket)
		}
		b.WriteString("\n")
	}
	b.WriteString("\nPilih yang ingin Anda lihat.\n")
	return Message{Text: b.String(), Buttons: [][]InlineButton{
		{{Text: LabelPantau, CallbackData: CallbackPantau}, {Text: LabelRiwayat, CallbackData: CallbackRiwayat + "0"}},
		{{Text: LabelHariIni, CallbackData: CallbackHariIni}, {Text: LabelPengaturan, CallbackData: CallbackPengaturan}},
	}}
}

// itoaOpsional menampilkan angka atau tanda hubung bila nol.
func itoaOpsional(v int) string {
	if v <= 0 {
		return "—"
	}
	return fmt.Sprintf("%d", v)
}
