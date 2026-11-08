package service

import (
	"context"
	"testing"

	"ila/backend/internal/dto"
	"ila/backend/pkg/apperror"
)

// Username disimpan huruf kecil supaya satu identitas tidak bisa didaftarkan
// dua kali dengan beda kapitalisasi.
func TestRegisterMenyimpanUsernameHurufKecil(t *testing.T) {
	svc, _, _ := newTestAuth()

	got, err := svc.Register(context.Background(), dto.RegisterRequest{
		Name:     "Budi Santoso",
		Username: "  @BudiS  ",
		Email:    "Budi@Example.com",
		Phone:    "+62 812-3456-7890",
		Password: "rahasia-panjang-123",
	})
	if err != nil {
		t.Fatalf("registrasi gagal: %v", err)
	}
	if got.Username != "budis" {
		t.Errorf("username ingin budis, dapat %q", got.Username)
	}
	if got.Phone != "+6281234567890" {
		t.Errorf("phone ingin +6281234567890, dapat %q", got.Phone)
	}
	if got.Email != "budi@example.com" {
		t.Errorf("email ingin huruf kecil, dapat %q", got.Email)
	}
}

// Dua akun berbeda tidak boleh memakai username yang sama.
func TestRegisterUsernameGandaDitolak(t *testing.T) {
	svc, _, _ := newTestAuth()
	ctx := context.Background()

	_, err := svc.Register(ctx, dto.RegisterRequest{
		Name: "Budi", Username: "budi", Email: "budi@example.com", Password: "rahasia-panjang-123",
	})
	if err != nil {
		t.Fatalf("registrasi pertama gagal: %v", err)
	}

	_, err = svc.Register(ctx, dto.RegisterRequest{
		Name: "Budi Lain", Username: "BUDI", Email: "lain@example.com", Password: "rahasia-panjang-123",
	})
	if err != apperror.ErrAlreadyExists {
		t.Errorf("username sama harus ErrAlreadyExists, dapat %v", err)
	}
}

// Username dengan spasi atau simbol membingungkan saat masuk, jadi ditolak.
func TestRegisterUsernameTidakSahDitolak(t *testing.T) {
	for _, username := range []string{"bu di", "budi!", "a", "budi/santoso"} {
		svc, _, _ := newTestAuth()
		_, err := svc.Register(context.Background(), dto.RegisterRequest{
			Name: "Budi", Username: username, Email: "budi@example.com", Password: "rahasia-panjang-123",
		})
		if err != apperror.ErrBadRequest {
			t.Errorf("username %q ingin ErrBadRequest, dapat %v", username, err)
		}
	}
}

// Tanpa username pun akun tetap sah: klien lama belum mengirim kolom itu.
func TestRegisterTanpaUsernameTetapBerhasil(t *testing.T) {
	svc, _, _ := newTestAuth()

	got, err := svc.Register(context.Background(), dto.RegisterRequest{
		Name: "Budi", Email: "budi@example.com", Password: "rahasia-panjang-123",
	})
	if err != nil {
		t.Fatalf("registrasi tanpa username gagal: %v", err)
	}
	if got.Username != "" {
		t.Errorf("username ingin kosong, dapat %q", got.Username)
	}
}

// Satu kolom masuk harus menerima email maupun username.
func TestLoginMenerimaEmailDanUsername(t *testing.T) {
	svc, _, _ := newTestAuth()
	ctx := context.Background()

	_, err := svc.Register(ctx, dto.RegisterRequest{
		Name: "Budi", Username: "budi", Email: "budi@example.com", Password: "rahasia-panjang-123",
	})
	if err != nil {
		t.Fatalf("registrasi gagal: %v", err)
	}

	for _, identifier := range []string{"budi@example.com", "budi", "BUDI"} {
		got, err := svc.Login(ctx, dto.LoginRequest{Identifier: identifier, Password: "rahasia-panjang-123"})
		if err != nil {
			t.Errorf("login dengan %q gagal: %v", identifier, err)
			continue
		}
		if got.Token == "" {
			t.Errorf("login dengan %q tidak menerbitkan token", identifier)
		}
	}
}

// Username yang tidak terdaftar harus tampak sama seperti password salah.
func TestLoginUsernameTidakAdaTidakMembocorkan(t *testing.T) {
	svc, _, _ := newTestAuth()

	_, err := svc.Login(context.Background(), dto.LoginRequest{Identifier: "siapa-yang-tahu", Password: "apa-saja"})
	if err != apperror.ErrUnauthorized {
		t.Errorf("username tak dikenal harus ErrUnauthorized, dapat %v", err)
	}
}
