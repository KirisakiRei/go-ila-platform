package runner

import (
	"context"
	"time"

	"ila/backend/internal/model"
)

// userSession membungkus user agar aturan trial punya satu tempat.
type userSession struct {
	User *model.User
}

// InTrial menandai user yang sedang menjalani uji coba 7 hari.
func InTrial(user *model.User, now time.Time) bool {
	if user == nil || user.TrialExpiresAt == nil {
		return false
	}
	return now.Before(*user.TrialExpiresAt)
}

// mustDryRun menentukan apakah eksekusi wajib disimulasikan.
//
// Trial sengaja tidak boleh menyentuh uang user (plans/14 §5.5.2): uji coba
// hanya menilai kualitas sinyal, bukan keberanian menaruh modal.
func (s *Service) mustDryRun(ctx context.Context, session *userSession, account *model.BrokerAccount) bool {
	if account.IsDryRun || s.dryRunForced(ctx) {
		return true
	}
	return InTrial(session.User, s.opts.Clock())
}

// MustDryRunForTest membuka pagar simulasi untuk pengujian.
// Dipakai tes paket luar agar aturan uang diuji langsung, bukan lewat alur
// yang lebih dulu tertahan aturan lain.
var MustDryRunForTest = func(s *Service, user *model.User, account *model.BrokerAccount) bool {
	return s.mustDryRun(context.Background(), &userSession{User: user}, account)
}
