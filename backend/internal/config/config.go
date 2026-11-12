package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Tier adalah tingkat langganan yang berlaku bagi akun.
type Tier string

const (
	TierFree  Tier = "free"
	TierBasic Tier = "basic"
	TierPlus  Tier = "plus"
)

// Role adalah peran akun di dalam platform.
// Admin sengaja tidak punya wewenang trading; lihat plans/12.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
	RoleStaff Role = "support"
)

// Config menampung seluruh konfigurasi proses dari environment.
type Config struct {
	AppEnv      string
	ServiceName string
	HTTPPort    string
	PublicURL   string

	DBDSN           string
	DBMaxIdleConns  int
	DBMaxOpenConns  int
	DBConnMaxLife   time.Duration
	DBAutoMigrate   bool
	JWTSecret       string
	JWTExpiry       time.Duration
	AllowedOrigins  []string
	ShutdownTimeout time.Duration
	LogLevel        string
	// UploadDir adalah akar penyimpanan lampiran. Berada di volume yang sama
	// dengan log agar tidak perlu volume baru; isinya tidak pernah disajikan
	// langsung sebagai berkas statis.
	UploadDir string

	// Sesi disimpan di cookie HttpOnly agar tidak terjangkau JavaScript.
	SessionCookieName string
	CSRFCookieName    string
	// AdminCookieName memisahkan sesi admin dari sesi user (plans/03 §2).
	AdminCookieName string
	// AdminCSRFCookieName terpisah agar pintu admin dan user tidak saling
	// menimpa nilai CSRF.
	AdminCSRFCookieName string
	// AdminJWTExpiry lebih pendek dari sesi user karena kewenangannya lebih besar.
	AdminJWTExpiry time.Duration
	// AdminIdleTimeout adalah batas diam yang memicu modal "masih di sini?"
	// di panel admin (plans/23 §2).
	AdminIdleTimeout time.Duration
	// CookieSecure wajib true di produksi (HTTPS). Boleh false saat pengembangan lokal.
	CookieSecure bool
	// CookieDomain kosong berarti host-only cookie.
	CookieDomain string

	EngineGRPCAddr   string
	EngineMTLSCert   string
	EngineMTLSKey    string
	EngineMTLSCA     string
	EngineServerName string
	EngineStrategy   string

	// ILA 3.0 berjalan sebagai proses terpisah dengan kontrak gRPC yang sama.
	// Alamat kosong berarti engine itu tidak dijalankan (plans/15 §1.1).
	Engine3GRPCAddr   string
	Engine3MTLSCert   string
	Engine3MTLSKey    string
	Engine3MTLSCA     string
	Engine3ServerName string
	ILA3Availability  string
	RunnerSubscriber  string
	DailyTimezone     string
	MaxDailyTrades    int
	MaxDailyLosses    int
	StopAfterWin      bool
	AutoExecute       bool
	DryRunAll         bool
	// TimeframeSwitch mengaktifkan penyaringan pasangan timeframe per akun.
	// Bawaannya NYALA mengikuti katalog pengaturan; bisa dimatikan lewat env
	// untuk kembalikan perilaku lama tanpa deploy.
	TimeframeSwitch bool
	// PairSwitch mengaktifkan saklar metode per akun pada pair.
	PairSwitch bool
	// InviteOnly menutup pendaftaran mandiri, uji coba, dan pembayaran.
	// Akun baru hanya dibuat admin selama flag ini menyala.
	InviteOnly bool

	// MasterKey membungkus kunci per user; wajib untuk kredensial broker.
	MasterKey        string
	MasterKeyVersion int
	// MasterKeysLegacy menjaga data lama tetap terbaca setelah rotasi.
	// Format: "1:<base64>,2:<base64>" — hanya kunci lama, bukan kunci aktif.
	MasterKeysLegacy string

	// Gateway pembayaran. Kunci tidak pernah dikirim ke klien; hanya dipakai
	// server untuk membuat tagihan dan memeriksa tanda tangan webhook.
	// PaymentProvider memilih adapter gateway pembayaran (pakasir/manual).
	PaymentProvider   string
	PakasirBaseURL    string
	PakasirSlug       string
	PakasirAPIKey     string
	PakasirWebhookKey string

	// InternalSecret melindungi endpoint internal Runner dari pemanggil liar.
	InternalSecret string
	// RunnerBaseURL adalah alamat HTTP internal Runner.
	RunnerBaseURL string

	TelegramBotToken string
	TelegramTimeout  time.Duration
	// TelegramBotUsername dipakai membentuk tautan t.me untuk onboarding.
	TelegramBotUsername string
	// TelegramWebhookSecret memverifikasi bahwa webhook benar dari Telegram.
	TelegramWebhookSecret string
	// TelegramAdminChatID menerima peringatan operasional admin (plans/18 §2).
	TelegramAdminChatID string

	// SMTP dipakai worker antrean surel. Kosong berarti surel dinonaktifkan
	// dan pesan tetap menumpuk di antrean, bukan hilang (plans/17 §2.2).
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	// cTrader Open API credentials (OAuth & Socket)
	CTraderClientID     string
	CTraderClientSecret string
	CTraderRedirectURI  string
}

// Load membaca env lalu memvalidasi nilai wajib.
// Proses tidak boleh mulai dengan konfigurasi setengah jadi.
func Load(service string) (*Config, error) {
	cfg := &Config{
		AppEnv:              env("APP_ENV", "development"),
		ServiceName:         service,
		HTTPPort:            env("HTTP_PORT", "8080"),
		PublicURL:           env("PUBLIC_BASE_URL", "http://localhost:3000"),
		DBDSN:               env("DB_DSN", ""),
		DBMaxIdleConns:      envInt("DB_MAX_IDLE_CONNS", 5),
		DBMaxOpenConns:      envInt("DB_MAX_OPEN_CONNS", 20),
		DBConnMaxLife:       envDuration("DB_CONN_MAX_LIFETIME", time.Hour),
		DBAutoMigrate:       envBool("DB_AUTO_MIGRATE", false),
		JWTSecret:           env("JWT_SECRET", ""),
		JWTExpiry:           envDuration("JWT_EXPIRY", 12*time.Hour),
		AllowedOrigins:      envList("ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		ShutdownTimeout:     envDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
		LogLevel:            env("LOG_LEVEL", "info"),
		UploadDir:           env("UPLOAD_DIR", "/var/log/ila/lampiran"),
		SessionCookieName:   env("SESSION_COOKIE_NAME", "ila_session"),
		CSRFCookieName:      env("CSRF_COOKIE_NAME", "ila_csrf"),
		AdminCookieName:     env("ADMIN_COOKIE_NAME", "ila_admin_session"),
		AdminCSRFCookieName: env("ADMIN_CSRF_COOKIE_NAME", "ila_admin_csrf"),
		AdminJWTExpiry:      envDuration("ADMIN_JWT_EXPIRY", 2*time.Hour),
		AdminIdleTimeout:    envDuration("ADMIN_IDLE_TIMEOUT", 10*time.Minute),
		CookieSecure:        envBool("COOKIE_SECURE", false),
		CookieDomain:        env("COOKIE_DOMAIN", ""),
		EngineGRPCAddr:      env("ENGINE_GRPC_ADDR", ""),
		EngineMTLSCert:      env("ENGINE_MTLS_CERT", ""),
		EngineMTLSKey:       env("ENGINE_MTLS_KEY", ""),
		EngineMTLSCA:        env("ENGINE_MTLS_CA", ""),
		EngineServerName:    env("ENGINE_SERVER_NAME", ""),
		EngineStrategy:      env("ENGINE_STRATEGY", "ila-2.0"),
		Engine3GRPCAddr:     env("ENGINE3_GRPC_ADDR", ""),
		Engine3MTLSCert:     env("ENGINE3_MTLS_CERT", ""),
		Engine3MTLSKey:      env("ENGINE3_MTLS_KEY", ""),
		Engine3MTLSCA:       env("ENGINE3_MTLS_CA", ""),
		Engine3ServerName:   env("ENGINE3_SERVER_NAME", ""),
		ILA3Availability:    env("ILA3_AVAILABILITY", "off"),
		RunnerSubscriber:    env("RUNNER_SUBSCRIBER_ID", "runner-1"),
		DailyTimezone:       env("DAILY_TZ", "Asia/Jakarta"),
		MaxDailyTrades:      envInt("MAX_DAILY_TRADES", 2),
		MaxDailyLosses:      envInt("MAX_DAILY_LOSSES", 2),
		StopAfterWin:        envBool("STOP_AFTER_WIN", true),
		// Default true mengikuti katalog settings, supaya perilaku tanpa env
		// sama dengan perilaku setelah flag diubah lewat admin.
		TimeframeSwitch: envBool("TIMEFRAME_SWITCH_ENABLED", true),
		PairSwitch:      envBool("PAIR_SWITCH_ENABLED", true),
		// Gate akses bawaan NYALA (fail-closed): platform sedang invite only.
		InviteOnly:            envBool("INVITE_ONLY", true),
		AutoExecute:           envBool("AUTO_EXECUTE", false),
		DryRunAll:             envBool("DRY_RUN_ALL", false),
		MasterKey:             env("MASTER_KEY", ""),
		MasterKeyVersion:      envInt("MASTER_KEY_VERSION", 1),
		MasterKeysLegacy:      env("MASTER_KEYS_LEGACY", ""),
		PaymentProvider:       env("PAYMENT_PROVIDER", "pakasir"),
		PakasirBaseURL:        env("PAKASIR_BASE_URL", "https://pakasir.com"),
		PakasirSlug:           env("PAKASIR_SLUG", ""),
		PakasirAPIKey:         env("PAKASIR_API_KEY", ""),
		PakasirWebhookKey:     env("PAKASIR_WEBHOOK_KEY", ""),
		InternalSecret:        env("INTERNAL_SECRET", ""),
		RunnerBaseURL:         env("RUNNER_BASE_URL", "http://runner:8081"),
		TelegramBotToken:      env("TELEGRAM_BOT_TOKEN", ""),
		TelegramTimeout:       envDuration("TELEGRAM_TIMEOUT", 10*time.Second),
		TelegramBotUsername:   env("TELEGRAM_BOT_USERNAME", ""),
		TelegramWebhookSecret: env("TELEGRAM_WEBHOOK_SECRET", ""),
		TelegramAdminChatID:   env("TELEGRAM_ADMIN_CHAT_ID", ""),
		SMTPHost:              env("SMTP_HOST", ""),
		SMTPPort:              envInt("SMTP_PORT", 587),
		SMTPUser:              env("SMTP_USER", ""),
		SMTPPass:              env("SMTP_PASS", ""),
		SMTPFrom:              env("SMTP_FROM", ""),
		CTraderClientID:       env("CTRADER_CLIENT_ID", ""),
		CTraderClientSecret:   env("CTRADER_CLIENT_SECRET", ""),
		CTraderRedirectURI:    env("CTRADER_REDIRECT_URI", ""),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.DBDSN == "" {
		return fmt.Errorf("DB_DSN wajib diisi")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET wajib diisi")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET minimal 32 karakter (sekarang %d)", len(c.JWTSecret))
	}
	if c.HTTPPort == "" {
		return fmt.Errorf("HTTP_PORT wajib diisi")
	}
	// SMTP yang hanya terisi separuh lebih berbahaya daripada kosong: surel
	// akan gagal dikirim tanpa alasan yang jelas saat dibaca log.
	if (c.SMTPHost == "") != (c.SMTPFrom == "") {
		return fmt.Errorf("SMTP_HOST dan SMTP_FROM harus diisi bersama")
	}
	if c.SMTPPort < 1 || c.SMTPPort > 65535 {
		return fmt.Errorf("SMTP_PORT tidak sah (sekarang %d)", c.SMTPPort)
	}
	if c.MaxDailyTrades < 1 || c.MaxDailyTrades > 3 {
		return fmt.Errorf("MAX_DAILY_TRADES harus 1 sampai 3 (sekarang %d)", c.MaxDailyTrades)
	}
	// Cookie tanpa Secure di produksi bisa terkirim lewat HTTP biasa.
	if c.IsProduction() && !c.CookieSecure {
		return fmt.Errorf("COOKIE_SECURE wajib true saat APP_ENV=production")
	}
	if c.SessionCookieName == c.CSRFCookieName {
		return fmt.Errorf("SESSION_COOKIE_NAME dan CSRF_COOKIE_NAME tidak boleh sama")
	}
	if c.AdminCookieName == c.SessionCookieName {
		return fmt.Errorf("ADMIN_COOKIE_NAME harus berbeda dari SESSION_COOKIE_NAME")
	}
	if c.AdminCSRFCookieName == c.CSRFCookieName {
		return fmt.Errorf("ADMIN_CSRF_COOKIE_NAME harus berbeda dari CSRF_COOKIE_NAME")
	}
	// Default localhost di produksi membuat frontend sendiri ditolak CORS
	// tanpa pesan jelas; lebih baik gagal saat start (plans/21 D3).
	if c.IsProduction() {
		if len(c.AllowedOrigins) == 0 {
			return fmt.Errorf("ALLOWED_ORIGINS wajib diisi saat APP_ENV=production")
		}
		for _, origin := range c.AllowedOrigins {
			if origin == "*" {
				return fmt.Errorf("ALLOWED_ORIGINS tidak boleh \"*\": cookie sesi butuh asal eksplisit")
			}
			if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
				return fmt.Errorf("ALLOWED_ORIGINS memuat alamat lokal (%s) saat produksi", origin)
			}
		}
	}
	return nil
}

// LegacyMasterKeys memecah MASTER_KEYS_LEGACY menjadi pasangan versi dan kunci.
// Nilai rusak dilewati agar satu entri salah tidak mematikan seluruh proses.
func (c *Config) LegacyMasterKeys() map[int]string {
	if strings.TrimSpace(c.MasterKeysLegacy) == "" {
		return nil
	}

	keys := make(map[int]string)
	for _, part := range strings.Split(c.MasterKeysLegacy, ",") {
		version, key, ok := strings.Cut(strings.TrimSpace(part), ":")
		if !ok {
			continue
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(version))
		if err != nil {
			continue
		}
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			keys[parsed] = trimmed
		}
	}

	if len(keys) == 0 {
		return nil
	}
	return keys
}

// IsProduction menandai konfigurasi produksi.
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.AppEnv, "production")
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func envBool(key string, fallback bool) bool {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return v
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return v
}

func envList(key string, fallback []string) []string {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
