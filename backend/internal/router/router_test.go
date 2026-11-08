package router

import (
	"net/http"
	"strings"
	"testing"

	"ila/backend/internal/middleware"
)

// Setiap route admin yang terpasang harus terdeklarasi, dan sebaliknya.
// Tes ini menangkap route admin baru yang lupa diberi permission.
func TestSemuaRouteAdminTerdeklarasi(t *testing.T) {
	cfg := testConfig(t)
	r := NewBackendAPI(cfg, testDB(t), testLogger())

	var adminCount int
	for _, info := range r.Routes() {
		if strings.HasPrefix(info.Path, "/api/v1/admin/") {
			adminCount++
		}
	}
	if adminCount == 0 {
		t.Fatal("tidak ada route admin yang terpasang")
	}

	if missing := middleware.UndeclaredAdminRoutes(r.Routes()); len(missing) > 0 {
		t.Fatalf("route admin tanpa permission: %v", missing)
	}
}

// Endpoint RBAC harus tersedia dengan penjagaan RBAC, bukan penjagaan user.
func TestRouteRbacTerpasang(t *testing.T) {
	cfg := testConfig(t)
	r := NewBackendAPI(cfg, testDB(t), testLogger())

	wanted := map[string]bool{
		"GET /api/v1/admin/roles":            false,
		"GET /api/v1/admin/roles/:id":        false,
		"GET /api/v1/admin/permissions":      false,
		"POST /api/v1/admin/roles":           false,
		"PATCH /api/v1/admin/roles/:id":      false,
		"DELETE /api/v1/admin/roles/:id":     false,
		"PATCH /api/v1/admin/users/:id/role": false,
		"GET /api/v1/admin/auth/me":          false,
		"POST /api/v1/admin/auth/heartbeat":  false,
		"POST /api/v1/admin/auth/login":      false,
		// Fase C: dokumen legal & 2FA.
		"GET /api/v1/admin/legal":               false,
		"POST /api/v1/admin/legal":              false,
		"PATCH /api/v1/admin/legal/:id":         false,
		"POST /api/v1/admin/legal/:id/publish":  false,
		"GET /api/v1/admin/2fa":                 false,
		"POST /api/v1/admin/2fa/setup":          false,
		"POST /api/v1/admin/2fa/activate":       false,
		"POST /api/v1/admin/2fa/disable":        false,
		"POST /api/v1/admin/2fa/recovery-codes": false,
	}

	for _, info := range r.Routes() {
		key := info.Method + " " + info.Path
		if _, ok := wanted[key]; ok {
			wanted[key] = true
		}
	}

	for key, found := range wanted {
		if !found {
			t.Errorf("route %s tidak terpasang", key)
		}
	}
}

// Pintu admin tanpa sesi harus 401, bukan membuka data.
func TestEndpointAdminButuhSesi(t *testing.T) {
	cfg := testConfig(t)
	r := NewBackendAPI(cfg, testDB(t), testLogger())

	for _, path := range []string{"/api/v1/admin/roles", "/api/v1/admin/permissions", "/api/v1/admin/auth/me"} {
		w := performRequest(r, http.MethodGet, path)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s = %d; ingin 401", path, w.Code)
		}
	}

	// Heartbeat tanpa sesi juga harus ditolak: ia memperpanjang cookie sesi.
	w := performRequest(r, http.MethodPost, "/api/v1/admin/auth/heartbeat")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("heartbeat = %d; ingin 401", w.Code)
	}
}

// Sesi user tidak bisa dipakai di pintu admin meski cookie-nya ada.
//
// Di pintu admin, token ber-scope user dijawab 403 — bukan 401 — sesuai
// plans/04 §5.3. Yang penting: data admin tidak terbuka.
func TestSesiUserDitolakDiPintuAdmin(t *testing.T) {
	cfg := testConfig(t)
	r := NewBackendAPI(cfg, testDB(t), testLogger())

	token := issueTestToken(t, cfg, "u1", "user", "user")

	// Token user di cookie admin: scope salah, ditolak 403.
	w := performRequestWithCookie(r, http.MethodGet, "/api/v1/admin/roles", cfg.AdminCookieName, token)
	if w.Code != http.StatusForbidden {
		t.Fatalf("cookie admin + token user = %d; ingin 403", w.Code)
	}

	// Token user di cookie user tidak dikenali pintu admin: 401.
	w = performRequestWithCookie(r, http.MethodGet, "/api/v1/admin/roles", cfg.SessionCookieName, token)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("cookie user + token user = %d; ingin 401", w.Code)
	}
}

// Token admin yang sah tetap tertahan tanpa sesi yang tercatat di DB.
// Tes ini tidak memakai database sungguhan, jadi penolakan berasal dari
// pencocokan sesi — bukan dari lubang pada penjagaan awal.
func TestTokenAdminButuhSesiTercatat(t *testing.T) {
	cfg := testConfig(t)
	r := NewBackendAPI(cfg, testDB(t), testLogger())

	token := issueTestToken(t, cfg, "a1", "admin", "admin")
	w := performRequestWithCookie(r, http.MethodGet, "/api/v1/admin/roles", cfg.AdminCookieName, token)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; ingin 401 (sesi tidak terbaca dari DB)", w.Code)
	}
}

// Bila sebuah route admin kehilangan deklarasinya, pemeriksaan akhir harus
// melaporkannya — inilah yang membuat aplikasi menolak start (plans/04 §5.2).
func TestPemeriksaMenangkapDeklarasiHilang(t *testing.T) {
	cfg := testConfig(t)
	r := NewBackendAPI(cfg, testDB(t), testLogger())

	if missing := middleware.UndeclaredAdminRoutes(r.Routes()); len(missing) > 0 {
		t.Fatalf("router sehat dilaporkan bolong: %v", missing)
	}

	// Hapus seluruh deklarasi: sekarang setiap route admin harus dianggap bolong.
	middleware.ResetAdminRoutesForTest()
	missing := middleware.UndeclaredAdminRoutes(r.Routes())
	if len(missing) == 0 {
		t.Fatal("route admin tanpa deklarasi tidak terdeteksi")
	}

	// Pulihkan deklarasi dengan merangkai ulang router.
	NewBackendAPI(cfg, testDB(t), testLogger())
	if missing := middleware.UndeclaredAdminRoutes(r.Routes()); len(missing) > 0 {
		t.Fatalf("deklarasi tidak pulih: %v", missing)
	}
}

// Dokumen legal harus dapat dibaca tanpa masuk: calon pengguna perlu
// membacanya sebelum memutuskan mendaftar.
func TestLegalPublikTerbukaTanpaSesi(t *testing.T) {
	cfg := testConfig(t)
	r := NewBackendAPI(cfg, testDB(t), testLogger())

	// Tanpa database sungguhan permintaannya gagal di lapisan data, tetapi
	// yang diuji di sini adalah penjaganya: BUKAN 401.
	for _, path := range []string{"/api/v1/legal", "/api/v1/legal/tos"} {
		w := performRequest(r, http.MethodGet, path)
		if w.Code == http.StatusUnauthorized {
			t.Errorf("%s = 401; dokumen publik seharusnya tidak menuntut sesi", path)
		}
	}
}

// Endpoint 2FA dan persetujuan legal user wajib bersesi.
func TestLegalDanDuaFaktorButuhSesi(t *testing.T) {
	cfg := testConfig(t)
	r := NewBackendAPI(cfg, testDB(t), testLogger())

	for _, path := range []string{"/api/v1/me/legal", "/api/v1/2fa"} {
		w := performRequest(r, http.MethodGet, path)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s = %d; ingin 401", path, w.Code)
		}
	}
}

// Tanpa kunci master, 2FA sengaja tidak tersedia: menyimpan secret tanpa
// enkripsi lebih buruk daripada tidak ada 2FA sama sekali.
func TestDuaFaktorNonaktifTanpaMasterKey(t *testing.T) {
	cfg := testConfig(t)
	cfg.MasterKey = ""
	r := NewBackendAPI(cfg, testDB(t), testLogger())

	for _, path := range []string{"/api/v1/2fa", "/api/v1/admin/2fa"} {
		w := performRequest(r, http.MethodGet, path)
		if w.Code == http.StatusUnauthorized {
			t.Errorf("%s = 401; route seharusnya tidak terpasang tanpa MASTER_KEY", path)
		}
	}
}

// Dengan kunci master sah, route 2FA terpasang dan tetap menuntut sesi.
func TestDuaFaktorAktifDenganMasterKey(t *testing.T) {
	cfg := testConfig(t)
	cfg.MasterKey = sealerTestKey()
	r := NewBackendAPI(cfg, testDB(t), testLogger())

	for _, path := range []string{"/api/v1/2fa", "/api/v1/admin/2fa"} {
		w := performRequest(r, http.MethodGet, path)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s = %d; ingin 401", path, w.Code)
		}
	}
}

// Registrasi tanpa centang persetujuan harus ditolak sebelum menyentuh data.
func TestRegistrasiTanpaCentangDitolakHTTP(t *testing.T) {
	cfg := testConfig(t)
	r := NewBackendAPI(cfg, testDB(t), testLogger())

	body := `{"name":"Uji Coba","email":"uji@contoh.id","password":"rahasia-panjang-123"}`
	w := performJSONRequest(r, http.MethodPost, "/api/v1/auth/register", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; ingin 400 (body: %s)", w.Code, w.Body.String())
	}
}
