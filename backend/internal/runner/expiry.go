package runner

import (
	"context"
	"ila/backend/internal/evalprice"
	"ila/backend/internal/model"
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
	s.nilaiSinyalBerakhir(ctx, hangus)
	s.kabariPenungguHarga(ctx, hangus)
}

// nilaiSinyalBerakhir menyimpan hasil TP/SL untuk setup yang berakhir tanpa
// dieksekusi.
//
// Dijalankan saat kedaluwarsa karena candle masih segar. Tanpa ini, riwayat
// hanya bisa menilai saat halaman dibuka, dan setelah beberapa hari feed tidak
// lagi memuat waktunya sehingga hasil setup hilang selamanya.
func (s *Service) nilaiSinyalBerakhir(ctx context.Context, signalIDs []string) {
	if s.opts.CandleFetcher == nil || s.opts.EvalSink == nil {
		return
	}
	for _, signalID := range signalIDs {
		signal, err := s.store.FindSignal(ctx, signalID)
		if err != nil || signal == nil || signal.EntryPrice <= 0 {
			continue
		}
		s.nilaiSatuSinyal(ctx, signal, signalID)
	}
}

// nilaiSatuSinyal menilai satu sinyal dan menyimpannya untuk setiap user yang
// memegang baris sinyal itu.
func (s *Service) nilaiSatuSinyal(ctx context.Context, signal *model.Signal, signalID string) {
	candles, err := s.opts.CandleFetcher(ctx, signal.Symbol, signal.FeedSource, signal.Ltf, EvalCandleLimit)
	if err != nil || len(candles) == 0 {
		s.logger.Warn("penilaian sinyal dilewati: candle tidak terbaca",
			"evt", "signal.eval.gagal", "signal_id", signalID, "error", errString(err))
		return
	}
	in := evalprice.InputDariSinyal(signal)
	var out evalprice.Outcome
	if evalprice.CandleTerlaluTua(in, candles) {
		out = evalprice.TanpaCandle()
	} else {
		var ok bool
		out, ok = evalprice.FromCandles(in, candles)
		if !ok {
			// Candle belum menutup jendela setup; percobaan berikutnya menyusul.
			return
		}
	}
	// Hanya hasil akhir yang dikunci di sini. RUNNING berarti harga baru masuk
	// dan belum keluar; menyimpannya akan membuat baris terlihat final padahal
	// jurnal masih harus menilai ulang. Membiarkannya kosong membuat jurnal
	// yang melanjutkan penilaian.
	if out.Status == model.EvalRunning {
		return
	}
	ev := &model.UserSignalEval{
		Status:  out.Status,
		EntryAt: out.EntryAt,
		At:      out.At,
		Price:   out.Price,
		RLower:  out.RLower,
		Note:    out.Note,
	}
	users, err := s.store.UsersOfSignal(ctx, signalID)
	if err != nil {
		s.logger.Warn("gagal membaca pemegang sinyal untuk penilaian",
			"signal_id", signalID, "error", err.Error())
		return
	}
	for _, userID := range users {
		if err := s.opts.EvalSink(ctx, userID, signalID, ev); err != nil {
			s.logger.Warn("gagal menyimpan hasil penilaian",
				"signal_id", signalID, "user_id", userID, "error", err.Error())
		}
	}
	s.logger.Info("sinyal dinilai saat berakhir",
		"evt", "signal.eval", "signal_id", signalID, "status", out.Status, "user", len(users))
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

// EvalCandleLimit sama dengan batas penilai jurnal: cukup untuk menutup setup
// harian tanpa memuat riwayat panjang pada setiap penilaian.
const EvalCandleLimit = 500

// errString mengubah galat menjadi teks pendek untuk log; nil menjadi "".
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
