package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/config"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/handler"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/middleware"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/repository"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/service"
)

// NewBackendAPI merangkai seluruh endpoint backend-api.
// Wiring sengaja dikumpulkan di satu tempat agar alur dependensi terlihat utuh.
func NewBackendAPI(cfg *config.Config, db *gorm.DB, logger *slog.Logger) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.ErrorHandler(logger))

	// Repositories
	userRepo := repository.NewUserRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	// Services
	authService := service.NewAuthService(userRepo, auditRepo, newTokenIssuer(cfg))

	// Handlers
	authHandler := handler.NewAuthHandler(authService)
	healthHandler := handler.NewHealthHandler(cfg.ServiceName, nil)

	// Health
	r.GET("/health", healthHandler.Live)
	r.GET("/ready", healthHandler.Ready)

	// API v1
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	me := r.Group("/api/v1")
	me.Use(middleware.Auth(newTokenIssuer(cfg)))
	{
		me.GET("/me", authHandler.Profile)
	}

	return r
}
