package service

import (
	"time"

	"github.com/KirisakiRei/go-ila-platform/backend/pkg/utils"
)

// TokenIssuer membuat dan memverifikasi token sesi.
// Dipisah dari HTTP agar service tidak bergantung pada Gin.
type TokenIssuer interface {
	Issue(userID, role string) (token string, expiresAt time.Time, err error)
	Verify(token string) (claims utils.TokenClaims, err error)
}
