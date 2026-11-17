package telegram

import (
	"fmt"
	"strings"
	"time"
)

// BarisRiwayat adalah satu posisi pada daftar riwayat.
type BarisRiwayat struct {
	PositionID string
	Status     string
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
		hari := row.ClosedAt.In(time.FixedZone("WIB", 7*3600)).Format("02 Jan 2006")
		if hari != hariTerakhir {
			fmt.Fprintf(&b, "\n%s\n", hari)
			hariTerakhir = hari
		}
		jam := row.ClosedAt.In(time.FixedZone("WIB", 7*3600)).Format("15:04 WIB")
		fmt.Fprintf(&b, "%s %s %s · %s · %s\n",
			simbolHasil(row.OutcomeR),
			arahTampil(row.Direction),
			row.Symbol,
			hasilR(row.OutcomeR),
			jam,
		)
	}

	buttons := tombolRiwayat(p.Halaman, p.TotalHalaman)
	for i, row := range p.Baris {
		if row.PositionID != "" {
			buttons = append(buttons, []InlineButton{{Text: fmt.Sprintf("%d · %s", i+1, row.Symbol), CallbackData: "view:history:" + row.PositionID}})
		}
	}
	return Message{Text: b.String(), Buttons: buttons}
}

// tombolRiwayat menyusun navigasi halaman.
//
// Boundary buttons acknowledge a no-op; Telegram does not allow actionless buttons.
func tombolRiwayat(halaman, total int) [][]InlineButton {
	if total <= 1 {
		return [][]InlineButton{{{Text: LabelMenu, CallbackData: CallbackMenu}}}
	}

	sebelum := InlineButton{Text: "‹ Sebelumnya", CallbackData: CallbackNoop}
	if halaman > 1 {
		sebelum.CallbackData = fmt.Sprintf("%s%d", CallbackRiwayat, (halaman-2)*RiwayatPerHalaman)
	}

	berikut := InlineButton{Text: "Berikutnya ›", CallbackData: CallbackNoop}
	if halaman < total {
		berikut.CallbackData = fmt.Sprintf("%s%d", CallbackRiwayat, halaman*RiwayatPerHalaman)
	}

	return [][]InlineButton{
		{sebelum, {Text: "Kembali", CallbackData: CallbackRiwayatMenu}, berikut},
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
	UpdatedAt     time.Time
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
		{"PnL hari ini", "Belum tersedia"},
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

	b.WriteString("\nPeriode harian mengikuti jadwal reset trading akun, bukan pergantian hari WIB.\n")
	if !p.UpdatedAt.IsZero() {
		b.WriteString("Diperbarui: " + p.UpdatedAt.In(time.FixedZone("WIB", 7*3600)).Format("02 Jan 2006, 15:04 WIB") + "\n")
	}
	var tombol [][]InlineButton
	tombol = append(tombol, []InlineButton{
		{Text: LabelRiwayat, CallbackData: CallbackRiwayat + "0", Style: "primary"},
		{Text: LabelMenu, CallbackData: CallbackMenu},
	})
	tombol = append(tombol, []InlineButton{{Text: "Pantau", CallbackData: CallbackPantau}, {Text: LabelSegarkan, CallbackData: CallbackHariIni}})
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
	buttons := [][]InlineButton{}
	if dashboardLink != "" {
		buttons = append(buttons,
			[]InlineButton{{Text: "Profil akun", URL: AppURL(dashboardLink, "/pengaturan/profil")}, {Text: "Keamanan", URL: AppURL(dashboardLink, "/pengaturan/keamanan")}},
			[]InlineButton{{Text: "Koneksi Telegram", URL: AppURL(dashboardLink, "/pengaturan")}},
			[]InlineButton{{Text: "Akun broker", URL: AppURL(dashboardLink, "/akun")}, {Text: "Strategi & eksekusi", URL: AppURL(dashboardLink, "/strategi")}},
		)
	}
	buttons = append(buttons, []InlineButton{{Text: LabelMenu, CallbackData: CallbackMenu}})
	return Message{Text: "⚙️ Pengaturan\n\nKelola profil, keamanan, koneksi, dan preferensi trading melalui dashboard. Pilih bagian yang ingin dibuka.", Buttons: buttons}
}

// FormatMenuMessage keeps the home independent of market or broker availability.
func FormatMenuMessage(nama, paket, dashboardLink string) Message {
	sapaan := "Selamat datang."
	if nama != "" {
		sapaan = "Halo, " + escapeHTML(nama) + "."
	}
	return Message{Text: "<b>BOT Trading ILA</b>\n\n" + sapaan + "\n\nPantau perkembangan setup, cek posisi, dan lihat hasil trading dari sini.\n\nPilih menu untuk melanjutkan.", Buttons: tombolMenu(dashboardLink)}
}

func tombolMenu(dashboardLink string) [][]InlineButton {
	rows := [][]InlineButton{
		{{Text: "📡 Pantau saat ini", CallbackData: CallbackPantau, Style: "primary"}},
		{{Text: "📋 Riwayat", CallbackData: CallbackRiwayatMenu}, {Text: "📈 Ringkasan hari ini", CallbackData: CallbackHariIni}},
	}
	if dashboardLink != "" {
		rows = append(rows, []InlineButton{{Text: CabangDashboard, URL: dashboardLink, Style: "primary"}})
	}
	return append(rows, []InlineButton{{Text: LabelPengaturan, CallbackData: CallbackPengaturan}, {Text: "❔ Bantuan", CallbackData: CallbackBantuan}})
}

func FormatRiwayatMenu() Message {
	return Message{Text: "📋 Riwayat\n\nTransaksi akun mencatat order broker. Tidak tereksekusi menampilkan hasil simulasi sinyal yang tidak menjadi transaksi akun.", Buttons: [][]InlineButton{
		{{Text: "Transaksi akun", CallbackData: CallbackRiwayat + "0", Style: "primary"}},
		{{Text: "Tidak tereksekusi", CallbackData: CallbackSimulasi + "1"}},
		{{Text: LabelMenu, CallbackData: CallbackMenu}},
	}}
}

func FormatBantuanMessage() Message {
	return Message{Text: "❔ Bantuan\n\n<b>Pantau</b>\nLihat posisi aktif dan perkembangan setup.\n\n<b>Riwayat</b>\nTinjau transaksi akun serta hasil simulasi sinyal yang tidak dieksekusi.\n\n<b>Keputusan entry</b>\nKonfirmasi atau lewati sinyal melalui tombol pada notifikasi. Syarat trading tetap diperiksa sebelum order dikirim.\n\nSimulasi bukan transaksi broker dan tidak mengubah saldo akun.", Buttons: [][]InlineButton{
		{{Text: "Panduan tahap sinyal", CallbackData: CallbackPanduan}},
		{{Text: LabelPengaturan, CallbackData: CallbackPengaturan}, {Text: LabelMenu, CallbackData: CallbackMenu}},
	}}
}

func FormatPanduanMessage() Message {
	return Message{Text: "Panduan tahap sinyal\n\n1/6 · Membaca struktur timeframe besar\n2/6 · Menunggu konfirmasi candle\n3/6 · Mencari pemicu entry\n4/6 · Setup siap untuk entry\n5/6 · Posisi akun berjalan\n6/6 · Posisi selesai\n\nLabel ILA 2.0 atau ILA 3.0 menunjukkan strategi sumber. Detail kondisi mengikuti strategi tersebut.\n\nTahap ditampilkan sesuai pembaruan yang diterima. Beberapa tahap dapat berlangsung berdekatan atau tidak teramati; bot tidak membuat riwayat tahap yang tidak diterima.", Buttons: [][]InlineButton{{{Text: "Kembali ke Bantuan", CallbackData: CallbackBantuan}, {Text: LabelMenu, CallbackData: CallbackMenu}}}}
}

// itoaOpsional menampilkan angka atau tanda hubung bila nol.
func itoaOpsional(v int) string {
	if v <= 0 {
		return "—"
	}
	return fmt.Sprintf("%d", v)
}
