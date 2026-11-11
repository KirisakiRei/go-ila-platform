package router

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ila/backend/internal/authstate"
	"ila/backend/internal/authz"
	"ila/backend/internal/config"
	"ila/backend/internal/ctraderapi"
	"ila/backend/internal/handler"
	"ila/backend/internal/market"
	"ila/backend/internal/middleware"
	"ila/backend/internal/model"
	"ila/backend/internal/payment"
	"ila/backend/internal/repository"
	"ila/backend/internal/runner"
	"ila/backend/internal/runnerclient"
	"ila/backend/internal/service"
	"ila/backend/internal/session"
	"ila/backend/internal/settings"
	"ila/backend/internal/telegram"
	"ila/backend/pkg/cryptox"
)

// adminPrefix adalah awalan tunggal seluruh endpoint administrasi.
const adminPrefix = "/api/v1/admin"

// Billing returns the billing service so a background sweeper can run without
// duplicating the wiring in cmd.
func Billing(cfg *config.Config, db *gorm.DB, logger *slog.Logger) service.BillingService {
	userRepo := repository.NewUserRepository(db)
	billingRepo := repository.NewBillingRepository(db)
	gateway, err := payment.NewFromConfig(cfg)
	if err != nil {
		logger.Error("gateway pembayaran tidak dikenal; pembayaran daring dimatikan", "error", err.Error())
		gateway = payment.NewManual()
	}
	pricing := service.NewPricingService(repository.NewPricingRepository(db))
	adminAudit := service.NewAuditRecorder(repository.NewAuditRepository(db))
	return service.NewBillingService(userRepo, billingRepo, pricing, billingRepo, gateway, adminAudit, repository.NewCreditCodeRepository(db))
}

// SystemOps mengembalikan layanan operasional untuk dipakai proses latar
// seperti watchdog. Disediakan agar cmd tidak perlu merangkai ulang.
func SystemOps(cfg *config.Config, db *gorm.DB, logger *slog.Logger) service.AdminSystemOps {
	flags := settings.NewService(repository.NewSettingsRepository(db), func(key string) string {
		return envFlag(cfg, key)
	})
	return service.NewSystemOpsService(repository.NewSystemRepository(db), repository.NewRunnerRepository(db), flags, service.NewAuditRecorder(repository.NewAuditRepository(db)))
}

// OpsiBackendAPI menyesuaikan perakitan endpoint tanpa mengubah pemanggil lama.
type OpsiBackendAPI func(*konfigurasiBackendAPI)

type konfigurasiBackendAPI struct {
	strategi *runner.Settings
	tgClient *telegram.Client
}

// DenganStrategi memakai pengaturan strategi yang sudah dimuat pemanggil.
// Pemanggil yang menyegarkan pengaturan secara berkala (mis. dari katalog
// paket) harus memakai opsi ini agar panel user melihat tahap yang sama.
func DenganStrategi(s *runner.Settings) OpsiBackendAPI {
	return func(k *konfigurasiBackendAPI) { k.strategi = s }
}

// DenganTelegramClient memakai klien telegram yang sudah disiapkan pemanggil.
func DenganTelegramClient(c *telegram.Client) OpsiBackendAPI {
	return func(k *konfigurasiBackendAPI) { k.tgClient = c }
}

// NewBackendAPI merangkai seluruh endpoint backend-api.
// Wiring sengaja dikumpulkan di satu tempat agar alur dependensi terlihat utuh.
func NewBackendAPI(cfg *config.Config, db *gorm.DB, logger *slog.Logger, opsi ...OpsiBackendAPI) *gin.Engine {
	pilihan := konfigurasiBackendAPI{}
	for _, f := range opsi {
		f(&pilihan)
	}

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
	legalRepo := repository.NewLegalRepository(db)
	twoFactorRepo := repository.NewTwoFactorRepository(db)

	// Klien Runner: backend-api meminta eksekusi, bukan mengeksekusi sendiri.
	runnerAPI := runnerclient.New(cfg.RunnerBaseURL, cfg.InternalSecret)

	// Kewenangan dibaca dari DB dengan cache pendek (plans/04 §4.3).
	permCache := authz.NewService(repository.NewAuthzLoader(roleRepo))

	// Services
	sessionRepo := repository.NewSessionRepository(db)
	// Kunci untuk secret 2FA; sama dengan master key kredensial broker.
	sealer, sealerErr := cryptox.NewSealer(cfg.MasterKeyVersion, cfg.MasterKey)
	if sealerErr != nil {
		logger.Error("master key tidak sah; endpoint 2FA dinonaktifkan", "error", sealerErr.Error())
	}

	// Legal & 2FA dipakai lintas layanan, jadi dirangkai lebih dulu.
	legalService := service.NewLegalService(legalRepo, auditRepo)
	var twoFactorSvc service.TwoFactorService
	if sealer != nil {
		twoFactorSvc = service.NewTwoFactorService(twoFactorRepo, sealer, userRepo)
	}

	authService := service.WithTwoFactor(
		service.WithLegal(
			service.WithSessionRecorder(
				service.NewAuthService(userRepo, auditRepo, newTokenIssuer(cfg)),
				service.NewSessionRecorder(sessionRepo),
			),
			legalService,
		),
		twoFactorSvc,
		twoFactorRepo,
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

	// Katalog pair: sumber kebenaran simbol yang boleh dipilih user.
	pairRepo := repository.NewPairRepository(db)
	pairCatalogService := service.NewPairCatalogService(pairRepo)
	accountPairStrategyRepo := repository.NewAccountPairStrategyRepository(db)
	accountPairTimeframeRepo := repository.NewAccountPairTimeframeRepository(db)

	// Katalog paket adalah sumber kebenaran tahap strategi. Nilai env hanya
	// cadangan bila katalog belum punya barisnya.
	strategiSettings := pilihan.strategi
	if strategiSettings == nil {
		strategiSettings = runner.LoadSettings()
		strategiSettings.Reload(context.Background(), repository.NewTierFeatureReader(db))
	}

	accountService := service.NewAccountService(service.AccountServiceDeps{
		Users:         userRepo,
		Accounts:      accountRepo,
		Credentials:   credentialRepo,
		Subscriptions: subscriptionRepo,
		States:        stateRepo,
		Audit:         auditRepo,
		Sealer:        keyring,
		Strategies:    strategiSettings,
		Revoker:       runnerAPI,
		Legal:         legalService,
		// Batas akun & simbol dibaca dari katalog paket, bukan dari kode.
		Features: repository.NewTierFeatureReader(db),
		// Kunci sidik jari sederajat rahasia server: dipakai mengenali token
		// yang dipakai dua akun sekaligus.
		FingerprintSecret: cfg.InternalSecret,
		CTrader:           ctraderapi.NewClient(cfg.CTraderClientID, cfg.CTraderClientSecret, cfg.CTraderRedirectURI),
		// Katalog pair dikelola admin; user hanya memilih dari daftar itu.
		Catalog: pairCatalogService,
	})
	// Saklar metode butuh daftar akun dan hak tier, sedangkan AccountService butuh
	// saklar untuk menyiapkan bawaan saat pair baru dipantau. Siklus ini diputus
	// dengan pemasangan setter setelah keduanya jadi.
	strategySwitchService := service.NewStrategySwitchService(accountRepo, accountPairStrategyRepo, pairCatalogService, accountService)
	service.SetStrategySwitchSeeder(accountService, strategySwitchService)
	service.WithSignupSubscriber(authService, accountService)

	// Saklar timeframe: pasangan datang dari data sinyal nyata, sedangkan hak
	// auto-eksekusi dibaca dari katalog paket (auto_execute), sumber yang sama
	// dengan pagar eksekusi di Runner.
	timeframeSwitchService := service.NewTimeframeSwitchService(
		accountRepo,
		accountPairStrategyRepo,
		accountPairTimeframeRepo,
		signalRepo,
		accountService,
		service.TierResolver(userRepo),
		repository.NewTierFeatureReader(db),
	)
	service.SetAccountStrategyLister(accountService, accountPairStrategyRepo)
	tradingService := service.NewTradingService(userRepo, signalRepo, positionRepo, runnerRepo)
	systemService := service.NewSystemService(service.SystemServiceDeps{
		Accounts:  accountRepo,
		States:    stateRepo,
		Engines:   runnerRepo,
		Daily:     runnerRepo,
		Positions: positionRepo,
		Legal:     legalService,
		MaxTrades: cfg.MaxDailyTrades,
	})
	telegramService := service.NewTelegramService(userRepo, onboardingRepo, auditRepo)
	trialService := service.NewTrialService(userRepo, auditRepo)

	// anonymizer dipakai penghapusan akun pengelola: baris pengguna dianonimkan
	// agar invoice dan dokumen legal yang menunjuknya tetap utuh.
	roleService := service.NewRoleService(roleRepo, userRepo, auditRepo, permCache, repository.NewAdminUserRepository(db))

	// Manajemen pelanggan: seluruh aksinya lewat perekam audit standar.
	adminAudit := service.NewAuditRecorder(auditRepo)
	adminUserService := service.NewAdminUserService(
		repository.NewAdminUserRepository(db),
		service.AdminUserExtras{
			Accounts:  repository.NewAdminAccountRepository(db),
			Positions: repository.NewAdminPositionRepository(db),
			Legal:     repository.NewAdminLegalRepository(db),
			Notes:     repository.NewAdminNoteRepository(db),
		},
		adminAudit,
		userRepo,
	)
	auditAdminService := service.NewAdminAuditService(repository.NewAuditReader(db), adminAudit)

	pricingRepo := repository.NewPricingRepository(db)
	pricingService := service.NewPricingService(pricingRepo)
	adminReportService := service.NewAdminReportService(repository.NewReportRepository(db), adminAudit)
	adminSignalService := service.NewAdminSignalService(repository.NewAdminSignalRepository(db), adminAudit)

	// Flag operasional: nilai database menimpa env, dengan cache 10 detik agar
	// rem darurat berlaku cepat tanpa membebani database (plans/11 I2).
	settingsRepo := repository.NewSettingsRepository(db)
	settingsService := settings.NewService(settingsRepo, func(key string) string {
		return envFlag(cfg, key)
	})

	systemRepo := repository.NewSystemRepository(db)
	systemOpsService := service.NewSystemOpsService(systemRepo, runnerRepo, settingsService, adminAudit)
	adminSystemService := service.NewAdminSystemService(settingsService, settingsRepo, adminAudit)

	// Konten: FAQ dan pengumuman diubah tanpa deploy (plans/11 Fase J).
	cmsService := service.NewCMSService(repository.NewCMSRepository(db))

	// Halaman publik: isi bagian, FAQ, dan benefit paket dikelola admin.
	pageService := service.NewPageService(repository.NewPageRepository(db))

	// Alur akun: lupa sandi, verifikasi email, onboarding (plans/17).
	emailRepo := repository.NewEmailRepository(db)
	emailAccountStore := &emailAccountStoreAdapter{
		UserRepository:   userRepo,
		EmailRepository:  emailRepo,
		accountRepo:      accountRepo,
		subscriptionRepo: subscriptionRepo,
	}
	emailAccountService := service.NewEmailAccountService(emailAccountStore, auditRepo, cfg.PublicURL)

	adminAuthService := service.NewAdminAuthService(service.AdminAuthDeps{
		Users:      userRepo,
		Audit:      auditRepo,
		Roles:      roleRepo,
		Perms:      permCache,
		Issuer:     newAdminTokenIssuer(cfg),
		Sessions:   service.NewSessionRecorder(sessionRepo),
		TwoFactor:  twoFactorSvc,
		Challenges: twoFactorRepo,
	})

	// Handlers
	cookies := session.NewConfig(
		cfg.SessionCookieName,
		cfg.CSRFCookieName,
		cfg.CookieSecure,
		cfg.CookieDomain,
	)
	adminCookies := cookies.WithSessionName(cfg.AdminCookieName).WithCSRFName(cfg.AdminCSRFCookieName)

	sessionHandler := handler.NewSessionHandler(sessionService, middleware.CurrentSessionID)
	authHandler := handler.NewAuthHandler(
		authService,
		middleware.NewLoginLimiter(),
		middleware.NewRegisterLimiter(),
		cookies,
		cfg.JWTExpiry,
	)
	approveHandler := handler.NewApproveHandler(approveService)
	healthHandler := handler.NewHealthHandler(cfg.ServiceName, nil)
	accountHandler := handler.NewAccountHandler(accountService)
	pairHandler := handler.NewPairHandler(pairCatalogService, accountService)
	riskHandler := handler.NewRiskHandler(service.NewRiskSimulator(accountService, pairCatalogService, nil))
	strategySwitchHandler := handler.NewStrategySwitchHandler(strategySwitchService)
	timeframeSwitchHandler := handler.NewTimeframeSwitchHandler(timeframeSwitchService)
	adminPairHandler := handler.NewAdminPairHandler(service.NewPairAdminService(pairRepo, auditRepo))
	tradingHandler := handler.NewTradingHandler(tradingService)

	// Data candle untuk chart dashboard. Cache pendek disimpan di proses ini
	// agar satu halaman yang terus diperbarui tidak menembak feed untuk setiap
	// permintaan.
	marketService := market.NewService(market.NewTradingViewProvider(), 10*time.Second)
	marketHandler := handler.NewMarketHandler(marketService, pairRepo)
	systemHandler := handler.NewSystemHandler(systemService)
	tgClient := pilihan.tgClient
	if tgClient == nil {
		tgClient = telegram.NewClient(telegram.Config{
			BotToken: cfg.TelegramBotToken,
			Timeout:  cfg.TelegramTimeout,
		}, logger)
	}
	telegramHandler := handler.NewTelegramHandler(telegramService, cfg.TelegramBotUsername, cfg.TelegramWebhookSecret, tgClient)
	trialHandler := handler.NewTrialHandler(trialService)
	adminAuthHandler := handler.NewAdminAuthHandler(
		adminAuthService,
		middleware.NewLoginLimiter(),
		adminCookies,
		cfg.AdminCSRFCookieName,
		cfg.AdminJWTExpiry,
		cfg.AdminIdleTimeout,
	)
	adminRoleHandler := handler.NewAdminRoleHandler(roleService)
	adminUserHandler := handler.NewAdminUserHandler(adminUserService)
	auditAdminHandler := handler.NewAdminAuditHandler(auditAdminService)
	adminReportHandler := handler.NewAdminReportHandler(adminReportService)
	adminSignalHandler := handler.NewAdminSignalHandler(adminSignalService)
	adminSystemHandler := handler.NewAdminSystemHandler(adminSystemService, systemOpsService, systemOpsService)
	publicStatusHandler := handler.NewPublicStatusHandler(systemOpsService)
	// Ringkasan arah sinyal untuk halaman depan. Membaca jadwal pasar dari
	// layanan status yang sama, sehingga buka/tutup pasar dihitung di satu
	// tempat dan tidak pernah berbeda antar halaman.
	publicSignalHandler := handler.NewPublicSignalHandler(
		service.NewPublicSignalService(repository.NewPublicSignalStore(db), systemOpsService),
	)
	cmsHandler := handler.NewCMSHandler(cmsService)
	pageHandler := handler.NewPageHandler(pageService)
	emailAccountHandler := handler.NewEmailAccountHandler(emailAccountService)
	pricingHandler := handler.NewPricingHandler(pricingService)

	// Gateway pembayaran: kunci tidak pernah ditampilkan ke siapa pun.
	// Penyedia dipilih lewat PAYMENT_PROVIDER agar penggantian gateway tidak
	// menyentuh service billing (plans/08 §6).
	gateway, gatewayErr := payment.NewFromConfig(cfg)
	if gatewayErr != nil {
		logger.Error("gateway pembayaran tidak dikenal; pembayaran daring dimatikan", "error", gatewayErr.Error())
		gateway = payment.NewManual()
	}
	logger.Info("gateway pembayaran", "provider", gateway.Name())
	billingRepo := repository.NewBillingStore(db)
	billingService := service.NewBillingService(userRepo, billingRepo, pricingService, billingRepo, gateway, adminAudit, billingRepo)
	adminBillingService := service.NewAdminBillingService(billingRepo, adminAudit, billingService, gateway)
	adminTrialService := service.NewAdminTrialService(userRepo, adminAudit)
	adminTrialHandler := handler.NewAdminTrialHandler(adminTrialService)
	adminBillingHandler := handler.NewAdminBillingHandler(adminBillingService)
	billingHandler := handler.NewBillingHandler(billingService)
	legalHandler := handler.NewLegalHandler(legalService)

	// Alamat surel dipakai sebagai label di aplikasi authenticator.
	accountLabel := func(c *gin.Context, userID string) string {
		user, err := userRepo.FindByID(c.Request.Context(), userID)
		if err != nil {
			return ""
		}
		return user.Email
	}
	// Tanpa kunci master, 2FA dinonaktifkan: lebih baik tidak tersedia
	// daripada menyimpan secret tanpa enkripsi.
	var twoFactorHandler *handler.TwoFactorHandler
	if twoFactorSvc != nil {
		twoFactorHandler = handler.NewTwoFactorHandler(twoFactorSvc, accountLabel)
	} else {
		logger.Warn("2FA nonaktif: MASTER_KEY tidak sah atau kosong")
	}

	// Health
	r.GET("/health", healthHandler.Live)
	r.GET("/ready", healthHandler.Ready)

	// Webhook Telegram: diautentikasi rahasia header, bukan sesi user.
	r.POST("/api/v1/telegram/webhook", telegramHandler.Webhook)

	// Dokumen legal publik: boleh dibaca tanpa masuk, karena calon pengguna
	// perlu membacanya sebelum memutuskan mendaftar.
	publicLegal := r.Group("/api/v1/legal")
	publicLegal.Use(middleware.RateLimitByIP(middleware.NewRateLimiter(0, 0)))
	{
		publicLegal.GET("", legalHandler.List)
		publicLegal.GET("/:slug", legalHandler.BySlug)
	}

	// Webhook pembayaran: diautentikasi tanda tangan gateway, bukan sesi.
	// Batas per IP hanya menahan banjir permintaan, bukan pengganti tanda tangan.
	webhook := r.Group("/api/v1/webhook")
	webhook.Use(middleware.RateLimitByIP(middleware.NewRateLimiter(0, 0)))
	{
		webhook.POST("/payment", billingHandler.Webhook)
	}

	// Daftar paket publik: dibaca halaman harga sebelum masuk.
	publicBilling := r.Group("/api/v1/billing")
	publicBilling.Use(middleware.RateLimitByIP(middleware.NewRateLimiter(0, 0)))
	{
		publicBilling.GET("/plans", pricingHandler.Plans)
	}

	// Status publik: dipakai halaman masuk dan dashboard untuk menampilkan
	// mode pemeliharaan tanpa memaksa login.
	publicStatus := r.Group("/api/v1")
	publicStatus.Use(middleware.RateLimitByIP(middleware.NewRateLimiter(0, 0)))
	{
		publicStatus.GET("/status", publicStatusHandler.Status)
		// Halaman depan memakai ringkasan arah sinyal tanpa login. Letaknya di
		// grup yang dibatasi per alamat IP, sama seperti status layanan.
		publicStatus.GET("/signals/preview", publicSignalHandler.Preview)
		cmsHandler.PublicRegister(publicStatus.Group(""))
		pageHandler.PublicRegister(publicStatus.Group(""))
	}

	// API v1
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/verify-2fa", authHandler.VerifyTwoFactor)
		// Logout hanya menghapus cookie; aman tanpa sesi.
		auth.POST("/logout", authHandler.Logout)
		auth.GET("/csrf", authHandler.CSRF)
		// Lupa sandi & verifikasi email: dibatasi per IP karena tanpa sesi.
		emailAccountHandler.RegisterPublic(auth, middleware.RateLimitByIP(middleware.NewRateLimiter(0, 0)))
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
		// Tampilan peringatan diam; batas sesi di server tidak berubah.
		me.PATCH("/me/idle-warning", authHandler.SetIdleWarning)

		// Perangkat yang sedang masuk & pencabutan per sesi.
		me.GET("/me/sessions", sessionHandler.List)
		me.DELETE("/me/sessions/:id", sessionHandler.Revoke)
		// Konfirmasi entry: token dari deep-link Telegram, wajib sesi login.
		me.GET("/approve/:token", approveHandler.Preview)
		me.POST("/approve/:token/confirm", approveHandler.Confirm)
		me.POST("/approve/:token/decline", approveHandler.Decline)
		// Token untuk tombol eksekusi di dashboard. Rute statis ini didaftarkan
		// sesudah rute token agar tidak tertukar dengan /approve/:token.
		me.POST("/approve/signals/:signalID", approveHandler.Issue)

		// Akun broker: kredensial masuk terenkripsi, tidak pernah keluar lagi.
		me.GET("/accounts", accountHandler.List)
		me.POST("/accounts", accountHandler.Create)
		me.PATCH("/accounts/:id", accountHandler.Update)
		me.DELETE("/accounts/:id", accountHandler.Delete)
		me.GET("/accounts/ctrader/auth-url", accountHandler.CTraderAuthURL)
		me.POST("/accounts/ctrader/callback", accountHandler.CTraderCallback)

		// Langganan simbol dan pilihan strategi.
		me.GET("/subscriptions", accountHandler.ListSubscriptions)
		me.POST("/subscriptions", accountHandler.Subscribe)
		me.DELETE("/subscriptions/:id", accountHandler.Unsubscribe)
		me.GET("/strategies", accountHandler.Strategies)

		// Katalog pair & checklist per akun. Simbol hanya dipilih dari katalog
		// yang dikelola admin, tidak pernah diketik bebas oleh user.
		pairHandler.Register(me)

		// Simulasi risiko: satu tempat hitungan lot untuk pratinjau dashboard.
		riskHandler.Register(me)

		// Saklar metode per akun × pair. Pair pantauan global; metode dipilih
		// per akun lewat saklar ini.
		strategySwitchHandler.Register(me)

		// Saklar pasangan timeframe per akun × pair.
		timeframeSwitchHandler.Register(me)

		// Penghubungan Telegram.
		me.POST("/telegram/link", telegramHandler.CreateLink)
		me.DELETE("/telegram/link", telegramHandler.Disconnect)

		// Uji coba 7 hari: setara Plus namun hanya simulasi.
		me.GET("/trial", trialHandler.Status)
		me.POST("/trial", trialHandler.Start)

		// Billing: checkout dan riwayat tagihan pengguna.
		billingHandler.RegisterUser(me)

		// Status pasar, engine, broker, dan kuota harian.
		me.GET("/system/status", systemHandler.Status)
		emailAccountHandler.RegisterUser(me)

		// Riwayat untuk dashboard.
		me.GET("/signals", tradingHandler.Signals)
		me.GET("/positions", tradingHandler.Positions)
		me.GET("/positions/export", tradingHandler.ExportPositions)

		// Candle untuk chart radar sinyal dan peninjau posisi. Feed mengikuti
		// pair yang dipilih admin agar selaras dengan evaluasi engine.
		me.GET("/market/candles", marketHandler.Candles)

		// Persetujuan dokumen legal. `blocked` pada balasan berarti eksekusi
		// dan pembayaran ditahan sampai dokumen wajib disetujui.
		me.GET("/me/legal", legalHandler.Me)
		me.POST("/me/legal/:slug/accept", legalHandler.Accept)

		// 2FA opsional untuk user: melindungi akun bila kata sandi bocor.
		if twoFactorHandler != nil {
			me.GET("/2fa", twoFactorHandler.Status)
			me.POST("/2fa/setup", twoFactorHandler.Setup)
			me.POST("/2fa/activate", twoFactorHandler.Activate)
			me.POST("/2fa/disable", twoFactorHandler.Disable)
			me.POST("/2fa/recovery-codes", twoFactorHandler.Regenerate)
		}
	}

	registerAdmin(r, cfg, logger, userRepo, permCache, adminAuthHandler, adminRoleHandler, twoFactorHandler, legalHandler, adminUserHandler, auditAdminHandler, adminBillingHandler, adminReportHandler, adminSystemHandler, cmsHandler, pageHandler, adminTrialHandler, adminPairHandler, adminSignalHandler)

	// Tidak ada endpoint admin tanpa izin: bila ada yang lupa dideklarasikan,
	// proses berhenti sebelum menerima permintaan (plans/04 §5.2).
	if missing := middleware.UndeclaredAdminRoutes(r.Routes()); len(missing) > 0 {
		panic(fmt.Sprintf("route admin tanpa permission: %v", missing))
	}

	return r
}

// envFlag menjadi nilai bawaan flag ketika belum ada override di database.
// Satu sumber untuk backend-api dan proses latar, sehingga angka yang tampil
// di panel sama dengan yang dipakai Runner (plans/11 I2).
func envFlag(cfg *config.Config, key string) string {
	switch key {
	case settings.KeyAutoExecute:
		return strconv.FormatBool(cfg.AutoExecute)
	case settings.KeyDryRunAll:
		return strconv.FormatBool(cfg.DryRunAll)
	case settings.KeyMaxDailyTrades:
		return strconv.Itoa(cfg.MaxDailyTrades)
	case settings.KeyMaxDailyLosses:
		return strconv.Itoa(cfg.MaxDailyLosses)
	case settings.KeyStopAfterWin:
		return strconv.FormatBool(cfg.StopAfterWin)
	case settings.KeyMarketOpenOnly:
		return "true"
	case settings.KeyLegalGate:
		return "true"
	}
	return ""
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
	twoFactorHandler *handler.TwoFactorHandler,
	legalHandler *handler.LegalHandler,
	userHandler *handler.AdminUserHandler,
	auditHandler *handler.AdminAuditHandler,
	billingHandler *handler.AdminBillingHandler,
	reportHandler *handler.AdminReportHandler,
	systemHandler *handler.AdminSystemHandler,
	cmsHandler *handler.CMSHandler,
	pageHandler *handler.PageHandler,
	adminTrialHandler *handler.AdminTrialHandler,
	adminPairHandler *handler.AdminPairHandler,
	adminSignalHandler *handler.AdminSignalHandler,
) {
	issuer := newAdminTokenIssuer(cfg)

	// Autentikasi: tidak membawa sesi admin yang sudah ada. Batas per IP
	// menahan tebakan password dari satu sumber.
	login := r.Group(adminPrefix + "/auth")
	login.Use(middleware.RateLimitByIP(middleware.NewRateLimiter(0, 0)))
	{
		login.POST("/login", authHandler.Login)
		login.POST("/verify-2fa", authHandler.VerifyTwoFactor)
		login.POST("/logout", authHandler.Logout)
		login.GET("/csrf", authHandler.CSRF)
	}

	admin := r.Group(adminPrefix)
	admin.Use(middleware.AdminScope(issuer, cfg.AdminCookieName, reader))
	admin.Use(middleware.RateLimitByUser(middleware.NewRateLimiter(0, 0)))
	admin.Use(middleware.RequireCSRF(cfg.AdminCSRFCookieName))

	// Profil sesi sendiri: cukup lolos pintu admin, tanpa permission.
	// Ditandai "-" karena hanya butuh sesi admin, bukan permission tertentu.
	middleware.RegisterAdminRoute(http.MethodGet, adminPrefix+"/auth/me", "-")
	admin.GET("/auth/me", authHandler.Profile)

	// Heartbeat memperpanjang cookie selama sesi masih sah. Dipanggil panel
	// admin berkala saat pengguna benar-benar berinteraksi (plans/23 §3).
	middleware.RegisterAdminRoute(http.MethodPost, adminPrefix+"/auth/heartbeat", "-")
	admin.POST("/auth/heartbeat", authHandler.Heartbeat)

	// guard mencatat izin sebuah route lalu mengembalikan penjaganya. Path
	// lengkap dibentuk dari awalan grup agar deklarasi tidak bisa meleset.
	guard := func(method, path, permission string) gin.HandlerFunc {
		middleware.RegisterAdminRoute(method, adminPrefix+path, permission)
		return middleware.RequirePermission(permCache, permission)
	}

	// Tata kelola: lihat role dan katalog permission.
	//
	// Handler wajib disertakan SETELAH guard. Tanpa itu, guard menjadi handler
	// terakhir dan Gin membalas 200 dengan body kosong — panel menerjemahkannya
	// sebagai "Balasan server tidak dapat dibaca".
	admin.GET("/roles", guard(http.MethodGet, "/roles", authz.RoleView), roleHandler.List)
	admin.GET("/roles/:id", guard(http.MethodGet, "/roles/:id", authz.RoleView), roleHandler.Detail)
	admin.GET("/permissions", guard(http.MethodGet, "/permissions", authz.RoleView), roleHandler.Permissions)

	// 2FA admin: opsional sebagai tambahan proteksi di atas kata sandi.
	// Cukup sesi admin — mengatur faktor milik sendiri bukan kewenangan orang
	// lain, jadi tidak butuh permission.
	if twoFactorHandler != nil {
		middleware.RegisterAdminRoute(http.MethodGet, adminPrefix+"/2fa", "-")
		middleware.RegisterAdminRoute(http.MethodPost, adminPrefix+"/2fa/setup", "-")
		middleware.RegisterAdminRoute(http.MethodPost, adminPrefix+"/2fa/activate", "-")
		middleware.RegisterAdminRoute(http.MethodPost, adminPrefix+"/2fa/disable", "-")
		middleware.RegisterAdminRoute(http.MethodPost, adminPrefix+"/2fa/recovery-codes", "-")

		admin.GET("/2fa", twoFactorHandler.Status)
		admin.POST("/2fa/setup", twoFactorHandler.Setup)
		admin.POST("/2fa/activate", twoFactorHandler.Activate)
		admin.POST("/2fa/disable", twoFactorHandler.Disable)
		admin.POST("/2fa/recovery-codes", twoFactorHandler.Regenerate)
	}

	// Konten: dokumen legal berversi. Dokumen terbit tidak pernah diubah;
	// perubahan berarti versi baru (plans/09 §2.1).
	admin.GET("/legal", guard(http.MethodGet, "/legal", authz.CMSView), legalHandler.AdminList)
	admin.POST("/legal", guard(http.MethodPost, "/legal", authz.CMSEdit), legalHandler.AdminCreate)
	admin.PATCH("/legal/:id", guard(http.MethodPatch, "/legal/:id", authz.CMSEdit), legalHandler.AdminUpdate)
	admin.POST("/legal/:id/publish", guard(http.MethodPost, "/legal/:id/publish", authz.CMSPublish), legalHandler.AdminPublish)

	// Tata kelola: ubah role dan penugasannya. Buat dan hapus dipisah dari
	// RoleEdit agar pemberi izin bisa membatasi dampak tanpa mencabut
	// kemampuan menyunting permission.
	admin.POST("/roles", guard(http.MethodPost, "/roles", authz.RoleCreate), roleHandler.Create)
	admin.PATCH("/roles/:id", guard(http.MethodPatch, "/roles/:id", authz.RoleEdit), roleHandler.Update)
	admin.DELETE("/roles/:id", guard(http.MethodDelete, "/roles/:id", authz.RoleDelete), roleHandler.Delete)
	admin.PATCH("/users/:id/role", guard(http.MethodPatch, "/users/:id/role", authz.RoleEdit), roleHandler.Assign)
	// Akun pengelola baru. Dijaga role.edit karena pembuatan akun pengelola pada
	// dasarnya adalah penugasan role; service tetap menolak role yang memuat
	// kewenangan di luar milik pelaku.
	admin.POST("/admins", guard(http.MethodPost, "/admins", authz.RoleEdit), roleHandler.CreateAdmin)
	// Penghapusan akun pengelola dijaga role.delete karena dampaknya setara
	// menghapus kewenangan: akun tidak bisa lagi masuk dan barisnya
	// dianonimkan. Akun superadmin utama berpenanda is_protected tetap
	// ditolak service. Larangan hapus akun sendiri juga dipegang service.
	admin.DELETE("/admins/:id", guard(http.MethodDelete, "/admins/:id", authz.RoleDelete), roleHandler.DeleteAdmin)

	// Pelanggan: daftar, profil, dan aksi berlapis.
	// Kredensial broker tidak punya endpoint di sini, dan itu disengaja.
	userHandler.Register(admin.Group(""), guard)
	auditHandler.Register(admin.Group(""), guard)
	billingHandler.Register(admin.Group(""), guard)
	adminTrialHandler.Register(admin.Group(""), guard)
	reportHandler.Register(admin.Group(""), guard)
	systemHandler.Register(admin.Group(""), guard)
	cmsHandler.Register(admin.Group(""), guard)
	adminPairHandler.Register(admin.Group(""), guard)

	// Sinyal lintas pengguna: baca tingkat lanjut untuk operasional.
	adminSignalHandler.Register(admin.Group(""), guard)
	pageHandler.Register(admin.Group(""), guard)

	logger.Info("pintu admin siap", "base", "/api/v1/admin")
}

// emailAccountStoreAdapter menggabungkan repository yang dibutuhkan alur akun
// tanpa bentrok nama method FindByID antara UserRepository dan BrokerAccountRepository.
type emailAccountStoreAdapter struct {
	repository.UserRepository
	*repository.EmailRepository
	accountRepo      repository.BrokerAccountRepository
	subscriptionRepo repository.SubscriptionRepository
}

func (a *emailAccountStoreAdapter) ListByUser(ctx context.Context, userID string) ([]model.BrokerAccount, error) {
	return a.accountRepo.ListByUser(ctx, userID)
}

func (a *emailAccountStoreAdapter) ListActiveByUser(ctx context.Context, userID string) ([]model.Subscription, error) {
	return a.subscriptionRepo.ListActiveByUser(ctx, userID)
}
