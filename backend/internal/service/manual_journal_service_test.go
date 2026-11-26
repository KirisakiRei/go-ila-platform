package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func manualTestService(t *testing.T) (*ManualJournalService, *gorm.DB, string, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.AutoMigrate(&model.ManualJournalAccount{}, &model.ManualJournalCapital{}, &model.ManualJournalEntry{}, &model.ManualJournalScreenshot{}, &model.ManualJournalImportToken{}); err != nil {
		t.Fatal(err)
	}
	s := NewManualJournalService(repository.NewManualJournalRepository(db), t.TempDir())
	user := uuid.NewString()
	a, err := s.SaveAccount(context.Background(), user, "", dto.ManualJournalAccountInput{Name: "Manual", Currency: "USD", InitialCapital: "1000", InitialDate: "2026-01-01", Checklist: []string{"Plan", "Risk"}})
	if err != nil {
		t.Fatal(err)
	}
	return s, db, user, a.ID
}
func manualPtr(s string) *string { return &s }
func manualClosed(account, pnl string) dto.ManualJournalEntryInput {
	opened := time.Date(2026, 1, 1, 16, 0, 0, 0, time.UTC)
	closed := opened.Add(2 * time.Hour)
	return dto.ManualJournalEntryInput{AccountID: account, Status: "closed", Asset: "XAUUSD", Side: "long", OpenedAt: &opened, ClosedAt: &closed, EntryPrice: manualPtr("2000.12"), Size: manualPtr("0.1"), SizeUnit: "lot", NetPNL: manualPtr(pnl), Tags: []string{"breakout"}, Setup: "Breakout", Session: "NY", Checklist: []model.ManualJournalCheck{{Label: "Plan", Checked: true}}, Notes: "=HYPERLINK(\"evil\")"}
}
func TestManualJournalExactStats(t *testing.T) {
	rows := []model.ManualJournalEntry{}
	for i, p := range []string{"100.0001", "-40.0001", "-80", "0", "30"} {
		closed := time.Date(2026, 1, 1+i, 18, 0, 0, 0, time.UTC)
		rows = append(rows, model.ManualJournalEntry{Base: model.Base{ID: fmt.Sprint(i)}, Status: "closed", ClosedAt: &closed, NetPNL: manualPtr(p), Asset: "GOLD", Tags: []string{"A"}})
	}
	rows = append(rows, model.ManualJournalEntry{Status: "open", NetPNL: manualPtr("999")}, model.ManualJournalEntry{Status: "draft"})
	got, err := manualCalculate(rows)
	if err != nil {
		t.Fatal(err)
	}
	if got.Trades != 5 || got.Wins != 2 || got.Losses != 2 || got.Breakeven != 1 || got.NetPNL != "10.0000" || got.MaxDrawdown != "120.0001" || got.WinRate != "50.0000" || got.AverageWin != "65.0001" || got.AverageLoss != "60.0001" {
		t.Fatalf("stats: %+v", got)
	}
	if got.ProfitFactor == nil || *got.ProfitFactor != "1.0833" {
		t.Fatalf("PF: %v", got.ProfitFactor)
	}
	if got.Curve[0].Date != "2026-01-02" || got.Calendar["2026-01-02"].NetPNL != "100.0001" || got.Breakdowns["tag"]["A"].Trades != 5 {
		t.Fatalf("WIB/groups: %+v", got)
	}
	noLoss, err := manualCalculate(rows[:1])
	if err != nil || noLoss.ProfitFactor != nil {
		t.Fatalf("no loss PF: %+v %v", noLoss, err)
	}
}
func TestManualJournalValidationOwnershipAndSnapshots(t *testing.T) {
	s, _, user, account := manualTestService(t)
	ctx := context.Background()
	other := uuid.NewString()
	in := manualClosed(account, "0")
	entry, err := s.SaveEntry(ctx, user, "", in)
	if err != nil {
		t.Fatal(err)
	}
	if len(entry.Checklist) != 2 || !entry.Checklist[0].Checked || entry.Checklist[1].Checked {
		t.Fatalf("snapshot: %+v", entry.Checklist)
	}
	for _, get := range []func() error{func() error { _, e := s.GetEntry(ctx, other, entry.ID); return e }, func() error { _, e := s.SaveEntry(ctx, other, "", in); return e }, func() error { return s.DeleteEntry(ctx, other, entry.ID) }, func() error { _, e := s.GetAccount(ctx, other, account); return e }} {
		if err := get(); !errors.Is(err, apperror.ErrNotFound) {
			t.Fatalf("isolation: %v", err)
		}
	}
	_, err = s.SaveAccount(ctx, user, account, dto.ManualJournalAccountInput{Name: "Changed", Currency: "USD", InitialCapital: "1000", InitialDate: "2026-01-01", Checklist: []string{"New"}})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := s.SaveEntry(ctx, user, entry.ID, in)
	if err != nil || len(updated.Checklist) != 2 || updated.Checklist[0].Label != "Plan" {
		t.Fatalf("snapshot mutated: %+v %v", updated, err)
	}
	tests := []struct {
		name   string
		change func(*dto.ManualJournalEntryInput)
	}{
		{"precision", func(v *dto.ManualJournalEntryInput) { v.NetPNL = manualPtr("0.00001") }},
		{"nondecimal", func(v *dto.ManualJournalEntryInput) { v.NetPNL = manualPtr("1e5") }},
		{"negative risk", func(v *dto.ManualJournalEntryInput) { v.RiskAmount = manualPtr("-1") }},
		{"missing pnl", func(v *dto.ManualJournalEntryInput) { v.NetPNL = nil }},
		{"bad chronology", func(v *dto.ManualJournalEntryInput) {
			v.ClosedAt = v.OpenedAt
			d := v.OpenedAt.Add(time.Hour)
			v.OpenedAt = &d
		}},
		{"foreign URL", func(v *dto.ManualJournalEntryInput) {
			v.TradingViewURLs = []string{"https://tradingview.com.evil.test/chart"}
		}},
		{"zero size", func(v *dto.ManualJournalEntryInput) { v.Size = manualPtr("0") }},
		{"open pnl", func(v *dto.ManualJournalEntryInput) { v.Status = "open" }},
		{"invalid checklist", func(v *dto.ManualJournalEntryInput) { v.Checklist = []model.ManualJournalCheck{{Label: "Unknown"}} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := in
			tt.change(&v)
			if _, err := s.SaveEntry(ctx, user, entry.ID, v); !errors.Is(err, apperror.ErrBadRequest) {
				t.Fatalf("accepted: %v", err)
			}
		})
	}
	if err := s.ArchiveAccount(ctx, user, account); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveEntry(ctx, user, entry.ID, in); !errors.Is(err, apperror.ErrConflict) {
		t.Fatalf("archived: %v", err)
	}
	if _, err := s.GetEntry(ctx, user, entry.ID); err != nil {
		t.Fatal("archive removed entries", err)
	}
}
func TestManualJournalWholeFilterCapitalAndCurrency(t *testing.T) {
	s, _, user, account := manualTestService(t)
	ctx := context.Background()
	for _, p := range []string{"0.1", "0.2", "-0.1"} {
		if _, err := s.SaveEntry(ctx, user, "", manualClosed(account, p)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.SaveCapital(ctx, user, "", dto.ManualJournalCapitalInput{AccountID: account, Kind: "deposit", Amount: "9999", Date: "2026-01-01"}); err != nil {
		t.Fatal(err)
	}
	list, meta, err := s.ListEntries(ctx, user, dto.ManualJournalFilter{PerPage: 1, Tag: "breakout", From: "2026-01-02", To: "2026-01-02"})
	if err != nil || len(list) != 1 || meta.TotalItems != 3 {
		t.Fatalf("list %v %v %v", list, meta, err)
	}
	got, err := s.Stats(ctx, user, dto.ManualJournalFilter{PerPage: 1, Tag: "breakout"})
	if err != nil || got.Trades != 3 || got.NetPNL != "0.2000" {
		t.Fatalf("whole set/capital %+v %v", got, err)
	}
	empty, err := s.Stats(ctx, user, dto.ManualJournalFilter{From: "2026-01-01", To: "2026-01-01"})
	if err != nil || empty.Trades != 0 {
		t.Fatalf("WIB filter %+v %v", empty, err)
	}
	a, err := s.SaveAccount(ctx, user, "", dto.ManualJournalAccountInput{Name: "Rupiah", Currency: "IDR", InitialCapital: "0", InitialDate: "2026-01-01"})
	if err != nil {
		t.Fatal(err)
	}
	in := manualClosed(a.ID, "10")
	in.Checklist = nil
	if _, err := s.SaveEntry(ctx, user, "", in); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Stats(ctx, user, dto.ManualJournalFilter{}); !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("mixed currency accepted: %v", err)
	}
	if _, err := s.Stats(ctx, user, dto.ManualJournalFilter{AccountID: account}); err != nil {
		t.Fatal(err)
	}
}
func TestManualJournalCSVAtomicAndFormulaSafe(t *testing.T) {
	s, db, user, account := manualTestService(t)
	ctx := context.Background()
	if _, err := s.SaveEntry(ctx, user, "", manualClosed(account, "-1.25")); err != nil {
		t.Fatal(err)
	}
	data, err := s.ExportCSV(ctx, user, dto.ManualJournalFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("'=HYPERLINK")) || !bytes.Contains(data, []byte("'-1.25")) {
		t.Fatalf("unsafe CSV: %s", data)
	}
	preview, err := s.PreviewCSV(ctx, user, string(data))
	if err != nil || preview.Count != 1 || preview.Rows[0].Notes != "=HYPERLINK(\"evil\")" || *preview.Rows[0].NetPNL != "-1.25" {
		t.Fatalf("roundtrip %+v %v", preview, err)
	}
	csvRows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	bad := append([]string(nil), csvRows[1]...)
	bad[12] = "not-money"
	csvRows = append(csvRows, bad)
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	if err := w.WriteAll(csvRows); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfirmCSV(ctx, user, b.String(), preview.Token); !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("invalid import: %v", err)
	}
	var count int64
	if err := db.Model(&model.ManualJournalEntry{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("partial import count=%d %v", count, err)
	}
	n, err := s.ConfirmCSV(ctx, user, string(data), preview.Token)
	if err != nil || n != 1 {
		t.Fatalf("confirm %d %v", n, err)
	}
	if _, err := s.PreviewCSV(ctx, uuid.NewString(), string(data)); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("foreign import: %v", err)
	}
	for _, text := range []string{"=1", " +1", "\tbad", "\n=1", "'literal", "'-literal", "normal"} {
		if got := manualCSVUnescape(manualCSVSafe(text)); got != text {
			t.Fatalf("escape roundtrip %q => %q", text, got)
		}
	}
}
func TestManualJournalCSVHistoricalSnapshotRoundtrip(t *testing.T) {
	s, db, user, account := manualTestService(t)
	ctx := context.Background()
	in := manualClosed(account, "-12.3456")
	in.Notes = strings.Repeat("\"<&>\n", 2000)
	original, err := s.SaveEntry(ctx, user, "", in)
	if err != nil {
		t.Fatal(err)
	}
	data, err := s.ExportCSV(ctx, user, dto.ManualJournalFilter{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.SaveAccount(ctx, user, account, dto.ManualJournalAccountInput{Name: "Updated", Currency: "USD", InitialCapital: "1000", InitialDate: "2026-01-01", Checklist: []string{"New template"}})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := s.PreviewCSV(ctx, user, string(data))
	if err != nil {
		t.Fatal(err)
	}
	// Changing the template between preview and confirm must not alter the imported snapshot either.
	_, err = s.SaveAccount(ctx, user, account, dto.ManualJournalAccountInput{Name: "Updated", Currency: "USD", InitialCapital: "1000", InitialDate: "2026-01-01", Checklist: []string{"Another template"}})
	if err != nil {
		t.Fatal(err)
	}
	if n, err := s.ConfirmCSV(ctx, user, string(data), preview.Token); err != nil || n != 1 {
		t.Fatalf("historical confirm %d %v", n, err)
	}
	var imported model.ManualJournalEntry
	if err := db.Where("user_id = ? AND id <> ?", user, original.ID).First(&imported).Error; err != nil {
		t.Fatal(err)
	}
	if imported.Notes != in.Notes || len(imported.Checklist) != 2 || imported.Checklist[0].Label != "Plan" || !imported.Checklist[0].Checked || imported.Checklist[1].Label != "Risk" || imported.Checklist[1].Checked {
		t.Fatalf("snapshot/long notes lost: %+v", imported)
	}
	records, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	tooMany := make([]model.ManualJournalCheck, 31)
	for i := range tooMany {
		tooMany[i].Label = fmt.Sprintf("label-%d", i)
	}
	tooManyJSON, err := json.Marshal(tooMany)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{string(tooManyJSON), `[{"label":"","checked":true}]`, `[{"label":"Plan","checked":true},{"label":"Plan","checked":false}]`, `[{"label":"` + strings.Repeat("x", 201) + `","checked":false}]`, strings.Repeat("[{", 31)} {
		records[1][20] = bad
		var b bytes.Buffer
		w := csv.NewWriter(&b)
		if err := w.WriteAll(records); err != nil {
			t.Fatal(err)
		}
		if _, err := s.PreviewCSV(ctx, user, b.String()); !errors.Is(err, apperror.ErrBadRequest) {
			t.Fatalf("invalid snapshot accepted: %s %v", bad, err)
		}
	}
}

func TestManualJournalCSVExportImportLimits(t *testing.T) {
	for _, tt := range []struct {
		name  string
		count int
		notes string
		ok    bool
	}{
		{"maximum rows", 500, "small", true},
		{"too many rows", 501, "small", false},
		{"long notes fit", 90, strings.Repeat("x", 10000), true},
		{"long notes exceed bytes", 110, strings.Repeat("x", 10000), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, db, user, account := manualTestService(t)
			ctx := context.Background()
			rows := make([]model.ManualJournalEntry, tt.count)
			for i := range rows {
				rows[i] = model.ManualJournalEntry{UserID: user, AccountID: account, Status: "draft", Notes: tt.notes}
			}
			if err := db.CreateInBatches(&rows, 50).Error; err != nil {
				t.Fatal(err)
			}
			data, err := s.ExportCSV(ctx, user, dto.ManualJournalFilter{})
			if !tt.ok {
				if !errors.Is(err, apperror.ErrBadRequest) || data != nil {
					t.Fatalf("export must fail without partial output: %d %v", len(data), err)
				}
				return
			}
			if err != nil || len(data) > ManualJournalMaxCSV {
				t.Fatalf("export size %d %v", len(data), err)
			}
			preview, err := s.PreviewCSV(ctx, user, string(data))
			if err != nil || preview.Count != tt.count {
				t.Fatalf("preview %+v %v", preview, err)
			}
			if n, err := s.ConfirmCSV(ctx, user, string(data), preview.Token); err != nil || n != tt.count {
				t.Fatalf("confirm %d %v", n, err)
			}
			var count int64
			if err := db.Model(&model.ManualJournalEntry{}).Where("user_id = ? AND notes = ?", user, tt.notes).Count(&count).Error; err != nil || count != int64(2*tt.count) {
				t.Fatalf("roundtrip rows/notes %d %v", count, err)
			}
		})
	}
}

func TestManualJournalPersistentImportTokens(t *testing.T) {
	s, db, user, account := manualTestService(t)
	ctx := context.Background()
	if _, err := s.SaveEntry(ctx, user, "", manualClosed(account, "1")); err != nil {
		t.Fatal(err)
	}
	data, err := s.ExportCSV(ctx, user, dto.ManualJournalFilter{})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := s.PreviewCSV(ctx, user, string(data))
	if err != nil {
		t.Fatal(err)
	}
	if preview.Token == "" || time.Until(preview.ExpiresAt) > 15*time.Minute || time.Until(preview.ExpiresAt) < 14*time.Minute {
		t.Fatalf("token expiry %+v", preview)
	}
	if _, err := s.ConfirmCSV(ctx, user, string(data), ""); !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("missing token: %v", err)
	}
	altered := strings.Replace(string(data), "XAUUSD", "EURUSD", 1)
	if _, err := s.ConfirmCSV(ctx, user, altered, preview.Token); !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("altered payload: %v", err)
	}
	// Valid foreign CSV isolates token ownership from account validation.
	other := uuid.NewString()
	a, err := s.SaveAccount(ctx, other, "", dto.ManualJournalAccountInput{Name: "Other", Currency: "USD", InitialCapital: "0", InitialDate: "2026-01-01", Checklist: []string{"Plan", "Risk"}})
	if err != nil {
		t.Fatal(err)
	}
	foreignCSV := strings.ReplaceAll(string(data), account, a.ID)
	if _, err := s.ConfirmCSV(ctx, other, foreignCSV, preview.Token); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("foreign token: %v", err)
	}
	restarted := NewManualJournalService(repository.NewManualJournalRepository(db), s.dir)
	if n, err := restarted.ConfirmCSV(ctx, user, string(data), preview.Token); err != nil || n != 1 {
		t.Fatalf("restart confirm: %d %v", n, err)
	}
	if _, err := s.ConfirmCSV(ctx, user, string(data), preview.Token); !errors.Is(err, apperror.ErrConflict) {
		t.Fatalf("replay: %v", err)
	}
	expired, err := s.PreviewCSV(ctx, user, string(data))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.ManualJournalImportToken{}).Where("id = ?", expired.Token).Update("expires_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfirmCSV(ctx, user, string(data), expired.Token); !errors.Is(err, apperror.ErrConflict) {
		t.Fatalf("expired: %v", err)
	}
	var count int64
	if err := db.Model(&model.ManualJournalEntry{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("duplicate/rejected insert: %d %v", count, err)
	}
}

func TestManualJournalConcurrentConfirm(t *testing.T) {
	s, db, user, account := manualTestService(t)
	ctx := context.Background()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := s.SaveEntry(ctx, user, "", manualClosed(account, "1")); err != nil {
		t.Fatal(err)
	}
	data, err := s.ExportCSV(ctx, user, dto.ManualJournalFilter{})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := s.PreviewCSV(ctx, user, string(data))
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := s.ConfirmCSV(ctx, user, string(data), preview.Token)
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, apperror.ErrConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("concurrent success=%d conflict=%d", success, conflict)
	}
	var count int64
	if err := db.Model(&model.ManualJournalEntry{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("concurrent duplicate: %d %v", count, err)
	}
}

func TestManualJournalWinrateExcludesBreakeven(t *testing.T) {
	now := time.Now()
	rows := []model.ManualJournalEntry{{Status: "closed", ClosedAt: &now, NetPNL: manualPtr("0")}}
	stats, err := manualCalculate(rows)
	if err != nil || stats.WinRate != "0.0000" || stats.Breakeven != 1 {
		t.Fatalf("BE only: %+v %v", stats, err)
	}
	rows = append(rows, model.ManualJournalEntry{Status: "closed", ClosedAt: &now, NetPNL: manualPtr("1")})
	stats, err = manualCalculate(rows)
	if err != nil || stats.WinRate != "100.0000" || stats.Trades != 2 || stats.Breakeven != 1 {
		t.Fatalf("BE denominator: %+v %v", stats, err)
	}
}

func TestManualJournalBalance(t *testing.T) {
	s, db, user, account := manualTestService(t)
	ctx := context.Background()
	for _, in := range []dto.ManualJournalCapitalInput{
		{AccountID: account, Kind: "deposit", Amount: "100.1234", Date: "2026-01-01"},
		{AccountID: account, Kind: "withdrawal", Amount: "50.0001", Date: "2026-01-02"},
	} {
		if _, err := s.SaveCapital(ctx, user, "", in); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{"0.1001", "-0.2001"} {
		if _, err := s.SaveEntry(ctx, user, "", manualClosed(account, p)); err != nil {
			t.Fatal(err)
		}
	}
	// Open PNL and foreign-owned rows cannot contribute, even with the same account ID.
	for _, row := range []model.ManualJournalEntry{
		{UserID: user, AccountID: account, Status: "open", NetPNL: manualPtr("999")},
		{UserID: uuid.NewString(), AccountID: account, Status: "closed", NetPNL: manualPtr("999")},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	assertBalance := func(expected string) {
		t.Helper()
		a, err := s.GetAccount(ctx, user, account)
		if err != nil || a.CurrentRealizedBalance == nil || *a.CurrentRealizedBalance != expected || !a.BalanceComplete || a.UnknownClosedPNL != 0 {
			t.Fatalf("balance %+v %v", a, err)
		}
	}
	assertBalance("1050.0233")
	rows, _, err := s.ListAccounts(ctx, user, dto.ManualJournalFilter{})
	if err != nil || len(rows) != 1 || rows[0].CurrentRealizedBalance == nil || *rows[0].CurrentRealizedBalance != "1050.0233" {
		t.Fatalf("list balances %+v %v", rows, err)
	}
	if err := s.ArchiveAccount(ctx, user, account); err != nil {
		t.Fatal(err)
	}
	assertBalance("1050.0233")
	unknown := model.ManualJournalEntry{UserID: user, AccountID: account, Status: "closed"}
	if err := db.Create(&unknown).Error; err != nil {
		t.Fatal(err)
	}
	a, err := s.GetAccount(ctx, user, account)
	if err != nil || a.CurrentRealizedBalance != nil || a.BalanceComplete || a.UnknownClosedPNL != 1 {
		t.Fatalf("unknown fabricated balance %+v %v", a, err)
	}
	if err := db.Delete(&unknown).Error; err != nil {
		t.Fatal(err)
	}
	assertBalance("1050.0233")
}

func TestManualJournalLosingStreakAndFilters(t *testing.T) {
	s, _, user, account := manualTestService(t)
	ctx := context.Background()
	// Insert out of chronological order; streak must follow close timestamps, not insert order.
	for _, i := range []int{3, 0, 2, 1} {
		in := manualClosed(account, "-1")
		if i == 0 {
			in.NetPNL = manualPtr("5")
		}
		closed := in.ClosedAt.Add(time.Duration(i) * time.Hour)
		in.ClosedAt = &closed
		in.Emotions = []string{"fear"}
		in.Mistakes = []string{"rush"}
		if _, err := s.SaveEntry(ctx, user, "", in); err != nil {
			t.Fatal(err)
		}
	}
	f := dto.ManualJournalFilter{Emotion: "fear", Mistake: "rush"}
	insights, err := s.Insights(ctx, user, f)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range insights {
		if v.Code == "losing_streak" {
			found = true
			if v.Count != 3 {
				t.Fatalf("streak count %+v", v)
			}
		}
	}
	if !found {
		t.Fatalf("missing streak %+v", insights)
	}
	stats, err := s.Stats(ctx, user, f)
	if err != nil || stats.Trades != 4 {
		t.Fatalf("stats filter %+v %v", stats, err)
	}
	f.Mistake = "other"
	stats, err = s.Stats(ctx, user, f)
	if err != nil || stats.Trades != 0 {
		t.Fatalf("AND filter %+v %v", stats, err)
	}
	for _, p := range []string{"0", "1"} {
		in := manualClosed(account, p)
		closed := in.ClosedAt.Add(10 * time.Hour)
		in.ClosedAt = &closed
		if _, err := s.SaveEntry(ctx, user, "", in); err != nil {
			t.Fatal(err)
		}
		insights, err = s.Insights(ctx, user, dto.ManualJournalFilter{})
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range insights {
			if v.Code == "losing_streak" {
				t.Fatalf("BE/win did not stop streak %+v", insights)
			}
		}
	}
	for _, f := range []dto.ManualJournalFilter{{Emotion: strings.Repeat("x", 101)}, {Mistake: strings.Repeat("x", 101)}} {
		if _, err := s.Filter(ctx, user, f); !errors.Is(err, apperror.ErrBadRequest) {
			t.Fatalf("oversize filter accepted %v", err)
		}
	}
}

func TestManualJournalScreenshotCleanup(t *testing.T) {
	for _, entryDelete := range []bool{false, true} {
		for _, failUnlink := range []bool{false, true} {
			t.Run(fmt.Sprintf("entry=%t/fail=%t", entryDelete, failUnlink), func(t *testing.T) {
				s, db, user, account := manualTestService(t)
				ctx := context.Background()
				entry, err := s.SaveEntry(ctx, user, "", manualClosed(account, "1"))
				if err != nil {
					t.Fatal(err)
				}
				png := append([]byte{137, 80, 78, 71, 13, 10, 26, 10}, make([]byte, 512)...)
				shot, err := s.UploadScreenshot(ctx, user, entry.ID, "screen.png", bytes.NewReader(png))
				if err != nil {
					t.Fatal(err)
				}
				var row model.ManualJournalScreenshot
				if err := db.WithContext(ctx).First(&row, "id = ?", shot.ID).Error; err != nil {
					t.Fatal(err)
				}
				path, err := s.screenshotPath(&row)
				if err != nil {
					t.Fatal(err)
				}
				if failUnlink {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(path, 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(path, "blocker"), []byte("block unlink"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if entryDelete {
					err = s.DeleteEntry(ctx, user, entry.ID)
				} else {
					err = s.DeleteScreenshot(ctx, user, entry.ID, shot.ID)
				}
				if (err != nil) != failUnlink {
					t.Fatalf("delete error = %v", err)
				}
				if _, err := s.OpenScreenshot(ctx, user, entry.ID, shot.ID); !errors.Is(err, apperror.ErrNotFound) {
					t.Fatalf("deleted screenshot accessible: %v", err)
				}
				if entryDelete {
					if _, err := s.GetEntry(ctx, user, entry.ID); !errors.Is(err, apperror.ErrNotFound) {
						t.Fatalf("entry still visible: %v", err)
					}
					late := model.ManualJournalScreenshot{UserID: user, EntryID: entry.ID, StoredAs: "late.png"}
					if err := s.repo.Create(ctx, &late); !errors.Is(err, apperror.ErrNotFound) {
						t.Fatalf("upload to deleted entry accepted: %v", err)
					}
				}
				if failUnlink {
					if err := db.WithContext(ctx).Unscoped().First(&row, "id = ?", shot.ID).Error; err != nil || !row.DeletedAt.Valid {
						t.Fatalf("lost tombstone: %+v %v", row, err)
					}
					// A new service instance must retry without the deleted parent or requesting owner.
					s = NewManualJournalService(repository.NewManualJournalRepository(db), s.dir)
					if err := s.CleanupScreenshots(ctx); err == nil {
						t.Fatal("unlink failure swallowed")
					}
					if err := os.Remove(filepath.Join(path, "blocker")); err != nil {
						t.Fatal(err)
					}
				}
				if err := s.CleanupScreenshots(ctx); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("file remains: %v", err)
				}
				if err := db.WithContext(ctx).Unscoped().First(&row, "id = ?", shot.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
					t.Fatalf("metadata not purged: %v", err)
				}
				if err := s.CleanupScreenshots(ctx); err != nil {
					t.Fatalf("retry not idempotent: %v", err)
				}
			})
		}
	}
}

func TestManualJournalRejectsNUL(t *testing.T) {
	s, _, user, account := manualTestService(t)
	ctx := context.Background()
	for _, field := range []string{"name", "broker", "checklist"} {
		t.Run(field, func(t *testing.T) {
			in := dto.ManualJournalAccountInput{Name: "Manual", Currency: "USD", InitialCapital: "1000", InitialDate: "2026-01-01"}
			switch field {
			case "name":
				in.Name = "bad\x00name"
			case "broker":
				in.Broker = "bad\x00broker"
			case "checklist":
				in.Checklist = []string{"bad\x00label"}
			}
			if _, err := s.SaveAccount(ctx, user, "", in); !errors.Is(err, apperror.ErrBadRequest) {
				t.Fatalf("NUL accepted: %v", err)
			}
		})
	}
	for _, field := range []string{"tags", "emotions", "mistakes", "urls", "checklist"} {
		t.Run(field, func(t *testing.T) {
			in := manualClosed(account, "1")
			switch field {
			case "tags":
				in.Tags = []string{"bad\x00tag"}
			case "emotions":
				in.Emotions = []string{"bad\x00emotion"}
			case "mistakes":
				in.Mistakes = []string{"bad\x00mistake"}
			case "urls":
				in.TradingViewURLs = []string{"https://www.tradingview.com/x/bad\x00"}
			case "checklist":
				in.Checklist[0].Label = "bad\x00label"
			}
			if _, err := s.SaveEntry(ctx, user, "", in); !errors.Is(err, apperror.ErrBadRequest) {
				t.Fatalf("NUL accepted: %v", err)
			}
		})
	}
}

func TestManualJournalCleanupBoundAndSafePath(t *testing.T) {
	s, db, user, account := manualTestService(t)
	ctx := context.Background()
	entry, err := s.SaveEntry(ctx, user, "", manualClosed(account, "1"))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []model.ManualJournalScreenshot{
		{EntryID: "..", StoredAs: "screen.png"},
		{EntryID: entry.ID, StoredAs: "../screen.png"},
		{EntryID: entry.ID, StoredAs: ".."},
		{EntryID: entry.ID, StoredAs: "screen.png:stream"},
	} {
		if _, err := s.screenshotPath(&row); err == nil {
			t.Fatalf("unsafe path accepted: %+v", row)
		}
	}
	rows := make([]model.ManualJournalScreenshot, 101)
	for i := range rows {
		rows[i] = model.ManualJournalScreenshot{UserID: user, EntryID: entry.ID, StoredAs: uuid.NewString() + ".png"}
	}
	if err := db.WithContext(ctx).Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteEntry(ctx, user, entry.ID); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.WithContext(ctx).Unscoped().Model(&model.ManualJournalScreenshot{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("cleanup not bounded: %d %v", count, err)
	}
	if err := s.CleanupScreenshots(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(ctx).Unscoped().Model(&model.ManualJournalScreenshot{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("cleanup did not finish: %d %v", count, err)
	}
}

func TestManualJournalPrivateScreenshots(t *testing.T) {
	s, _, user, account := manualTestService(t)
	ctx := context.Background()
	entry, err := s.SaveEntry(ctx, user, "", manualClosed(account, "1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UploadScreenshot(ctx, user, entry.ID, "x.svg", strings.NewReader("<svg>bad</svg>")); !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("svg accepted %v", err)
	}
	png := append([]byte{137, 80, 78, 71, 13, 10, 26, 10}, make([]byte, 512)...)
	if _, err := s.UploadScreenshot(ctx, user, entry.ID, "big.png", bytes.NewReader(append(png, make([]byte, ManualJournalMaxUpload)...))); !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("oversized accepted %v", err)
	}
	shot, err := s.UploadScreenshot(ctx, user, entry.ID, "../../screen.png", bytes.NewReader(png))
	if err != nil {
		t.Fatal(err)
	}
	if shot.FileName != "screen.png" {
		t.Fatal(shot.FileName)
	}
	if _, err := s.OpenScreenshot(ctx, uuid.NewString(), entry.ID, shot.ID); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("foreign screenshot %v", err)
	}
	file, err := s.OpenScreenshot(ctx, user, entry.ID, shot.ID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(file.File)
	closeErr := file.File.Close()
	if err != nil || closeErr != nil || !bytes.Equal(b, png) {
		t.Fatalf("file %v %v", err, closeErr)
	}
	if err := s.DeleteScreenshot(ctx, user, entry.ID, shot.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.OpenScreenshot(ctx, user, entry.ID, shot.ID); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("deleted visible %v", err)
	}
}
