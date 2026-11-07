package router

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ila/backend/internal/config"
	"ila/backend/pkg/utils"
)

// testConfig menyusun konfigurasi minimal yang lolos validasi.
func testConfig(t *testing.T) *config.Config {
	t.Helper()

	return &config.Config{
		AppEnv:            "development",
		ServiceName:       "uji",
		HTTPPort:          "8080",
		PublicURL:         "http://localhost:3000",
		JWTSecret:         "rahasia-panjang-untuk-uji-1234567890",
		JWTExpiry:         time.Hour,
		AdminJWTExpiry:    2 * time.Hour,
		AllowedOrigins:    []string{"http://localhost:3000"},
		ShutdownTimeout:   time.Second,
		LogLevel:          "error",
		SessionCookieName: "ila_session",
		CSRFCookieName:    "ila_csrf",
		AdminCookieName:   "ila_admin_session",
		CookieSecure:      false,
		DailyTimezone:     "Asia/Jakarta",
		MaxDailyTrades:    2,
		EngineStrategy:    "ila-2.0",
		RunnerBaseURL:     "http://127.0.0.1:8081",
		// Master key lemah: endpoint akun dinonaktifkan, tetapi route tetap terpasang.
		MasterKey:        "",
		MasterKeyVersion: 1,
		InternalSecret:   "rahasia-internal-uji",
		MaxDailyLosses:   2,
	}
}

// testLogger menulis log ke tempat sampah agar keluaran tes bersih.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// testDB membuat handle GORM tanpa membuka koneksi; cukup untuk merangkai
// repository yang belum dipanggil pada tes ini.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	// Driver tiruan: cukup untuk merangkai repository tanpa koneksi nyata.
	sqlDB, err := sql.Open("ila-stub", "")
	if err != nil {
		t.Fatalf("sql open error: %v", err)
	}

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger:               logger.Discard,
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("gorm open error: %v", err)
	}
	return db
}

// performRequest mengirim satu permintaan tanpa cookie.
func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// performRequestWithCookie mengirim permintaan dengan satu cookie.
func performRequestWithCookie(r http.Handler, method, path, name, value string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{Name: name, Value: value})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// issueTestToken membuat token sah dengan scope tertentu.
func issueTestToken(t *testing.T, cfg *config.Config, userID, role, scope string) string {
	t.Helper()

	issuer := utils.NewJWTIssuer(cfg.JWTSecret, time.Hour)
	token, _, err := issuer.IssueWithClaims(utils.Claims{UserID: userID, Role: role, Scope: scope, Session: 1})
	if err != nil {
		t.Fatalf("IssueWithClaims error: %v", err)
	}
	return token
}

// memastikan gin tidak berjalan dalam mode debug saat tes.
func init() {
	gin.SetMode(gin.TestMode)
	sql.Register("ila-stub", stubDriver{})
}

// stubDriver adalah driver database kosong: tes ini merangkai router, bukan
// menyentuh data, sehingga tidak perlu koneksi sungguhan.
type stubDriver struct{}

func (stubDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("tanpa koneksi database pada tes router")
}
