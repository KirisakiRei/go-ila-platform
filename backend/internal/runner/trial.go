package runner

import (
	"time"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/model"
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
func (s *Service) mustDryRun(session *userSession, account *model.BrokerAccount) bool {
	if account.IsDryRun || s.opts.DryRunAll {
		return true
	}
	return InTrial(session.User, s.opts.Clock())
}
