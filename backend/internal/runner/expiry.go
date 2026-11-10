package runner

import "context"

// ExpireStaleSignals menutup sinyal yang sudah melewati masa berlakunya.
//
// Engine hanya mengirim status terminal lewat SignalUpdate, sehingga sinyal
// READY_TO_ENTRY yang tidak pernah dieksekusi akan tertinggal selamanya dan
// membuat radar tampak macet. Penyapu ini hanya mengubah status baris sinyal;
// posisi, eksekusi, dan kuota tidak disentuh.
func (s *Service) ExpireStaleSignals(ctx context.Context) {
	jumlah, err := s.store.ExpireStaleSignals(ctx, s.opts.Clock())
	if err != nil {
		s.logger.Error("gagal menyapu sinyal kedaluwarsa",
			"evt", "signal.expire.gagal", "error", err.Error())
		return
	}
	if jumlah > 0 {
		s.logger.Info("sinyal kedaluwarsa ditutup",
			"evt", "signal.expire", "jumlah", jumlah)
	}
}
