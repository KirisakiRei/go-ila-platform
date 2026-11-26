package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"ila/backend/internal/model"
	"ila/backend/internal/runner"
	"ila/backend/internal/runner/notify"
	"ila/backend/internal/settings"
	"ila/backend/internal/telegram"
)

// flagUji adalah sumbu kuota harian yang bisa diatur tes.
type flagUji map[string]int

func (f flagUji) Bool(_ context.Context, _ string, fallback bool) bool { return fallback }
func (f flagUji) Int(_ context.Context, key string, fallback int) int {
	if v, ada := f[key]; ada {
		return v
	}
	return fallback
}

func publisherUji(flags runner.Flags) *runnerPublisher {
	return &runnerPublisher{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		flags:  flags,
	}
}

// TestKuotaAngkaIkutPengaturan memastikan batas kuota di notifikasi memakai
// angka yang berlaku, bukan nilai bawaan. Notifikasi yang menyebut angka salah
// membuat user mengira masih ada sisa trade padahal tidak.
func TestKuotaAngkaIkutPengaturan(t *testing.T) {
	p := publisherUji(flagUji{settings.KeyMaxDailyTrades: 3})

	if got := p.kuotaAngka(context.Background(), "u-1", runner.RejectDailyQuota, true); got != 3 {
		t.Fatalf("batas kuota = %d, ingin 3 dari pengaturan", got)
	}
}

// TestKuotaAngkaDijepitPagar memastikan angka dari pengaturan tetap dijepit
// pada batas aman, sama seperti yang dilakukan Runner saat menahan setup.
func TestKuotaAngkaDijepitPagar(t *testing.T) {
	p := publisherUji(flagUji{settings.KeyMaxDailyTrades: 99})

	if got := p.kuotaAngka(context.Background(), "u-1", runner.RejectDailyQuota, true); got != 3 {
		t.Fatalf("batas kuota = %d, ingin dijepit ke 3", got)
	}
}

// TestKuotaAngkaHanyaUntukGrupKuota memastikan alasan lain tidak memicu
// pembacaan kuota yang tidak relevan.
func TestKuotaAngkaHanyaUntukGrupKuota(t *testing.T) {
	p := publisherUji(flagUji{settings.KeyMaxDailyTrades: 3})

	if got := p.kuotaAngka(context.Background(), "u-1", runner.RejectSlippage, true); got != 0 {
		t.Fatalf("alasan non-kuota tidak boleh mengisi angka kuota, dapat %d", got)
	}
}

type cycleStore struct {
	*storeUji
	position *model.Position
}

func (s *cycleStore) HitungKejadianSlot(context.Context, string, string, string, string, string) (int, error) {
	return 0, nil
}
func (s *cycleStore) FindOpenPositionBySignal(context.Context, string, string) (*model.Position, error) {
	return s.position, nil
}
func (s *cycleStore) FindUserByID(context.Context, string) (*model.User, error) {
	chat := "123"
	return &model.User{TelegramChatID: &chat, NotifyEnabled: true}, nil
}

type cycleTransport func(*http.Request) (*http.Response, error)

func (f cycleTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPublisherConsumedRetainsSetupCycle(t *testing.T) {
	for _, withPosition := range []bool{false, true} {
		t.Run(map[bool]string{false: "signal-fallback", true: "existing-position"}[withPosition], func(t *testing.T) {
			signal := &model.Signal{SignalID: "sig-1", Symbol: "EURUSD", Direction: "BUY", Htf: "H4", Ltf: "M1"}
			store := &cycleStore{storeUji: &storeUji{signals: map[string]*model.Signal{"sig-1": signal}}}
			if withPosition {
				store.position = &model.Position{Symbol: signal.Symbol, Direction: signal.Direction, SourceSignalID: signal.SignalID}
			}
			var text string
			previous := http.DefaultTransport
			http.DefaultTransport = cycleTransport(func(r *http.Request) (*http.Response, error) {
				var payload struct {
					Text string `json:"text"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					return nil, err
				}
				text = payload.Text
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true,"result":{"message_id":1}}`)), Header: make(http.Header)}, nil
			})
			defer func() { http.DefaultTransport = previous }()
			p := publisherUji(nil)
			p.store = store
			client := telegram.NewClient(telegram.Config{BotToken: "test"}, p.logger)
			p.notifier = notify.New(store, nil, client, nil, p.logger, 1)
			p.PublishSignal(context.Background(), "u-1", "sig-1", model.SignalStatusConsumed, "")
			p.notifier.Close()
			if !strings.Contains(text, "EURUSD · H4 ➔ M1") || !strings.Contains(text, "Setup: H4 ➔ M1\nCycle: DAILY\n") {
				t.Fatalf("konteks consumed hilang: %s", text)
			}
		})
	}
}

// TestFormatClosedMenampilkanTimeframe memastikan pesan posisi ditutup
// menyebut pasangan timeframe dari sinyal asalnya.
func TestFormatClosedMenampilkanTimeframe(t *testing.T) {
	msg := telegram.FormatClosedMessage(telegram.ClosedParams{
		Symbol: "XAUUSD", Direction: "SELL", Htf: "H4", Ltf: "M15",
		Result: "Target tercapai", EntryPrice: 3880.24, ExitPrice: 3852.64,
		ProfitLoss: 143.5, OutcomeR: 2, Duration: "3 jam 12 menit",
	})

	for _, ingin := range []string{"XAUUSD · H4 ➔ M15", "Tahap 6/6", "2,00R", "$143,50"} {
		if !strings.Contains(msg.Text, ingin) {
			t.Errorf("pesan closed tidak memuat %q:\n%s", ingin, msg.Text)
		}
	}
}
