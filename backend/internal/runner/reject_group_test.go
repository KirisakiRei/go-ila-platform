package runner_test

import (
	"testing"

	"ila/backend/internal/runner"
)

// TestRejectGroupLengkap memastikan setiap alasan penolakan punya grup.
// Alasan yang lupa dipetakan akan jatuh ke GroupBatal diam-diam dan membuat
// user diberi tahu setup berakhir padahal masih berjalan.
func TestRejectGroupLengkap(t *testing.T) {
	kasus := map[runner.RejectReason]runner.RejectGroup{
		runner.RejectNone:             runner.GroupNone,
		runner.RejectExpired:          runner.GroupBatal,
		runner.RejectWaitingNext:      runner.GroupBatal,
		runner.RejectNotReady:         runner.GroupBatal,
		runner.RejectTerminal:         runner.GroupBatal,
		runner.RejectBadSignal:        runner.GroupBatal,
		runner.RejectSlippage:         runner.GroupHarga,
		runner.RejectDailyQuota:       runner.GroupKuota,
		runner.RejectDailyWin:         runner.GroupKuota,
		runner.RejectTwoStrike:        runner.GroupKuota,
		runner.RejectOpenPosition:     runner.GroupPosisi,
		runner.RejectLegalPending:     runner.GroupAkun,
		runner.RejectAccountFrozen:    runner.GroupAkun,
		runner.RejectSuspended:        runner.GroupAkun,
		runner.RejectAccountBanned:    runner.GroupAkun,
		runner.RejectDuplicate:        runner.GroupAkun,
		runner.RejectEntitlement:      runner.GroupTerkunci,
		runner.RejectStrategyTier:     runner.GroupTerkunci,
		runner.RejectStrategyDisabled: runner.GroupTerkunci,
		runner.RejectMarketClosed:     runner.GroupPasar,
		runner.RejectMaintenance:      runner.GroupSistem,
		runner.RejectReadOnly:         runner.GroupSistem,
	}

	for reason, ingin := range kasus {
		if got := runner.RejectGroupOf(reason); got != ingin {
			t.Errorf("RejectGroupOf(%q) = %q, ingin %q", reason, got, ingin)
		}
	}
}

// TestRejectTerkunciTidakDikirim memastikan penahanan karena tier atau metode
// tidak pernah menjadi pesan Telegram.
func TestRejectTerkunciTidakDikirim(t *testing.T) {
	for _, reason := range []runner.RejectReason{
		runner.RejectEntitlement,
		runner.RejectStrategyTier,
		runner.RejectStrategyDisabled,
	} {
		if runner.RejectTerkirimKeTelegram(reason) {
			t.Errorf("%q seharusnya tidak dikirim ke Telegram", reason)
		}
	}

	for _, reason := range []runner.RejectReason{
		runner.RejectDailyQuota,
		runner.RejectSlippage,
		runner.RejectMarketClosed,
	} {
		if !runner.RejectTerkirimKeTelegram(reason) {
			t.Errorf("%q seharusnya tetap dikirim ke Telegram", reason)
		}
	}
}
