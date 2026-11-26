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
	// Kalimat dipertahankan; hanya istilahnya yang diterjemahkan.
	if !strings.Contains(got, "M15") || !strings.Contains(got, "Bullish") {
		t.Fatalf("isi kalimat tidak boleh hilang: %q", got)
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
		Link:       "https://rekavia.com/app/approve?token=abc",
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
	if len(msg.Buttons) != 1 || len(msg.Buttons[0]) != 1 || !strings.Contains(msg.Buttons[0][0].Text, "Eksekusi Manual") {
		t.Errorf("penahanan posisi terbuka harus menampilkan tombol Eksekusi Manual, dapat: %+v", msg.Buttons)
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
	if !strings.Contains(msg.Text, "2 dari 2 trade otomatis") {
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

// TestSanitizeMempertahankanAngka memastikan istilah internal diganti per kata,
// bukan seluruh kalimat. Sebelumnya satu istilah terlarang membuang seluruh
// isi pesan sehingga user kehilangan level dan arahnya.
func TestSanitizeMempertahankanAngka(t *testing.T) {
	masukan := "HTF H4 sweep valid. Menunggu CISD M15 di level 3880.24 (SL: 3895.33)."
	hasil := Sanitize(masukan)

	if ContainsForbiddenTerms(hasil) {
		t.Fatalf("hasil masih memuat istilah terlarang: %q", hasil)
	}
	for _, ingin := range []string{"3880.24", "3895.33", "H4", "M15"} {
		if !strings.Contains(hasil, ingin) {
			t.Errorf("hasil kehilangan %q: %q", ingin, hasil)
		}
	}
	if strings.Contains(hasil, "Detail level tersedia di dashboard") {
		t.Errorf("kalimat tidak boleh diganti utuh: %q", hasil)
	}
}

// TestRadarLTFTanpaLevelTetapInformatif memastikan tahap 3 tidak menjadi
// pesan kosong saat tingkat acuan belum terbentuk.
func TestCycleLabels(t *testing.T) {
	for _, tc := range []struct{ htf, ltf, cycle string }{
		{"Q", "W", "YEARLY"}, {"MN1", "D1", "QUARTERLY"},
		{"W1", "H4", "MONTHLY"}, {"D1", "H1", "WEEKLY"},
		{"4h", "M15", "DAILY"}, {"60", "M5", "90M"}, {"15m", "M1", "MICRO"},
		{"H4", "M1", "DAILY"}, // ILA3: LTF tidak menentukan cycle.
		{"", "M15", "Tidak diketahui"}, {"H2", "M15", "Tidak diketahui"},
		{"1M", "M1", "Tidak diketahui"}, {"M5", "M1", "Tidak diketahui"},
	} {
		t.Run(tc.htf+"-"+tc.ltf, func(t *testing.T) {
			msg := FormatSetupDetailedMessage(SetupMessageParams{Symbol: "EURUSD", Htf: tc.htf, Ltf: tc.ltf})
			if !strings.Contains(msg.Text, "Cycle: "+tc.cycle+"\n") {
				t.Fatalf("cycle salah: %s", msg.Text)
			}
		})
	}
}

func TestAllTradingAlertsRetainSetupCycle(t *testing.T) {
	messages := []Message{
		FormatRadarHTFMessage(RadarHTFParams{Htf: "H4", Ltf: "M15"}),
		FormatRadarSweepMessage(RadarHTFParams{Htf: "H4", Ltf: "M15"}),
		FormatRadarLTFMessage(RadarLTFParams{Htf: "H4", Ltf: "M15"}),
		FormatSetupDetailedMessage(SetupMessageParams{Htf: "H4", Ltf: "M15"}),
		FormatHeldMessage(HeldParams{Htf: "H4", Ltf: "M15", Alasan: PenahananKuota}),
		FormatCancelledMessage(HeldParams{Htf: "H4", Ltf: "M15"}),
		FormatCancelledMessage(HeldParams{Htf: "H4", Ltf: "M15", Alasan: PenahananHarga}),
		FormatPositionOpenedMessage(PositionOpenedParams{Htf: "H4", Ltf: "M15"}),
		FormatClosedMessage(ClosedParams{Htf: "H4", Ltf: "M15"}),
		FormatOpportunityMessage("EURUSD", "BUY", "ila-3.0", "H4", "M15", 1, 1, 1, 1, ""),
		FormatRadarStepMessage("EURUSD", "ila-2.0", "H4", "M15", "Radar", ""),
		FormatEksekusiBerhasil(EksekusiParams{Htf: "H4", Ltf: "M15"}),
		FormatEksekusiGagal("EURUSD", "H4", "M15", "Ditolak"),
		FormatDilewatiMessage("EURUSD", "H4", "M15"),
		FormatSudahTidakBerlaku("EURUSD", "H4", "M15", "Berakhir"),
	}
	for _, msg := range messages {
		if !strings.Contains(msg.Text, "Setup: H4 ➔ M15\nCycle: DAILY\n") {
			t.Errorf("konteks hilang: %s", msg.Text)
		}
		if ContainsForbiddenTerms(msg.Text) {
			t.Errorf("istilah internal bocor: %s", msg.Text)
		}
	}
	for _, msg := range []Message{FormatFinishedMessage("EURUSD", "Berakhir"), FormatKedaluwarsaTautan("EURUSD"), FormatSetupMessage("EURUSD", "BUY", "ila-2.0", 1, 1, 1, 1, 1, 1, "")} {
		if !strings.Contains(msg.Text, "Setup: Tidak diketahui\nCycle: Tidak diketahui\n") {
			t.Errorf("konteks kosong harus eksplisit: %s", msg.Text)
		}
	}
}

func TestRetainSetupContext(t *testing.T) {
	for _, original := range []string{
		"🟢 EURUSD · H4 ➔ M1\nTahap 4/6 · Siap masuk\nSetup: H4 ➔ M1\nCycle: DAILY\n",
		"🟢 EURUSD · H4 ➔ M1\nTahap 4/6 · Siap masuk\n", // Pesan sebelum label cycle ditambahkan.
	} {
		for _, msg := range []Message{FormatDilewatiMessage("", "", ""), FormatEksekusiGagal("", "", "", "Ditolak"), FormatKedaluwarsaTautan(""), {Text: "⚡ Eksekusi dikirim\n\nRincian menyusul.\n"}} {
			got := RetainSetupContext(msg, original)
			if !strings.Contains(got.Text, "EURUSD · H4 ➔ M1\n") || !strings.Contains(got.Text, "Setup: H4 ➔ M1\nCycle: DAILY\n") {
				t.Errorf("konteks edit hilang: %s", got.Text)
			}
		}
	}
	got := RetainSetupContext(FormatDilewatiMessage("", "", ""), "🟢 A<& · H4 ➔ M1\n")
	if strings.Contains(got.Text, "A<&") || !strings.Contains(got.Text, "A&lt;&amp;") {
		t.Fatalf("identitas harus di-escape: %s", got.Text)
	}
	for _, original := range []string{"", "pesan", "🟢 EURUSD · M15\n"} {
		got := RetainSetupContext(FormatKedaluwarsaTautan(""), original)
		if !strings.Contains(got.Text, "Cycle: Tidak diketahui\n") {
			t.Fatalf("jangan menebak HTF dari LTF: %s", got.Text)
		}
	}
}

func TestRadarLTFTanpaLevelTetapInformatif(t *testing.T) {
	msg := FormatRadarLTFMessage(RadarLTFParams{
		Symbol: "XAUUSD", Direction: "BUY", Htf: "H1", Ltf: "M5",
		StopLoss: 3875.10, TakeProfit: 3910.20, Siklus: "Candle H1 · 09:00 NY",
	})

	for _, ingin := range []string{"XAUUSD · H1 ➔ M5", "Tahap 3/6", "3.875,10", "3.910,20", "belum terbentuk"} {
		if !strings.Contains(msg.Text, ingin) {
			t.Errorf("pesan tidak memuat %q:\n%s", ingin, msg.Text)
		}
	}
}
