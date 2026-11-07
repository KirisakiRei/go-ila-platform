package router

import (
	"ila/backend/internal/config"
	"ila/backend/internal/service"
	"ila/backend/pkg/utils"
)

func newTokenIssuer(cfg *config.Config) service.TokenIssuer {
	return utils.NewJWTIssuer(cfg.JWTSecret, cfg.JWTExpiry)
}
