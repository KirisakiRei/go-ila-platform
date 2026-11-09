package service

import (
	"context"
	"time"

	"ila/backend/internal/model"
)

// userFinder adalah kebutuhan TierResolver terhadap penyimpanan pengguna.
type userFinder interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
}

// TierResolver mengembalikan tier efektif seorang pengguna.
//
// Tier efektif memperhitungkan masa trial dan masa berlaku paket, sehingga
// keputusan hak akses di panel tidak menyimpang dari yang dipakai Runner.
func TierResolver(users userFinder) func(ctx context.Context, userID string) (string, error) {
	return func(ctx context.Context, userID string) (string, error) {
		user, err := users.FindByID(ctx, userID)
		if err != nil {
			return "", err
		}
		return effectiveTier(user, time.Now().UTC()), nil
	}
}
