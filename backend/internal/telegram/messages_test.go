package telegram

import (
	"strings"
	"testing"
	"time"
)

func TestContainsForbiddenTerms(t *testing.T) {
	forbidden := []string{
		"FVG Bullish aktif",
		"sweep ekor valid",
		"CISD terkonfirmasi",
		"body break body",
		"Draw on Liquidity tercapai",
	}
	for _, text := range forbidden {
		if !ContainsForbiddenTerms(text) {
			t.Fatalf("%q seharusnya terdeteksi", text)
		}
	}

	aman := "Setup sesuai rencana. Entry 4.120,50 dengan batas risiko 0,5%."
	if ContainsForbiddenTerms(aman) {
		t.Fatal("teks aman tidak boleh ditandai")
	}
}

func TestSanitizeReplacesLeakingText(t *testing.T) {
	got := Sanitize("FVG Bullish aktif di M15")
	if ContainsForbiddenTerms(got) {
		t.Fatalf("hasil sanitasi masih memuat istilah terlarang: %q", got)
	}
	if !strings.Contains(got, "dashboard") {
		t.Fatalf("hasil sanitasi harus mengarahkan ke dashboard: %q", got)
	}

	bersih := "Setup sesuai rencana."
	if Sanitize(bersih) != bersih {
		t.Fatal("teks bersih tidak boleh diubah")
	}
}

func TestFormatSetupMessageNumbers(t *testing.T) {
	msg := FormatSetupMessage("XAUUSD", "BUY", "ILA 2.0", 4120.50, 4112.30, 4149.20, 2.3, 0.5, 51.25, "https://app.test/approve/abc")

	if ContainsForbiddenTerms(msg.Text) {
		t.Fatalf("pesan memuat istilah internal: %s", msg.Text)
	}

	// Gaya Indonesia: titik ribuan, koma desimal.
	if !strings.Contains(msg.Text, "4.120,50") {
		t.Fatalf("harga tidak diformat gaya Indonesia:\n%s", msg.Text)
	}
	if !strings.Contains(msg.Text, "$51,25") {
		t.Fatalf("nominal risiko tidak diformat:\n%s", msg.Text)
	}
	if !strings.Contains(msg.Text, "1:2,30") {
		t.Fatalf("RR tidak diformat:\n%s", msg.Text)
	}
	if len(msg.Buttons) == 0 || len(msg.Buttons[0]) == 0 || msg.Buttons[0][0].URL != "https://app.test/approve/abc" {
		t.Fatalf("tombol deep-link tidak ada atau salah: %v", msg.Buttons)
	}
}

func TestFormatSetupWithoutLink(t *testing.T) {
	msg := FormatSetupMessage("EURUSD", "SELL", "ILA 2.0", 1.08500, 1.09000, 1.07500, 2.0, 0.5, 25, "")

	if strings.Contains(msg.Text, "](http") {
		t.Fatal("tautan kosong tidak boleh membentuk markdown link")
	}
	if len(msg.Buttons) != 0 {
		t.Fatalf("tanpa link tidak boleh ada tombol, dapat: %v", msg.Buttons)
	}
}

func TestFormatHeldMessageAlwaysExplainsReason(t *testing.T) {
	msg := FormatHeldMessage(HeldParams{
		Symbol: "USTEC", Direction: "SELL", StrategyVersion: "ila-2.0",
		Htf: "H4", Ltf: "M15", RiskReward: 2.1,
		Alasan:     PenahananPosisi,
		Keterangan: "Masih ada posisi terbuka. Tunggu sampai ditutup.",
	})

	if !strings.Contains(msg.Text, "Masih ada posisi terbuka") {
		t.Fatalf("alasan penolakan harus tampil:\n%s", msg.Text)
	}
	if ContainsForbiddenTerms(msg.Text) {
		t.Fatal("pesan memuat istilah internal")
	}
	if !strings.Contains(msg.Text, "Tahap 4/6") {
		t.Errorf("tahap siklus harus tampil:\n%s", msg.Text)
	}
}

// TestHeldKuotaMenampilkanAngka memastikan penahanan kuota menyebut besar
// kuotanya, bukan hanya "sudah habis".
func TestHeldKuotaMenampilkanAngka(t *testing.T) {
	msg := FormatHeldMessage(HeldParams{
		Symbol: "GBPUSD", Direction: "BUY", Htf: "H4", Ltf: "M15",
		EntryPrice: 1.2845, StopLoss: 1.2810, TakeProfit: 1.2915, RiskReward: 2,
		Alasan: PenahananKuota, KuotaTerpakai: 2, KuotaMaks: 2,
	})

	if ContainsForbiddenTerms(msg.Text) {
		t.Fatalf("pesan memuat istilah internal:\n%s", msg.Text)
	}
	if !strings.Contains(msg.Text, "2 dari 2 trade terpakai") {
		t.Errorf("angka kuota tidak tampil:\n%s", msg.Text)
	}
	if strings.Contains(msg.Text, "Tidak dieksekusi:") {
		t.Errorf("kalimat penolakan lama tidak boleh dipakai:\n%s", msg.Text)
	}
}

// TestCancelledTidakMenjanjikanPemantauan memastikan setup yang sudah mati
// tidak diberi kalimat "tetap dipantau".
func TestCancelledTidakMenjanjikanPemantauan(t *testing.T) {
	msg := FormatCancelledMessage(HeldParams{
		Symbol: "EURUSD", Direction: "SELL", Htf: "H4", Ltf: "M15",
		Alasan:     PenahananBatal,
		Keterangan: "Struktur ditembus.",
	})

	if strings.Contains(msg.Text, "tetap dipantau") {
		t.Errorf("setup batal tidak boleh dijanjikan dipantau:\n%s", msg.Text)
	}
	if !strings.Contains(msg.Text, "sudah gak berlaku") {
		t.Errorf("pesan batal harus tegas:\n%s", msg.Text)
	}
}

// TestHeldHargaMenungguKembali memastikan penahanan karena harga lewat memakai
// bahasa menunggu, bukan bahasa batal.
func TestHeldHargaMenungguKembali(t *testing.T) {
	msg := FormatHeldMessage(HeldParams{
		Symbol: "USTEC", Direction: "SELL", Htf: "H4", Ltf: "M15",
		EntryPrice: 24812.5, StopLoss: 24905, TakeProfit: 24627.5, RiskReward: 2,
		Alasan:      PenahananHarga,
		BatasTunggu: "11 Nov 15:40 WIB",
	})

	if !strings.Contains(msg.Text, "Menunggu harga kembali") {
		t.Errorf("judul harus menyebut menunggu harga:\n%s", msg.Text)
	}
	if !strings.Contains(msg.Text, "11 Nov 15:40 WIB") {
		t.Errorf("batas tunggu harus tampil:\n%s", msg.Text)
	}
	if ContainsForbiddenTerms(msg.Text) {
		t.Fatalf("pesan memuat istilah internal:\n%s", msg.Text)
	}
}

func TestDeepLink(t *testing.T) {
	if got := DeepLink("https://app.test/", "tok"); got != "https://app.test/app/approve/tok" {
		t.Fatalf("DeepLink = %q", got)
	}
	if got := DeepLink("https://app.test/app", "tok"); got != "https://app.test/app/approve/tok" {
		t.Fatalf("DeepLink with existing /app = %q", got)
	}
	if DeepLink("", "tok") != "" {
		t.Fatal("tanpa base URL harus kosong")
	}
	if DeepLink("https://app.test", "") != "" {
		t.Fatal("tanpa token harus kosong")
	}
}

func TestAppURL(t *testing.T) {
	if got := AppURL("https://rekavia.com", "/sinyal?pair=XAUUSD"); got != "https://rekavia.com/app/sinyal?pair=XAUUSD" {
		t.Fatalf("AppURL = %q", got)
	}
	if got := AppURL("https://rekavia.com/app", "/posisi"); got != "https://rekavia.com/app/posisi" {
		t.Fatalf("AppURL with existing /app = %q", got)
	}
	if AppURL("", "/posisi") != "" {
		t.Fatal("tanpa base URL harus kosong")
	}
}

func TestFormatRadarHTFMessage(t *testing.T) {
	msg := FormatRadarHTFMessage(RadarHTFParams{
		Symbol:          "XAUUSD",
		Direction:       "BUY",
		StrategyVersion: "ILA 2.0",
		Htf:             "H4",
		Ltf:             "M15",
		TargetDOL:       2765.50,
		StopLoss:        2738.20,
		Details:         "Area likuiditas telah disentuh.",
		DashboardLink:   "https://rekavia.com/app/sinyal?pair=XAUUSD",
	})

	if ContainsForbiddenTerms(msg.Text) {
		t.Fatalf("pesan memuat istilah internal: %s", msg.Text)
	}
	for _, want := range []string{"XAUUSD · H4 ➔ M15", "Tahap 1/6", "BUY", "2.765,50", "2.738,20"} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("pesan HTF tidak memuat %q:\n%s", want, msg.Text)
		}
	}
	if len(msg.Buttons) == 0 || msg.Buttons[0][0].URL != "https://rekavia.com/app/sinyal?pair=XAUUSD" {
		t.Fatalf("tombol dashboard HTF tidak ada: %v", msg.Buttons)
	}
}

func TestFormatRadarLTFMessage(t *testing.T) {
	msg := FormatRadarLTFMessage(RadarLTFParams{
		Symbol:          "XAUUSD",
		Direction:       "BUY",
		StrategyVersion: "ILA 2.0",
		Htf:             "H4",
		Ltf:             "M15",
		EntryPrice:      2747.50,
		StopLoss:        2737.80,
		TakeProfit:      2765.50,
		RiskReward:      2.85,
		CISDLevel:       2746.80,
		Details:         "",
		DashboardLink:   "https://rekavia.com/app/sinyal?pair=XAUUSD",
	})

	if ContainsForbiddenTerms(msg.Text) {
		t.Fatalf("pesan memuat istilah internal: %s", msg.Text)
	}
	for _, want := range []string{"XAUUSD · H4 ➔ M15", "Tahap 3/6", "2.737,80", "2.765,50", "2.746,80"} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("pesan LTF tidak memuat %q:\n%s", want, msg.Text)
		}
	}
	if len(msg.Buttons) == 0 || msg.Buttons[0][0].URL != "https://rekavia.com/app/sinyal?pair=XAUUSD" {
		t.Fatalf("tombol dashboard LTF tidak ada: %v", msg.Buttons)
	}
}

func TestFormatPositionOpenedMessage(t *testing.T) {
	openedAt := time.Date(2026, 11, 11, 7, 15, 2, 0, time.UTC)
	msg := FormatPositionOpenedMessage(PositionOpenedParams{
		Symbol:       "XAUUSD",
		Direction:    "BUY",
		TicketNumber: "8492015",
		BrokerType:   "cTrader Live",
		LotSize:      0.10,
		OpenPrice:    2748.15,
		StopLoss:     2737.80,
		TakeProfit:   2765.50,
		OutcomeR:     2.72,
		OpenedAt:     openedAt,
		PosisiLink:   "https://rekavia.com/app/posisi",
	})

	if ContainsForbiddenTerms(msg.Text) {
		t.Fatalf("pesan memuat istilah internal: %s", msg.Text)
	}
	for _, want := range []string{"XAUUSD", "Tahap 5/6", "BUY #8492015", "cTrader Live", "0,10 lot", "2.748,15", "2.737,80", "2.765,50", "1:2,72"} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("pesan posisi terpasang tidak memuat %q:\n%s", want, msg.Text)
		}
	}
	if len(msg.Buttons) == 0 || msg.Buttons[0][0].URL != "https://rekavia.com/app/posisi" {
		t.Fatalf("tombol posisi tidak ada: %v", msg.Buttons)
	}
}

func TestThrottleSpacing(t *testing.T) {
	c := NewClient(Config{MinInterval: 10 * time.Millisecond}, nil)

	start := time.Now()
	c.throttle(nil, "chat-1")
	c.throttle(nil, "chat-1")

	if elapsed := time.Since(start); elapsed < 10*time.Millisecond {
		t.Fatalf("jeda antar pesan = %v, ingin minimal 10ms", elapsed)
	}
}

func TestFormatThousands(t *testing.T) {
	cases := map[float64]string{
		4120.5:     "4.120,50",
		21450:      "21.450,00",
		0.5:        "0,50",
		1234567.89: "1.234.567,89",
	}
	for input, want := range cases {
		if got := formatThousands(input, 2); got != want {
			t.Fatalf("formatThousands(%v) = %q, ingin %q", input, got, want)
		}
	}
}

// TestFormatOpportunityMessageMenyebutTimeframe memastikan pesan peluang
// berbeda dari pesan tertahan: ia menyebut pasangan timeframe dan menawarkan
// langkah berikutnya, bukan sekadar memberi tahu setup tidak dijalankan.
func TestFormatOpportunityMessageMenyebutTimeframe(t *testing.T) {
	msg := FormatOpportunityMessage("XAUUSD", "BUY", "ila-2.0", "W1", "H4",
		4120.5, 4112.3, 4149.2, 2.3, "https://rekavia.com/approve/abc")

	for _, ingin := range []string{"XAUUSD · W1 ➔ H4", "Tahap 4/6", "4.120,50"} {
		if !strings.Contains(msg.Text, ingin) {
			t.Errorf("pesan tidak memuat %q:\n%s", ingin, msg.Text)
		}
	}
	if len(msg.Buttons) == 0 || msg.Buttons[0][0].URL != "https://rekavia.com/approve/abc" {
		t.Fatalf("tombol peluang tidak ada: %v", msg.Buttons)
	}
	if strings.Contains(msg.Text, "Tidak dieksekusi:") {
		t.Error("pesan peluang tidak boleh memakai kalimat penolakan")
	}
}

// TestFormatOpportunityTanpaTimeframe memastikan pesan tetap rapi bila
// timeframe kosong (engine belum mengirim), tanpa baris kosong menggantung.
func TestFormatOpportunityTanpaTimeframe(t *testing.T) {
	msg := FormatOpportunityMessage("XAUUSD", "BUY", "ila-2.0", "", "",
		4120.5, 4112.3, 4149.2, 2.3, "")

	if strings.Contains(msg.Text, "Timeframe:") {
		t.Errorf("baris timeframe tidak boleh muncul saat kosong:\n%s", msg.Text)
	}
	if len(msg.Buttons) != 0 {
		t.Errorf("tanpa link tidak boleh ada tombol, dapat: %v", msg.Buttons)
	}
}

// TestBoxMemakaiPreBukanBacktick memastikan kotak memakai <pre>, bukan
// backtick. Client mengirim dengan parse_mode HTML; backtick akan tampil
// mentah sebagai tiga karakter di Telegram.
func TestBoxMemakaiPreBukanBacktick(t *testing.T) {
	msg := FormatRadarHTFMessage(RadarHTFParams{
		Symbol: "EURUSD", Htf: "H4", Ltf: "M15", Siklus: "Candle H4 · 08:00 NY",
	})

	if strings.Contains(msg.Text, "```") {
		t.Errorf("pesan tidak boleh memuat backtick:\n%s", msg.Text)
	}
	if !strings.Contains(msg.Text, "<pre>") || !strings.Contains(msg.Text, "</pre>") {
		t.Errorf("kotak harus memakai <pre>:\n%s", msg.Text)
	}
}

// TestBoxMengamankanKarakterHTML memastikan teks dari engine tidak merusak
// pesan. Keterangan alasan berasal dari luar dan bisa memuat < atau &.
func TestBoxMengamankanKarakterHTML(t *testing.T) {
	msg := FormatHeldMessage(HeldParams{
		Symbol: "EURUSD", Direction: "SELL", Htf: "H4", Ltf: "M15",
		Alasan:     PenahananAkun,
		Keterangan: "Saldo < minimum & akun ditahan",
	})

	if strings.Contains(msg.Text, "< minimum") {
		t.Errorf("karakter khusus tidak di-escape:\n%s", msg.Text)
	}
	if !strings.Contains(msg.Text, "&lt; minimum &amp; akun") {
		t.Errorf("escape HTML tidak sesuai:\n%s", msg.Text)
	}
}
