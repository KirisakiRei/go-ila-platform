package telegram

import (
	"fmt"
	"strings"
	"time"
)

// BarisPantau adalah satu posisi yang sedang berjalan.
type BarisPantau struct {
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
	Symbol       string
	Direction    string
	Tahap        string
	Target       float64
	StopLoss     float64
	TriggerHarga float64
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
}

// FormatPantauMessage menyusun halaman "Pantau Saat Ini".
//
// Urutannya mengikuti yang paling dicari trader: posisi yang sedang berjalan
// lebih dulu, lalu setup yang menunggu, terakhir yang sudah selesai hari ini.
func FormatPantauMessage(p PantauParams) Message {
	var b strings.Builder

	waktu := ""
	if !p.Diperbarui.IsZero() {
		waktu = " · " + p.Diperbarui.In(time.FixedZone("WIB", 7*3600)).Format("15:04") + " WIB"
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
		fmt.Fprintf(&b, "\nMenunggu entry (%d)\n", len(p.Menunggu))
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
		b.WriteString("\nHarga terkini belum bisa diambil dari broker. Coba segarkan lagi sebentar.\n")
	}

	return Message{Text: b.String(), Buttons: tombolPantau(p.DashboardLink)}
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
func kotakMenunggu(m BarisMenunggu) string {
	baris := [][2]string{
		{"Arah", arahTampil(m.Direction)},
		{"Tahap", m.Tahap},
		{"Target", formatHargaOpsional(m.Target)},
		{"Batas", formatHargaOpsional(m.StopLoss)},
		{"Trigger", formatHargaOpsional(m.TriggerHarga)},
	}
	return m.Symbol + "\n" + daftar(baris)
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

// tombolPantau menyusun tombol bawah halaman pantauan.
func tombolPantau(dashboardLink string) [][]InlineButton {
	var baris [][]InlineButton
	baris = append(baris, []InlineButton{
		{Text: "Segarkan", CallbackData: CallbackSegarkan},
	})
	if dashboardLink != "" {
		baris = append(baris, []InlineButton{
			{Text: "Buka di dashboard", URL: dashboardLink},
		})
	}
	return baris
}

// menuButtons menyusun tombol sederhana menuju dashboard.
func menuButtons(dashboardLink string) [][]InlineButton {
	baris := [][]InlineButton{{{Text: LabelSegarkan, CallbackData: CallbackSegarkan}, {Text: LabelMenu, CallbackData: CallbackMenu}}}
	if dashboardLink != "" {
		baris = append(baris, []InlineButton{{Text: "Buka di dashboard", URL: dashboardLink}})
	}
	return baris
}
