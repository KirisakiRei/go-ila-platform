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

	// Sesi disimpan di cookie HttpOnly agar tidak terjangkau JavaScript.
	SessionCookieName string
	CSRFCookieName    string
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

	// MasterKey membungkus kunci per user; wajib untuk kredensial broker.
	MasterKey        string
	MasterKeyVersion int
	// MasterKeysLegacy menjaga data lama tetap terbaca setelah rotasi.
	// Format: "1:<base64>,2:<base64>" — hanya kunci lama, bukan kunci aktif.
	MasterKeysLegacy string

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
}

// Load membaca env lalu memvalidasi nilai wajib.
// Proses tidak boleh mulai dengan konfigurasi setengah jadi.
func Load(service string) (*Config, error) {
	cfg := &Config{
		AppEnv:                env("APP_ENV", "development"),
		ServiceName:           service,
		HTTPPort:              env("HTTP_PORT", "8080"),
		PublicURL:             env("PUBLIC_BASE_URL", "http://localhost:3000"),
		DBDSN:                 env("DB_DSN", ""),
		DBMaxIdleConns:        envInt("DB_MAX_IDLE_CONNS", 5),
		DBMaxOpenConns:        envInt("DB_MAX_OPEN_CONNS", 20),
		DBConnMaxLife:         envDuration("DB_CONN_MAX_LIFETIME", time.Hour),
		DBAutoMigrate:         envBool("DB_AUTO_MIGRATE", false),
		JWTSecret:             env("JWT_SECRET", ""),
		JWTExpiry:             envDuration("JWT_EXPIRY", 12*time.Hour),
		AllowedOrigins:        envList("ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		ShutdownTimeout:       envDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
		LogLevel:              env("LOG_LEVEL", "info"),
		SessionCookieName:     env("SESSION_COOKIE_NAME", "ila_session"),
		CSRFCookieName:        env("CSRF_COOKIE_NAME", "ila_csrf"),
		CookieSecure:          envBool("COOKIE_SECURE", false),
		CookieDomain:          env("COOKIE_DOMAIN", ""),
		EngineGRPCAddr:        env("ENGINE_GRPC_ADDR", ""),
		EngineMTLSCert:        env("ENGINE_MTLS_CERT", ""),
		EngineMTLSKey:         env("ENGINE_MTLS_KEY", ""),
		EngineMTLSCA:          env("ENGINE_MTLS_CA", ""),
		EngineServerName:      env("ENGINE_SERVER_NAME", ""),
		EngineStrategy:        env("ENGINE_STRATEGY", "ila-2.0"),
		Engine3GRPCAddr:       env("ENGINE3_GRPC_ADDR", ""),
		Engine3MTLSCert:       env("ENGINE3_MTLS_CERT", ""),
		Engine3MTLSKey:        env("ENGINE3_MTLS_KEY", ""),
		Engine3MTLSCA:         env("ENGINE3_MTLS_CA", ""),
		Engine3ServerName:     env("ENGINE3_SERVER_NAME", ""),
		ILA3Availability:      env("ILA3_AVAILABILITY", "off"),
		RunnerSubscriber:      env("RUNNER_SUBSCRIBER_ID", "runner-1"),
		DailyTimezone:         env("DAILY_TZ", "Asia/Jakarta"),
		MaxDailyTrades:        envInt("MAX_DAILY_TRADES", 2),
		MaxDailyLosses:        envInt("MAX_DAILY_LOSSES", 2),
		StopAfterWin:          envBool("STOP_AFTER_WIN", true),
		AutoExecute:           envBool("AUTO_EXECUTE", false),
		DryRunAll:             envBool("DRY_RUN_ALL", false),
		MasterKey:             env("MASTER_KEY", ""),
		MasterKeyVersion:      envInt("MASTER_KEY_VERSION", 1),
		MasterKeysLegacy:      env("MASTER_KEYS_LEGACY", ""),
		InternalSecret:        env("INTERNAL_SECRET", ""),
		RunnerBaseURL:         env("RUNNER_BASE_URL", "http://runner:8081"),
		TelegramBotToken:      env("TELEGRAM_BOT_TOKEN", ""),
		TelegramTimeout:       envDuration("TELEGRAM_TIMEOUT", 10*time.Second),
		TelegramBotUsername:   env("TELEGRAM_BOT_USERNAME", ""),
		TelegramWebhookSecret: env("TELEGRAM_WEBHOOK_SECRET", ""),
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
