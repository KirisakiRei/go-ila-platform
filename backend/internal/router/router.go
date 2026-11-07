package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ila/backend/internal/config"
	"ila/backend/internal/handler"
	"ila/backend/internal/middleware"
	"ila/backend/internal/repository"
	"ila/backend/internal/runner"
	"ila/backend/internal/runnerclient"
	"ila/backend/internal/service"
	"ila/backend/internal/session"
	"ila/backend/pkg/cryptox"
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
	runnerRepo := repository.NewRunnerRepository(db)
	approvalRepo := repository.NewApprovalTokenRepository(db)
	accountRepo := repository.NewBrokerAccountRepository(db)
	credentialRepo := repository.NewCredentialRepository(db)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	signalRepo := repository.NewSignalRepository(db)
	positionRepo := repository.NewPositionRepository(db)
	stateRepo := repository.NewBrokerAccountStateRepository(db)
	onboardingRepo := repository.NewOnboardingTokenRepository(db)

	// Klien Runner: backend-api meminta eksekusi, bukan mengeksekusi sendiri.
	runnerAPI := runnerclient.New(cfg.RunnerBaseURL, cfg.InternalSecret)

	// Services
	sessionRepo := repository.NewSessionRepository(db)
	authService := service.WithSessionRecorder(
		service.NewAuthService(userRepo, auditRepo, newTokenIssuer(cfg)),
		service.NewSessionRecorder(sessionRepo),
	)
	sessionService := service.NewSessionService(sessionRepo, userRepo, auditRepo)
	approveService := service.NewApproveService(
		approvalRepo,
		runnerRepo,
		runnerRepo,
		auditRepo,
		runnerAPI,
	)

	// Kredensial broker dienkripsi sebelum menyentuh database.
	keyring, err := cryptox.NewKeyring(cfg.MasterKeyVersion, cfg.MasterKey, cfg.LegacyMasterKeys())
	if err != nil {
		logger.Error("master key tidak sah; endpoint akun dinonaktifkan", "error", err.Error())
	}

	accountService := service.NewAccountService(service.AccountServiceDeps{
		Users:         userRepo,
		Accounts:      accountRepo,
		Credentials:   credentialRepo,
		Subscriptions: subscriptionRepo,
		States:        stateRepo,
		Audit:         auditRepo,
		Sealer:        keyring,
		Strategies:    runner.LoadSettings(),
		Revoker:       runnerAPI,
	})
	tradingService := service.NewTradingService(userRepo, signalRepo, positionRepo, runnerRepo)
	systemService := service.NewSystemService(service.SystemServiceDeps{
		Accounts:  accountRepo,
		States:    stateRepo,
		Engines:   runnerRepo,
		Daily:     runnerRepo,
		Positions: positionRepo,
		MaxTrades: cfg.MaxDailyTrades,
	})
	telegramService := service.NewTelegramService(userRepo, onboardingRepo, auditRepo)
	trialService := service.NewTrialService(userRepo, auditRepo)

	// Handlers
	cookies := session.NewConfig(
		cfg.SessionCookieName,
		cfg.CSRFCookieName,
		cfg.CookieSecure,
		cfg.CookieDomain,
	)

	sessionHandler := handler.NewSessionHandler(sessionService, middleware.CurrentSessionID)
	authHandler := handler.NewAuthHandler(
		authService,
		middleware.NewLoginLimiter(),
		cookies,
		cfg.JWTExpiry,
	)
	approveHandler := handler.NewApproveHandler(approveService)
	healthHandler := handler.NewHealthHandler(cfg.ServiceName, nil)
	accountHandler := handler.NewAccountHandler(accountService)
	tradingHandler := handler.NewTradingHandler(tradingService)
	systemHandler := handler.NewSystemHandler(systemService)
	telegramHandler := handler.NewTelegramHandler(telegramService, cfg.TelegramBotUsername, cfg.TelegramWebhookSecret)
	trialHandler := handler.NewTrialHandler(trialService)

	// Health
	r.GET("/health", healthHandler.Live)
	r.GET("/ready", healthHandler.Ready)

	// Webhook Telegram: diautentikasi rahasia header, bukan sesi user.
	r.POST("/api/v1/telegram/webhook", telegramHandler.Webhook)

	// API v1
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		// Logout hanya menghapus cookie; aman tanpa sesi.
		auth.POST("/logout", authHandler.Logout)
		auth.GET("/csrf", authHandler.CSRF)
	}

	me := r.Group("/api/v1")
	me.Use(middleware.Auth(newTokenIssuer(cfg), cfg.SessionCookieName, userRepo))
	// Batas per akun mencegah satu klien yang rusak menghabiskan memori.
	me.Use(middleware.RateLimitByUser(middleware.NewRateLimiter(0, 0)))
	// Cookie HttpOnly menahan pencurian token lewat XSS, tetapi permintaan
	// lintas situs tetap bisa ikut membawa cookie; CSRF menutup celah itu.
	me.Use(middleware.RequireCSRF(cfg.CSRFCookieName))
	{
		me.GET("/me", authHandler.Profile)
		// Ganti password & cabut seluruh sesi (plans/23 §6).
		me.POST("/me/password", authHandler.ChangePassword)
		me.POST("/me/logout-all", authHandler.LogoutAll)

		// Perangkat yang sedang masuk & pencabutan per sesi.
		me.GET("/me/sessions", sessionHandler.List)
		me.DELETE("/me/sessions/:id", sessionHandler.Revoke)
		// Konfirmasi entry: token dari deep-link Telegram, wajib sesi login.
		me.GET("/approve/:token", approveHandler.Preview)
		me.POST("/approve/:token/confirm", approveHandler.Confirm)
		me.POST("/approve/:token/decline", approveHandler.Decline)

		// Akun broker: kredensial masuk terenkripsi, tidak pernah keluar lagi.
		me.GET("/accounts", accountHandler.List)
		me.POST("/accounts", accountHandler.Create)
		me.PATCH("/accounts/:id", accountHandler.Update)
		me.DELETE("/accounts/:id", accountHandler.Delete)

		// Langganan simbol dan pilihan strategi.
		me.GET("/subscriptions", accountHandler.ListSubscriptions)
		me.POST("/subscriptions", accountHandler.Subscribe)
		me.DELETE("/subscriptions/:id", accountHandler.Unsubscribe)
		me.GET("/strategies", accountHandler.Strategies)

		// Penghubungan Telegram.
		me.POST("/telegram/link", telegramHandler.CreateLink)
		me.DELETE("/telegram/link", telegramHandler.Disconnect)

		// Uji coba 7 hari: setara Plus namun hanya simulasi.
		me.GET("/trial", trialHandler.Status)
		me.POST("/trial", trialHandler.Start)

		// Status pasar, engine, broker, dan kuota harian.
		me.GET("/system/status", systemHandler.Status)

		// Riwayat untuk dashboard.
		me.GET("/signals", tradingHandler.Signals)
		me.GET("/positions", tradingHandler.Positions)
	}

	return r
}
