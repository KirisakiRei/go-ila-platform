package router

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ila/backend/internal/authstate"
	"ila/backend/internal/authz"
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

// adminPrefix adalah awalan tunggal seluruh endpoint administrasi.
const adminPrefix = "/api/v1/admin"

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
	roleRepo := repository.NewRoleRepository(db)

	// Klien Runner: backend-api meminta eksekusi, bukan mengeksekusi sendiri.
	runnerAPI := runnerclient.New(cfg.RunnerBaseURL, cfg.InternalSecret)

	// Kewenangan dibaca dari DB dengan cache pendek (plans/04 §4.3).
	permCache := authz.NewService(repository.NewAuthzLoader(roleRepo))

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

	roleService := service.NewRoleService(roleRepo, userRepo, auditRepo, permCache)
	adminAuthService := service.NewAdminAuthService(service.AdminAuthDeps{
		Users:    userRepo,
		Audit:    auditRepo,
		Roles:    roleRepo,
		Perms:    permCache,
		Issuer:   newAdminTokenIssuer(cfg),
		Sessions: service.NewSessionRecorder(sessionRepo),
	})

	// Handlers
	cookies := session.NewConfig(
		cfg.SessionCookieName,
		cfg.CSRFCookieName,
		cfg.CookieSecure,
		cfg.CookieDomain,
	)
	adminCookies := cookies.WithSessionName(cfg.AdminCookieName)

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
	adminAuthHandler := handler.NewAdminAuthHandler(
		adminAuthService,
		middleware.NewLoginLimiter(),
		adminCookies,
		cfg.CSRFCookieName,
		cfg.AdminJWTExpiry,
	)
	adminRoleHandler := handler.NewAdminRoleHandler(roleService)

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

	registerAdmin(r, cfg, logger, userRepo, permCache, adminAuthHandler, adminRoleHandler)

	// Tidak ada endpoint admin tanpa izin: bila ada yang lupa dideklarasikan,
	// proses berhenti sebelum menerima permintaan (plans/04 §5.2).
	if missing := middleware.UndeclaredAdminRoutes(r.Routes()); len(missing) > 0 {
		panic(fmt.Sprintf("route admin tanpa permission: %v", missing))
	}

	return r
}

// registerAdmin memasang pintu admin: login, sesi, lalu route ber-permission.
//
// Setiap route admin mendeklarasikan izinnya tepat di barisnya sehingga tidak
// ada endpoint admin tanpa kewenangan karena lupa (plans/04 §5.2).
func registerAdmin(
	r *gin.Engine,
	cfg *config.Config,
	logger *slog.Logger,
	reader authstate.Reader,
	permCache *authz.Service,
	authHandler *handler.AdminAuthHandler,
	roleHandler *handler.AdminRoleHandler,
) {
	issuer := newAdminTokenIssuer(cfg)

	// Autentikasi: tidak membawa sesi admin yang sudah ada. Batas per IP
	// menahan tebakan password dari satu sumber.
	login := r.Group(adminPrefix + "/auth")
	login.Use(middleware.RateLimitByIP(middleware.NewRateLimiter(0, 0)))
	{
		login.POST("/login", authHandler.Login)
		login.POST("/logout", authHandler.Logout)
		login.GET("/csrf", authHandler.CSRF)
	}

	admin := r.Group(adminPrefix)
	admin.Use(middleware.AdminScope(issuer, cfg.AdminCookieName, reader))
	admin.Use(middleware.RateLimitByUser(middleware.NewRateLimiter(0, 0)))
	admin.Use(middleware.RequireCSRF(cfg.CSRFCookieName))

	// Profil sesi sendiri: cukup lolos pintu admin, tanpa permission.
	// Ditandai "-" karena hanya butuh sesi admin, bukan permission tertentu.
	middleware.RegisterAdminRoute(http.MethodGet, adminPrefix+"/auth/me", "-")
	admin.GET("/auth/me", authHandler.Profile)

	// guard mencatat izin sebuah route lalu mengembalikan penjaganya. Path
	// lengkap dibentuk dari awalan grup agar deklarasi tidak bisa meleset.
	guard := func(method, path, permission string) gin.HandlerFunc {
		middleware.RegisterAdminRoute(method, adminPrefix+path, permission)
		return middleware.RequirePermission(permCache, permission)
	}

	// Tata kelola: lihat role dan katalog permission.
	admin.GET("/roles", guard(http.MethodGet, "/roles", authz.RoleView))
	admin.GET("/roles/:id", guard(http.MethodGet, "/roles/:id", authz.RoleView))
	admin.GET("/permissions", guard(http.MethodGet, "/permissions", authz.RoleView))

	// Tata kelola: ubah role dan penugasannya.
	admin.POST("/roles", guard(http.MethodPost, "/roles", authz.RoleEdit))
	admin.PATCH("/roles/:id", guard(http.MethodPatch, "/roles/:id", authz.RoleEdit))
	admin.DELETE("/roles/:id", guard(http.MethodDelete, "/roles/:id", authz.RoleEdit))
	admin.PATCH("/users/:id/role", guard(http.MethodPatch, "/users/:id/role", authz.RoleEdit))

	logger.Info("pintu admin siap", "base", "/api/v1/admin")
}
