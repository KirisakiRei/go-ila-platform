package telegram

import (
	"fmt"
	"ila/backend/internal/timezone"
	"strings"
	"time"
)

// BarisPantau adalah satu posisi yang sedang berjalan.
type BarisPantau struct {
	DetailURL  string
	StopLoss   float64
	TakeProfit float64
	OpenedAt   time.Time
	Ticket     string
	Symbol     string
	Direction  string
	Lots       float64
	EntryPrice float64
	NowPrice   float64 // nol berarti harga live belum tersedia
	OutcomeR   float64 // terisi hanya bila NowPrice tersedia
	ProfitLoss float64 // terisi hanya bila NowPrice tersedia
	HasPnL     bool
}

// BarisMenunggu adalah setup yang masih mencari entry.
type BarisMenunggu struct {
	DetailURL       string
	StrategyVersion string
	Symbol          string
	Htf             string
	Ltf             string
	Direction       string
	// Tahap adalah penanda siklus seperti "3/6".
	Tahap string
	// TahapTeks menjelaskan apa yang sedang ditunggu pada tahap itu.
	TahapTeks    string
	Target       float64
	StopLoss     float64
	TriggerHarga float64
	// Sejak adalah waktu tahap ini dimulai dan Lama sudah berapa lama berjalan.
	Sejak time.Time
	Lama  string
}

// BarisSelesai adalah posisi yang sudah tutup hari ini.
type BarisSelesai struct {
	Symbol    string
	Direction string
	OutcomeR  float64
	ClosedAt  time.Time
}

// PantauParams memuat isi halaman pantauan.
type PantauParams struct {
	Diperbarui    time.Time
	Posisi        []BarisPantau
	Menunggu      []BarisMenunggu
	Selesai       []BarisSelesai
	HargaTersedia bool
	DashboardLink string
	// Halaman dan TotalHalaman mengatur navigasi daftar setup menunggu.
	Halaman      int
	TotalHalaman int
}

// PantauPerHalaman adalah jumlah setup menunggu pada satu halaman pantauan.
const PantauPerHalaman = 5

// FormatPantauMessage menyusun halaman "Pantau Saat Ini".
//
// Urutannya mengikuti yang paling dicari trader: posisi yang sedang berjalan
// lebih dulu, lalu setup yang menunggu, terakhir yang sudah selesai hari ini.
func FormatPantauMessage(p PantauParams) Message {
	var b strings.Builder
	waktu := ""
	if !p.Diperbarui.IsZero() {
		waktu = " · " + p.Diperbarui.In(time.FixedZone("WIB", 7*3600)).Format("02 Jan 2006, 15:04") + " WIB"
	}
	b.WriteString("📊 Pantauan" + waktu + "\n")
	if len(p.Posisi) == 0 && len(p.Menunggu) == 0 && len(p.Selesai) == 0 {
		b.WriteString("\nBelum ada posisi berjalan dan belum ada setup yang menunggu.\n")
		b.WriteString("Bot akan mengabari begitu ada peluang masuk.\n")
		return Message{Text: b.String(), Buttons: menuButtons(p.DashboardLink)}
	}
	if len(p.Posisi) > 0 {
		fmt.Fprintf(&b, "\nPosisi berjalan (%d)\n", len(p.Posisi))
		for _, pos := range p.Posisi {
			b.WriteString(kotakPosisi(pos))
		}
	}
	if len(p.Menunggu) > 0 {
		halaman := p.Halaman
		if halaman < 1 {
			halaman = 1
		}
		total := p.TotalHalaman
		if total < 1 {
			total = 1
		}
		if total > 1 {
			fmt.Fprintf(&b, "\nSetup yang dipantau · halaman %d/%d\n", halaman, total)
		} else {
			fmt.Fprintf(&b, "\nSetup yang dipantau (%d)\n", len(p.Menunggu))
		}
		for _, m := range p.Menunggu {
			b.WriteString(kotakMenunggu(m))
		}
	}
	if len(p.Selesai) > 0 {
		fmt.Fprintf(&b, "\nSelesai hari ini (%d)\n", len(p.Selesai))
		for _, s := range p.Selesai {
			b.WriteString(kotakSelesai(s))
		}
	}
	if !p.HargaTersedia && len(p.Posisi) > 0 {
		b.WriteString("\nHarga live dan PnL berjalan tersedia di akun broker; halaman ini tidak mengambil kutipan harga live.\n")
	}
	buttons := tombolPantau(p.DashboardLink, p.Halaman, p.TotalHalaman)
	for i, pos := range p.Posisi {
		if pos.DetailURL != "" {
			buttons = append(buttons, []InlineButton{{Text: fmt.Sprintf("Posisi %d · %s", i+1, pos.Symbol), CallbackData: pos.DetailURL}})
		}
	}
	for i, row := range p.Menunggu {
		if row.DetailURL != "" {
			buttons = append(buttons, []InlineButton{{Text: fmt.Sprintf("Setup %d · %s %s/%s", i+1, row.Symbol, row.Htf, row.Ltf), CallbackData: row.DetailURL}})
		}
	}
	return Message{Text: b.String(), Buttons: buttons}
}

// kotakPosisi merangkum satu posisi berjalan.
//
// Harga terkini hanya ditampilkan bila benar-benar tersedia. Tanpa itu kolom
// "Kini" diisi tanda hubung, bukan angka lama yang bisa menyesatkan.
func kotakPosisi(pos BarisPantau) string {
	kini := "—"
	if pos.NowPrice != 0 {
		kini = formatPrice(pos.NowPrice)
	}
	baris := [][2]string{
		{"Arah", arahTampil(pos.Direction)},
		{"Order", nomorOrder(pos.Ticket)},
		{"Lot", lotTampil(pos.Lots)},
		{"Entry", formatPrice(pos.EntryPrice)},
		{"Kini", kini},
	}
	if pos.HasPnL {
		baris = append(baris,
			[2]string{"Hasil", hasilR(pos.OutcomeR)},
			[2]string{"Untung/Rugi", formatMoney(pos.ProfitLoss)},
		)
	}
	return "◆ " + pos.Symbol + "\n" + daftar(baris)
}

// kotakMenunggu merangkum satu setup yang menunggu entry.
//
// Baris harga yang belum tersedia dilewati, bukan diisi tanda hubung: pada
// tahap awal level target, batas, dan trigger memang belum ditentukan.
func kotakMenunggu(m BarisMenunggu) string {
	judul := m.Symbol
	if m.Htf != "" && m.Ltf != "" {
		judul += " · " + m.Htf + " ➔ " + m.Ltf
	}
	baris := [][2]string{{"Strategi", metodeTampil(m.StrategyVersion)}, {"Arah", arahTampil(m.Direction)}}
	if m.Tahap != "" {
		tahap := m.Tahap
		if m.TahapTeks != "" {
			tahap += " · " + m.TahapTeks
		}
		baris = append(baris, [2]string{"Tahap", tahap})
	}
	if m.Target != 0 {
		baris = append(baris, [2]string{"Target", formatPrice(m.Target)})
	}
	if m.StopLoss != 0 {
		baris = append(baris, [2]string{"Batas", formatPrice(m.StopLoss)})
	}
	if m.TriggerHarga != 0 {
		baris = append(baris, [2]string{"Trigger", formatPrice(m.TriggerHarga)})
	}
	if !m.Sejak.IsZero() {
		sejak := formatWIBdanNY(m.Sejak)
		if m.Lama != "" {
			sejak += " · " + m.Lama
		}
		baris = append(baris, [2]string{"Sejak", sejak})
	}
	return "◇ " + judul + "\n" + daftar(baris)
}

// kotakSelesai merangkum satu posisi yang sudah tutup.
func kotakSelesai(s BarisSelesai) string {
	jam := ""
	if !s.ClosedAt.IsZero() {
		jam = s.ClosedAt.In(time.FixedZone("WIB", 7*3600)).Format("15:04")
	}
	return fmt.Sprintf("%s %s · %s · %s\n", simbolHasil(s.OutcomeR), s.Symbol, hasilR(s.OutcomeR), jam)
}

// simbolHasil memilih penanda hasil: menang, kalah, atau seri.
func simbolHasil(outcomeR float64) string {
	switch {
	case outcomeR > 0:
		return "✓"
	case outcomeR < 0:
		return "✕"
	default:
		return "—"
	}
}

// formatWIBdanNY menampilkan satu waktu dalam WIB dan NY.
//
// Dua zona sengaja ditampilkan berdampingan: sesi pasar dibaca dalam NY,
// sedangkan hari akuntansi dan kebiasaan trader memakai WIB. Tanpa keduanya,
// trader harus menghitung sendiri selisihnya setiap kali melihat setup.
func formatWIBdanNY(t time.Time) string {
	wib := t.In(time.FixedZone("WIB", 7*3600)).Format("02 Jan 2006, 15:04")
	ny := t.In(timezone.NY).Format("15:04")
	return wib + " WIB · " + ny + " NY"
}

// tombolPantau menyusun tombol bawah halaman pantauan.
//
// Navigation preserves three columns with valid no-op callbacks at page boundaries.
func tombolPantau(dashboardLink string, halaman, total int) [][]InlineButton {
	var baris [][]InlineButton
	if total > 1 {
		if halaman < 1 {
			halaman = 1
		}
		sebelum := InlineButton{Text: "‹ Sebelumnya", CallbackData: CallbackNoop}
		if halaman > 1 {
			sebelum.CallbackData = fmt.Sprintf("%s%d", CallbackPantauHalaman, halaman-1)
		}
		berikut := InlineButton{Text: "Berikutnya ›", CallbackData: CallbackNoop}
		if halaman < total {
			berikut.CallbackData = fmt.Sprintf("%s%d", CallbackPantauHalaman, halaman+1)
		}
		baris = append(baris, []InlineButton{
			sebelum,
			{Text: "Kembali", CallbackData: CallbackMenu},
			berikut,
		})
	}
	baris = append(baris, []InlineButton{
		{Text: "Segarkan", CallbackData: fmt.Sprintf("%s%d", CallbackPantauHalaman, max(1, halaman)), Style: "primary"},
		{Text: LabelMenu, CallbackData: CallbackMenu},
	})
	if dashboardLink != "" {
		baris = append(baris, []InlineButton{
			{Text: CabangDashboard, URL: dashboardLink},
		})
	}
	return baris
}

// menuButtons menyusun tombol sederhana menuju dashboard.
func menuButtons(dashboardLink string) [][]InlineButton {
	baris := [][]InlineButton{{{Text: LabelSegarkan, CallbackData: CallbackSegarkan}, {Text: LabelMenu, CallbackData: CallbackMenu}}}
	if dashboardLink != "" {
		baris = append(baris, []InlineButton{{Text: CabangDashboard, URL: dashboardLink}})
	}
	return baris
}
