package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/repository"
)

type menuJournal struct {
	filter  repository.JournalFilter
	limit   int
	user    string
	skipped []model.UserSignal
}

func (m *menuJournal) ListPositions(_ context.Context, user string, f repository.JournalFilter, limit int) ([]model.Position, error) {
	m.user = user
	m.filter = f
	m.limit = limit
	return nil, nil
}
func (*menuJournal) Stats(context.Context, string, repository.JournalFilter) (repository.JournalStats, error) {
	return repository.JournalStats{ClosedTP: 12}, nil
}
func (m *menuJournal) ListSkipped(_ context.Context, user string, f repository.JournalFilter, offset, limit int) ([]model.UserSignal, error) {
	m.user = user
	m.filter = f
	m.limit = limit
	return m.skipped, nil
}
func (m *menuJournal) CountSkipped(context.Context, string, repository.JournalFilter) (int, error) {
	return len(m.skipped), nil
}

func TestTelegramHistoryUsesBoundedDatabasePagination(t *testing.T) {
	repo := &menuJournal{}
	svc := &telegramMenuService{journal: repo}
	_, err := svc.Riwayat(context.Background(), &model.User{Base: model.Base{ID: "owner"}}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if repo.user != "owner" || !repo.filter.ClosedOnly || repo.filter.Offset != 5 || repo.limit != 5 {
		t.Fatalf("wrong query: %+v", repo)
	}
}

func TestTelegramSimulationUsesPersistedSnapshotAndOwner(t *testing.T) {
	now := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	data, err := json.Marshal(model.Signal{Symbol: "EURUSD", Direction: "SELL", Htf: "H4", Ltf: "M15", StrategyVersion: "ila-3.0"})
	if err != nil {
		t.Fatal(err)
	}
	repo := &menuJournal{skipped: []model.UserSignal{{Base: model.Base{ID: "sim"}, EvalSnapshot: string(data), EvalStatus: model.EvalRunning, EvalEntryAt: &now}}}
	svc := &telegramMenuService{journal: repo}
	msg, err := svc.Simulasi(context.Background(), &model.User{Base: model.Base{ID: "owner"}}, "https://rekavia.com/app/dashboard", 1)
	if err != nil {
		t.Fatal(err)
	}
	if repo.user != "owner" || repo.limit != 5 || !strings.Contains(msg.Text, "ILA 3.0") || !strings.Contains(msg.Text, "Sedang berjalan") {
		t.Fatalf("wrong simulation: %+v", msg)
	}
	for _, row := range msg.Buttons {
		for _, button := range row {
			if button.URL == "" && button.CallbackData == "" {
				t.Fatalf("invalid keyboard: %+v", button)
			}
		}
	}
}
