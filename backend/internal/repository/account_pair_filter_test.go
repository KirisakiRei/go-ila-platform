package repository

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ila/backend/internal/model"
)

func TestSubscriptionRepository_AccountPairFilterAndDisableAuto(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.AutoMigrate(&model.Subscription{}, &model.AccountPairTimeframe{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}

	ctx := context.Background()
	subRepo := NewSubscriptionRepository(db)
	tfRepo := NewAccountPairTimeframeRepository(db)

	userID := uuid.NewString()
	accID := uuid.NewString()

	// Seed watchlist global (broker_account_id IS NULL).
	for _, sym := range []string{"XAUUSD", "EURUSD", "GBPUSD"} {
		if err := subRepo.Create(ctx, &model.Subscription{
			UserID:   userID,
			Symbol:   sym,
			IsActive: true,
		}); err != nil {
			t.Fatalf("seed global %s: %v", sym, err)
		}
	}

	// Sebelum pernah dikonfigurasi, akun mewarisi watchlist global.
	initial, err := subRepo.ListActiveByAccount(ctx, userID, accID)
	if err != nil {
		t.Fatalf("ListActiveByAccount awal: %v", err)
	}
	if len(initial) != 3 {
		t.Fatalf("awal = %d simbol, ingin 3", len(initial))
	}

	// Seed saklar timeframe (watch=true, auto=true) untuk XAUUSD & EURUSD.
	if err := tfRepo.Ensure(ctx, []model.AccountPairTimeframe{
		{BrokerAccountID: accID, UserID: userID, Symbol: "XAUUSD", Strategy: "ila-2.0", Htf: "H4", Ltf: "M15", IsWatch: true, IsAuto: true},
		{BrokerAccountID: accID, UserID: userID, Symbol: "EURUSD", Strategy: "ila-2.0", Htf: "H4", Ltf: "M15", IsWatch: true, IsAuto: true},
	}); err != nil {
		t.Fatalf("seed timeframe: %v", err)
	}

	// Pilih HANYA XAUUSD pada akun ini.
	if err := subRepo.ReplaceAccountPairs(ctx, userID, accID, []string{"XAUUSD"}); err != nil {
		t.Fatalf("ReplaceAccountPairs XAUUSD: %v", err)
	}
	if err := tfRepo.DisableAutoExcept(ctx, accID, []string{"XAUUSD"}); err != nil {
		t.Fatalf("DisableAutoExcept: %v", err)
	}

	after, err := subRepo.ListActiveByAccount(ctx, userID, accID)
	if err != nil {
		t.Fatalf("ListActiveByAccount setelah simpan: %v", err)
	}
	got := make([]string, 0, len(after))
	for _, s := range after {
		got = append(got, s.Symbol)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != "XAUUSD" {
		t.Fatalf("setelah simpan hanya XAUUSD, dapat %v", got)
	}

	// Watchlist global tetap utuh (3 simbol aktif).
	allActive, err := subRepo.ListActiveByUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListActiveByUser: %v", err)
	}
	uniq := map[string]bool{}
	for _, s := range allActive {
		uniq[s.Symbol] = true
	}
	if len(uniq) != 3 {
		t.Fatalf("watchlist global seharusnya tetap 3 simbol, dapat %v", uniq)
	}

	// Cek timeframe: XAUUSD tetap auto=true, EURUSD jadi auto=false tapi watch tetap true.
	tfRows, err := tfRepo.ListByAccount(ctx, accID)
	if err != nil {
		t.Fatalf("ListByAccount: %v", err)
	}
	for _, row := range tfRows {
		if row.Symbol == "XAUUSD" && (!row.IsWatch || !row.IsAuto) {
			t.Fatalf("XAUUSD seharusnya watch=true auto=true, dapat %+v", row)
		}
		if row.Symbol == "EURUSD" && (!row.IsWatch || row.IsAuto) {
			t.Fatalf("EURUSD seharusnya watch=true auto=false, dapat %+v", row)
		}
	}

	// Batal pilih semua pair -> ListActiveByAccount wajib kosong (tidak balik ke global).
	if err := subRepo.ReplaceAccountPairs(ctx, userID, accID, []string{}); err != nil {
		t.Fatalf("ReplaceAccountPairs kosong: %v", err)
	}
	if err := tfRepo.DisableAutoExcept(ctx, accID, []string{}); err != nil {
		t.Fatalf("DisableAutoExcept kosong: %v", err)
	}
	empty, err := subRepo.ListActiveByAccount(ctx, userID, accID)
	if err != nil {
		t.Fatalf("ListActiveByAccount kosong: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("setelah batal pilih semua, ingin 0 simbol, dapat %d", len(empty))
	}
	tfRows, err = tfRepo.ListByAccount(ctx, accID)
	if err != nil {
		t.Fatalf("ListByAccount kosong: %v", err)
	}
	for _, row := range tfRows {
		if !row.IsWatch || row.IsAuto {
			t.Fatalf("setelah batal pilih semua, %s seharusnya watch=true auto=false, dapat %+v", row.Symbol, row)
		}
	}
}
