package runner

import (
	"context"
	"time"

	"ila/backend/internal/model"
)

// TelemetryInterval adalah jeda pembaruan saldo akun broker (plans/05 §9).
// Satu kali per menit cukup untuk dashboard dan tetap ringan di VPS 1 GB.
const TelemetryInterval = 60 * time.Second

// RefreshTelemetry membaca saldo setiap akun aktif lalu menyimpannya.
//
// Berbeda dari MonitorPositions yang berjalan rapat, telemetry boleh lambat:
// yang dibutuhkan user adalah gambaran saldo, bukan tick harga.
// Kegagalan satu akun tidak menghentikan akun lain.
func (s *Service) RefreshTelemetry(ctx context.Context) {
	accounts, err := s.store.ListActiveBrokerAccounts(ctx)
	if err != nil {
		s.logger.Error("gagal membaca akun aktif", "error", err.Error())
		return
	}

	for i := range accounts {
		if ctx.Err() != nil {
			return
		}
		s.refreshAccount(ctx, &accounts[i])
	}
}

// refreshAccount memperbarui satu akun, termasuk saat akun tidak terjangkau.
func (s *Service) refreshAccount(ctx context.Context, account *model.BrokerAccount) {
	now := s.opts.Clock()

	state := &model.BrokerAccountState{
		BrokerAccountID: account.ID,
		UserID:          account.UserID,
		Connected:       false,
		LastSyncedAt:    &now,
		UpdatedAt:       now,
	}

	// Semua akun punya klien, termasuk simulasi (paper), sehingga saldo
	// dry-run tetap terlihat di dashboard.
	client, err := s.client(ctx, account)
	if err != nil {
		state.LastError = truncate(errText(err), model.RejectReasonMaxLen)
		s.persistState(ctx, state)
		return
	}

	snapshot, err := client.Account(ctx)
	if err != nil {
		state.LastError = truncate(errText(err), model.RejectReasonMaxLen)
		s.persistState(ctx, state)
		return
	}

	state.Connected = true
	state.Currency = snapshot.Currency
	state.Balance = snapshot.Balance
	state.Equity = snapshot.Equity
	state.FreeMargin = snapshot.FreeMargin
	s.persistState(ctx, state)
}

func (s *Service) persistState(ctx context.Context, state *model.BrokerAccountState) {
	if err := s.store.UpsertAccountState(ctx, state); err != nil {
		s.logger.Error("gagal menyimpan telemetry akun",
			"account_id", state.BrokerAccountID, "error", err.Error())
	}
	s.pub.PublishTelemetry(ctx, state.BrokerAccountID)
}
