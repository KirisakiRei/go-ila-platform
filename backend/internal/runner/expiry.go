package runner

import (
	"context"
)

// ExpireStaleSignals menutup sinyal yang sudah melewati masa berlakunya.
//
// Engine hanya mengirim status terminal lewat SignalUpdate, sehingga sinyal
// READY_TO_ENTRY yang tidak pernah dieksekusi akan tertinggal selamanya dan
// membuat radar tampak macet. Penyapu ini hanya mengubah status baris sinyal;
// posisi, eksekusi, dan kuota tidak disentuh.
//
// User yang sempat diberi tahu bahwa setup menunggu harga kembali ikut dikabari
// saat jendelanya habis. Tanpa kabar kedua, mereka hanya melihat pesan
// "menunggu" tanpa pernah tahu setupnya berakhir.
func (s *Service) ExpireStaleSignals(ctx context.Context) {
	hangus, err := s.store.ExpireStaleSignals(ctx, s.opts.Clock())
	if err != nil {
		s.logger.Error("gagal menyapu sinyal kedaluwarsa",
			"evt", "signal.expire.gagal", "error", err.Error())
		return
	}
	if len(hangus) == 0 {
		return
	}

	s.logger.Info("sinyal kedaluwarsa ditutup",
		"evt", "signal.expire", "jumlah", len(hangus))
	s.kabariPenungguHarga(ctx, hangus)
}

// kabariPenungguHarga memberi tahu user yang setupnya dibatalkan karena harga
// tidak kembali ke level entry sebelum batas waktu.
//
// Hanya user yang tertahan karena slippage yang dikirimi; user lain sudah
// menerima pesan batalnya sendiri saat sinyal dibatalkan.
func (s *Service) kabariPenungguHarga(ctx context.Context, signalIDs []string) {
	penunggu, err := s.store.SlippageWaitingUsers(ctx, signalIDs)
	if err != nil {
		s.logger.Error("gagal membaca user penunggu harga",
			"evt", "signal.expire.penunggu.gagal", "error", err.Error())
		return
	}

	for signalID, users := range penunggu {
		for _, userID := range users {
			s.recordUserSignal(ctx, userID, signalID, SignalExpired,
				"Harga tidak kembali ke level entry sebelum batas waktu.", nil)
			s.pub.PublishSignal(ctx, userID, signalID, SignalRejected,
				string(RejectExpired))
		}
	}
}

// EvalCandleLimit bounds each grouped forward-test feed request.
const EvalCandleLimit = 500
