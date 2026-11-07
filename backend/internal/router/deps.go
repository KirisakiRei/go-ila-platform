package router

import (
	"ila/backend/internal/config"
	"ila/backend/internal/service"
	"ila/backend/pkg/utils"
)

// newTokenIssuer menerbitkan sesi user.
func newTokenIssuer(cfg *config.Config) service.TokenIssuer {
	return utils.NewJWTIssuer(cfg.JWTSecret, cfg.JWTExpiry)
}

// newAdminTokenIssuer menerbitkan sesi admin dengan masa berlaku lebih pendek
// karena kewenangannya lebih besar (plans/03 §2).
func newAdminTokenIssuer(cfg *config.Config) service.TokenIssuer {
	return utils.NewJWTIssuer(cfg.JWTSecret, cfg.AdminJWTExpiry)
}
