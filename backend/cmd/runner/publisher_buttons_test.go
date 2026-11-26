package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/telegram"
)

type cycleMessageStore struct {
	repository.TelegramMessageRepository
	cleared []string
}

func (s *cycleMessageStore) BelumDibersihkan(context.Context, string) ([]model.TelegramMessage, error) {
	return []model.TelegramMessage{{ID: "own", UserID: "u-1", ChatID: "123", MessageID: 1}, {ID: "foreign", UserID: "u-2", ChatID: "456", MessageID: 2}}, nil
}
func (s *cycleMessageStore) TandaiDibersihkan(_ context.Context, id string, _ time.Time) error {
	s.cleared = append(s.cleared, id)
	return nil
}

func TestTerminalEditRetainsSetupCycle(t *testing.T) {
	for _, missing := range []bool{false, true} {
		p := publisherUji(nil)
		p.store = &storeUji{signals: map[string]*model.Signal{}}
		if !missing {
			p.store = &storeUji{signals: map[string]*model.Signal{"sig-1": {Symbol: "EURUSD", Htf: "H4", Ltf: "M1"}}}
		}
		messages := &cycleMessageStore{}
		p.jejak = messages
		var text string
		calls := 0
		previous := http.DefaultTransport
		http.DefaultTransport = cycleTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			var payload struct {
				Text   string `json:"text"`
				ChatID string `json:"chat_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				return nil, err
			}
			if payload.ChatID != "123" {
				t.Errorf("foreign chat edited: %s", payload.ChatID)
			}
			text = payload.Text
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
		})
		p.tg = telegram.NewClient(telegram.Config{BotToken: "test"}, p.logger)
		p.lepasTombol(context.Background(), "u-1", "sig-1", model.SignalStatusExpired, "")
		http.DefaultTransport = previous
		want := "Setup: H4 ➔ M1\nCycle: DAILY\n"
		if missing {
			want = "Setup: Tidak diketahui\nCycle: Tidak diketahui\n"
		}
		if calls != 1 || len(messages.cleared) != 1 || messages.cleared[0] != "own" || !strings.Contains(text, want) {
			t.Fatalf("terminal edit: calls=%d cleared=%v text=%s", calls, messages.cleared, text)
		}
	}
}

func TestStatusTerminal(t *testing.T) {
	terminal := []string{
		model.SignalStatusConsumed, model.SignalStatusCancelled,
		model.SignalStatusExpired, model.SignalStatusSuperseded,
		model.SignalStatusRejected, model.SignalStatusSkipped,
	}
	for _, status := range terminal {
		if !statusTerminal(status) {
			t.Fatalf("%s harus dianggap status akhir", status)
		}
	}

	// Status menengah masih bisa menjadi entry, tombolnya tidak boleh dilepas.
	for _, status := range []string{
		model.SignalStatusReadyToEntry,
		model.SignalStatusWaitLTFCISD,
		model.SignalStatusWaitHTFCandleClose,
	} {
		if statusTerminal(status) {
			t.Fatalf("%s bukan status akhir", status)
		}
	}
}

func TestAlasanTidakBerlaku(t *testing.T) {
	if got := alasanTidakBerlaku(model.SignalStatusRejected, "Harga sudah lewat."); got != "Harga sudah lewat." {
		t.Fatalf("alasan dari Runner harus dipakai apa adanya, dapat %q", got)
	}
	if got := alasanTidakBerlaku(model.SignalStatusExpired, ""); got == "" {
		t.Fatal("status akhir harus punya kalimat penjelas")
	}
	if got := alasanTidakBerlaku("STATUS_ASING", ""); got != "" {
		t.Fatalf("status tak dikenal tidak perlu kalimat, dapat %q", got)
	}
}
