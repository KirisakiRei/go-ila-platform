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

	EngineGRPCAddr   string
	EngineMTLSCert   string
	EngineMTLSKey    string
	EngineMTLSCA     string
	RunnerSubscribe  string
	DailyTimezone    string
	MaxDailyTrades   int
	StopAfterWin     bool
	TelegramBotToken string
	TelegramTimeout  time.Duration
}

// Load membaca env lalu memvalidasi nilai wajib.
// Proses tidak boleh mulai dengan konfigurasi setengah jadi.
func Load(service string) (*Config, error) {
	cfg := &Config{
		AppEnv:           env("APP_ENV", "development"),
		ServiceName:      service,
		HTTPPort:         env("HTTP_PORT", "8080"),
		PublicURL:        env("PUBLIC_BASE_URL", "http://localhost:3000"),
		DBDSN:            env("DB_DSN", ""),
		DBMaxIdleConns:   envInt("DB_MAX_IDLE_CONNS", 5),
		DBMaxOpenConns:   envInt("DB_MAX_OPEN_CONNS", 20),
		DBConnMaxLife:    envDuration("DB_CONN_MAX_LIFETIME", time.Hour),
		DBAutoMigrate:    envBool("DB_AUTO_MIGRATE", false),
		JWTSecret:        env("JWT_SECRET", ""),
		JWTExpiry:        envDuration("JWT_EXPIRY", 12*time.Hour),
		AllowedOrigins:   envList("ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		ShutdownTimeout:  envDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
		LogLevel:         env("LOG_LEVEL", "info"),
		EngineGRPCAddr:   env("ENGINE_GRPC_ADDR", ""),
		EngineMTLSCert:   env("ENGINE_MTLS_CERT", ""),
		EngineMTLSKey:    env("ENGINE_MTLS_KEY", ""),
		EngineMTLSCA:     env("ENGINE_MTLS_CA", ""),
		RunnerSubscribe:  env("RUNNER_SUBSCRIBE_SYMBOLS", ""),
		DailyTimezone:    env("DAILY_TZ", "Asia/Jakarta"),
		MaxDailyTrades:   envInt("MAX_DAILY_TRADES", 2),
		StopAfterWin:     envBool("STOP_AFTER_WIN", true),
		TelegramBotToken: env("TELEGRAM_BOT_TOKEN", ""),
		TelegramTimeout:  envDuration("TELEGRAM_TIMEOUT", 10*time.Second),
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
	return nil
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
