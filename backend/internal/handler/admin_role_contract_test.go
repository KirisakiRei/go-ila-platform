package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/handler"
	"ila/backend/internal/middleware"
)

// stubRoleService hanya melayani ListRoles untuk uji kontrak balasan.
type stubRoleService struct {
	roles []dto.RoleResponse
}

func (s stubRoleService) ListRoles(context.Context, string) ([]dto.RoleResponse, error) {
	return s.roles, nil
}

func (s stubRoleService) GetRole(context.Context, string, string) (*dto.RoleDetailResponse, error) {
	return nil, nil
}

func (s stubRoleService) CreateRole(context.Context, string, dto.CreateRoleRequest) (*dto.RoleResponse, error) {
	return nil, nil
}

func (s stubRoleService) UpdateRole(context.Context, string, string, dto.UpdateRoleRequest) (*dto.RoleResponse, error) {
	return nil, nil
}

func (s stubRoleService) DeleteRole(context.Context, string, string) error {
	return nil
}

func (s stubRoleService) AssignRole(context.Context, string, string, dto.AssignRoleRequest) error {
	return nil
}

func (s stubRoleService) ListPermissions(context.Context, string) ([]dto.PermissionGroup, error) {
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
	r.GET("/roles", handler.NewAdminRoleHandler(stub).List)

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
	r.GET("/permissions", handler.NewAdminRoleHandler(katalogStub{}).Permissions)

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
