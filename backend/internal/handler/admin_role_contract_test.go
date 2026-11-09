package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/handler"
	"ila/backend/internal/middleware"
	"ila/backend/pkg/httpx"
)

// stubRoleService melayani kontrak balasan daftar role dan penghapusan akun
// pengelola.
type stubRoleService struct {
	roles     []dto.RoleResponse
	deleted   []string
	deleteErr error
}

func (s *stubRoleService) ListRoles(context.Context, string) ([]dto.RoleResponse, error) {
	return s.roles, nil
}

func (s *stubRoleService) GetRole(context.Context, string, string) (*dto.RoleDetailResponse, error) {
	return nil, nil
}

func (s *stubRoleService) CreateRole(context.Context, string, dto.CreateRoleRequest) (*dto.RoleResponse, error) {
	return nil, nil
}

func (s *stubRoleService) UpdateRole(context.Context, string, string, dto.UpdateRoleRequest) (*dto.RoleResponse, error) {
	return nil, nil
}

func (s *stubRoleService) DeleteRole(context.Context, string, string) error {
	return nil
}

func (s *stubRoleService) AssignRole(context.Context, string, string, dto.AssignRoleRequest) error {
	return nil
}

func (s *stubRoleService) CreateAdmin(context.Context, string, dto.CreateAdminRequest) (*dto.UserListItem, error) {
	return nil, nil
}

func (s *stubRoleService) DeleteAdmin(_ context.Context, _ /*actorID*/, userID, reason string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, userID+"|"+reason)
	return nil
}

func (s *stubRoleService) ListPermissions(context.Context, string) ([]dto.PermissionGroup, error) {
	return nil, nil
}

/*
Bentuk balasan GET /admin/roles harus sesuai yang dibaca antarmuka.

Bug yang dijaga di sini: halaman RBAC gagal dengan "Balasan server tidak dapat
dibaca". Penyebabnya bukan status HTTP (server membalas 200), melainkan bentuk
`data` — bila yang dikirim sebuah objek berkunci (mis. `{"roles": [...]}`)
padahal panel mengharapkan larik, panel tidak bisa membaca daftarnya.

Kunci yang dipegang: `data` adalah LARIK role, dan setiap role memuat
`permissions` sebagai larik string, `editable`, dan `is_system` yang dipakai
panel untuk mengunci tombol.
*/
func TestDaftarRoleDikirimSebagaiLarik(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))

	stub := stubRoleService{roles: []dto.RoleResponse{
		{
			ID:          "role-1",
			Code:        "marketing",
			Name:        "Marketing",
			Description: "Mengurus kampanye",
			IsSystem:    false,
			Editable:    true,
			Permissions: []string{"cms.view", "billing.voucher"},
			UserCount:   2,
			CreatedAt:   "2026-11-09T00:00:00Z",
		},
	}}
	r.GET("/roles", handler.NewAdminRoleHandler(&stub).List)

	req := httptest.NewRequest(http.MethodGet, "/roles", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, seharusnya 200", rec.Code)
	}

	var envelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("balasan bukan JSON: %v", err)
	}
	if !envelope.Success {
		t.Fatal("success harus true")
	}

	// Harus larik, bukan objek: panel memakai `data` langsung sebagai daftar.
	if len(envelope.Data) == 0 || envelope.Data[0] != '[' {
		t.Fatalf("data harus larik JSON, diterima: %s", truncate(string(envelope.Data)))
	}

	var roles []map[string]json.RawMessage
	if err := json.Unmarshal(envelope.Data, &roles); err != nil {
		t.Fatalf("data bukan larik role: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("jumlah role = %d, seharusnya 1", len(roles))
	}

	// `permissions` wajib larik string; panel menghitung panjangnya.
	var perms []string
	if err := json.Unmarshal(roles[0]["permissions"], &perms); err != nil {
		t.Fatalf("permissions bukan larik string: %v (%s)", err, roles[0]["permissions"])
	}
	if len(perms) != 2 {
		t.Errorf("permissions = %v, seharusnya 2 izin", perms)
	}

	for _, key := range []string{"id", "code", "name", "editable", "is_system", "user_count", "created_at"} {
		if _, ok := roles[0][key]; !ok {
			t.Errorf("field %q tidak ada di balasan", key)
		}
	}
}

// Katalog permission dipakai panel untuk merender matriks centang: harus larik
// grup, dan setiap grup memuat larik `permissions` dengan `grantable`.
func TestKatalogPermissionDikirimSebagaiLarikGrup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))
	r.GET("/permissions", handler.NewAdminRoleHandler(&katalogStub{}).Permissions)

	req := httptest.NewRequest(http.MethodGet, "/permissions", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("balasan bukan JSON: %v", err)
	}
	if len(envelope.Data) == 0 || envelope.Data[0] != '[' {
		t.Fatalf("data harus larik grup, diterima: %s", truncate(string(envelope.Data)))
	}

	var groups []struct {
		Group       string `json:"group"`
		Permissions []struct {
			Key       string `json:"key"`
			Label     string `json:"label"`
			Risk      string `json:"risk"`
			Grantable bool   `json:"grantable"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(envelope.Data, &groups); err != nil {
		t.Fatalf("data bukan larik grup permission: %v", err)
	}
	if len(groups) != 1 || groups[0].Group != "Tata Kelola" {
		t.Fatalf("grup tak terduga: %+v", groups)
	}
	if len(groups[0].Permissions) != 1 || groups[0].Permissions[0].Key != "role.view" {
		t.Fatalf("permission tak terduga: %+v", groups[0].Permissions)
	}
}

type katalogStub struct{ stubRoleService }

func (katalogStub) ListPermissions(context.Context, string) ([]dto.PermissionGroup, error) {
	return []dto.PermissionGroup{
		{
			Group: "Tata Kelola",
			Permissions: []dto.PermissionResponse{
				{Key: "role.view", Group: "Tata Kelola", Label: "Lihat role", Risk: "rendah", Grantable: true},
			},
		},
	}, nil
}

func truncate(s string) string {
	if len(s) <= 200 {
		return s
	}
	return s[:200] + "…"
}

/*
Penghapusan akun pengelola lewat DELETE /admin/admins/:id.

Yang dijaga:
  - Alasan wajib dan minimal 10 karakter; lebih pendek ditolak sebelum service
    sempat menghapus apa pun.
  - Alasan boleh datang dari body JSON maupun query, sehingga panel dan
    perkakas CLI sama-sama bisa memakainya.
  - Penolakan service (mis. akun terlindungi) diteruskan sebagai galat, bukan
    dibalas 200.
*/
func TestHapusAdminButuhAlasan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := stubRoleService{}
	r := gin.New()
	r.Use(middleware.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))
	r.DELETE("/admins/:id", handler.NewAdminRoleHandler(&stub).DeleteAdmin)

	for _, body := range []string{`{}`, `{"reason":"pendek"}`, `bukan json`} {
		req := httptest.NewRequest(http.MethodDelete, "/admins/u9", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %q: status = %d, seharusnya 400", body, rec.Code)
		}
	}
	if len(stub.deleted) != 0 {
		t.Fatalf("penghapusan dijalankan tanpa alasan sah: %v", stub.deleted)
	}
}

func TestHapusAdminMeneruskanAlasan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const alasan = "Akun tidak lagi bertugas di tim operasional"

	kasus := []struct {
		nama string
		url  string
		body string
	}{
		{"body", "/admins/u9", `{"reason":"` + alasan + `"}`},
		{"query", "/admins/u9?reason=Akun+tidak+lagi+bertugas+di+tim+operasional", ``},
	}

	for _, kasus := range kasus {
		t.Run(kasus.nama, func(t *testing.T) {
			stub := stubRoleService{}
			r := gin.New()
			r.Use(middleware.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))
			r.DELETE("/admins/:id", handler.NewAdminRoleHandler(&stub).DeleteAdmin)

			req := httptest.NewRequest(http.MethodDelete, kasus.url, strings.NewReader(kasus.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body: %s", rec.Code, truncate(rec.Body.String()))
			}
			var envelope struct {
				Success bool `json:"success"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil || !envelope.Success {
				t.Fatalf("balasan bukan JSON sukses: %s", truncate(rec.Body.String()))
			}
			if len(stub.deleted) != 1 {
				t.Fatalf("penghapusan tidak diteruskan: %v", stub.deleted)
			}
			if !strings.HasPrefix(stub.deleted[0], "u9|Akun tidak lagi bertugas") {
				t.Errorf("argumen service = %q", stub.deleted[0])
			}
		})
	}
}

// Penolakan service wajib sampai ke panel sebagai galat, bukan 200 kosong.
func TestHapusAdminPenolakanDiteruskan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := stubRoleService{deleteErr: httpx.WrapValidation("akun superadmin utama tidak dapat dihapus")}

	r := gin.New()
	r.Use(middleware.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))
	r.DELETE("/admins/:id", handler.NewAdminRoleHandler(&stub).DeleteAdmin)

	req := httptest.NewRequest(http.MethodDelete, "/admins/u9",
		strings.NewReader(`{"reason":"Akun tidak lagi bertugas di tim"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, seharusnya 400 (body: %s)", rec.Code, truncate(rec.Body.String()))
	}
	if !strings.Contains(rec.Body.String(), "superadmin utama") {
		t.Errorf("alasan penolakan tidak sampai ke panel: %s", truncate(rec.Body.String()))
	}
}
