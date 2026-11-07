package router

import (
	"github.com/KirisakiRei/go-ila-platform/backend/internal/config"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/service"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/utils"
)

func newTokenIssuer(cfg *config.Config) service.TokenIssuer {
	return utils.NewJWTIssuer(cfg.JWTSecret, cfg.JWTExpiry)
}
