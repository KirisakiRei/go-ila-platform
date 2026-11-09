package runner

import (
	"context"
	"time"
)

// AccountLockTTL adalah umur maksimum kunci akun. Jauh lebih lama daripada satu
// siklus kirim order, sehingga eksekusi normal tidak pernah kehilangan kuncinya,
// tetapi tetap membebaskan akun bila proses berhenti di tengah jalan.
const AccountLockTTL = 5 * time.Minute

// lockAccount mengunci satu akun broker untuk satu sinyal.
//
// Pagar "satu posisi terbuka" pada evaluate hanya melihat posisi yang SUDAH
// tercatat. Dua sinyal berbeda untuk akun yang sama — misalnya dua pasangan
// timeframe pada pair yang sama — bisa melewati pemeriksaan itu bersamaan lalu
// mengeksekusi keduanya. Kunci ini menutup celah tersebut di tingkat basis data,
// bukan di memori yang bisa dilewati dua goroutine sekaligus.
//
// Nilai balik kedua adalah fungsi pelepas. Pemanggil WAJIB memanggilnya
// (biasanya lewat defer) agar akun tidak terkunci setelah order selesai.
// Bila kunci tidak didapat, pelepas nil dan penguncian gagal.
func (s *Service) lockAccount(ctx context.Context, accountID, signalID string, now time.Time) (func(), bool, error) {
	// Bersihkan kunci peninggalan proses yang berhenti sebelum sempat melepas.
	if err := s.store.ReleaseStaleAccountLock(ctx, now.Add(-AccountLockTTL)); err != nil {
		s.logger.Warn("gagal melepas kunci akun kedaluwarsa", "error", err.Error())
	}

	dikunci, err := s.store.ClaimAccountLock(ctx, accountID, signalID, now)
	if err != nil {
		return nil, false, err
	}
	if !dikunci {
		return nil, false, nil
	}

	release := func() {
		if err := s.store.ReleaseAccountLock(ctx, accountID); err != nil {
			s.logger.Warn("gagal melepas kunci akun",
				"account_id", accountID, "error", err.Error())
		}
	}
	return release, true, nil
}
