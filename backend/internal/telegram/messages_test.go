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
	if !strings.Contains(msg.Text, "2,30") {
		t.Fatalf("RR tidak diformat:\n%s", msg.Text)
	}
	if !strings.Contains(msg.Text, "https://app.test/approve/abc") {
		t.Fatalf("deep-link tidak ada:\n%s", msg.Text)
	}
}

func TestFormatSetupWithoutLink(t *testing.T) {
	msg := FormatSetupMessage("EURUSD", "SELL", "ILA 2.0", 1.08500, 1.09000, 1.07500, 2.0, 0.5, 25, "")

	if strings.Contains(msg.Text, "](http") {
		t.Fatal("tautan kosong tidak boleh membentuk markdown link")
	}
	if !strings.Contains(msg.Text, "dashboard") {
		t.Fatalf("harus mengarahkan ke dashboard:\n%s", msg.Text)
	}
}

func TestFormatHeldMessageAlwaysExplainsReason(t *testing.T) {
	msg := FormatHeldMessage("USTEC", "SELL", "ILA 2.0", 2.1, "masih ada posisi terbuka")

	if !strings.Contains(msg.Text, "masih ada posisi terbuka") {
		t.Fatalf("alasan penolakan harus tampil:\n%s", msg.Text)
	}
	if ContainsForbiddenTerms(msg.Text) {
		t.Fatal("pesan memuat istilah internal")
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
	for _, want := range []string{"RADAR ILA 2.0 — XAUUSD", "BUY", "H4 (HTF) ➔ M15 (LTF)", "1/4", "2.765,50", "2.738,20", "https://rekavia.com/app/sinyal?pair=XAUUSD"} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("pesan HTF tidak memuat %q:\n%s", want, msg.Text)
		}
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
	for _, want := range []string{"RADAR ILA 2.0 — XAUUSD", "2/4", "H4 ➔ M15", "2.747,50", "2.737,80", "2.765,50", "2,85", "https://rekavia.com/app/sinyal?pair=XAUUSD"} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("pesan LTF tidak memuat %q:\n%s", want, msg.Text)
		}
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
	for _, want := range []string{"POSISI TERPASANG — XAUUSD", "BUY #8492015", "cTrader Live", "4/4", "0,10 Lot", "2.748,15", "2.737,80", "2.765,50", "+2,72R", "https://rekavia.com/app/posisi"} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("pesan posisi terpasang tidak memuat %q:\n%s", want, msg.Text)
		}
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

	for _, ingin := range []string{"PELUANG ENTRY", "XAUUSD", "W1-H4", "Eksekusi Manual"} {
		if !strings.Contains(msg.Text, ingin) {
			t.Errorf("pesan tidak memuat %q:\n%s", ingin, msg.Text)
		}
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
	if !strings.Contains(msg.Text, "Buka dashboard") {
		t.Errorf("harus ada arahan tanpa link:\n%s", msg.Text)
	}
}
