package runner

import (
	"context"
	"errors"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// ErrAmbiguousAccount menandakan ada lebih dari satu akun yang cocok.
var ErrAmbiguousAccount = errors.New("akun tidak unik")

// ProcessSignalForAccount menjalankan alur keputusan untuk satu akun atas satu
// sinyal yang SUDAH tersimpan.
//
// Hanya untuk keperluan pengujian: jalur produksi selalu masuk lewat
// HandleSignal agar sinyal tercatat lebih dulu. Diekspor supaya tes bisa
// menguji status mentah (mis. WAIT_*) yang tidak bisa dibentuk lewat payload
// protobuf.
func (s *Service) ProcessSignalForAccount(ctx context.Context, accountID string, signal *model.Signal) error {
	accounts, err := s.store.ListActiveBrokerAccounts(ctx)
	if err != nil {
		return err
	}
	var found *model.BrokerAccount
	for i := range accounts {
		if accounts[i].ID != accountID {
			continue
		}
		if found != nil {
			return ErrAmbiguousAccount
		}
		found = &accounts[i]
	}
	if found == nil {
		return apperror.ErrNotFound
	}
	user, err := s.store.FindUserByID(ctx, found.UserID)
	if err != nil {
		return err
	}
	return s.processSignalForAccount(ctx, user, found, signal)
}
