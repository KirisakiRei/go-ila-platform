package dto

// Pengujian tag binding Registrasi terhadap validator yang dipakai Gin.
//
// Kasus nyata: user yang mendaftar lewat tautan undangan memakai username
// `budi.santoso` dan ditolak server dengan pesan
// "failed on the 'alphanum' tag". Padahal frontend mengizinkan titik, garis
// bawah, dan strip (schemas.ts), dan service layer juga mengizinkannya
// (auth_service.go isValidUsername). Yang menolak hanya tag binding DTO,
// sehingga aturan di tiga tempat saling bertentangan.
//
// Pengujian ini mengunci kesepakatan: DTO hanya memeriksa panjang, sedangkan
// bentuk karakter diperiksa oleh service (satu sumber kebenaran) agar tidak
// ada lagi aturan ganda yang bisa berbeda.

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

func registerValid() RegisterRequest {
	return RegisterRequest{
		Name:        "Budi Santoso",
		Username:    "budisantoso",
		Email:       "budi@example.com",
		Password:    "rahasia-panjang-123",
		AcceptTerms: true,
	}
}

// validatorBinding meniru Gin: Gin membaca tag `binding`, sedangkan validator
// bawaan membaca `validate`. Tanpa SetTagName, semua pengujian di file ini lolos
// palsu karena tag binding tidak pernah dibaca.
func validatorBinding() *validator.Validate {
	v := validator.New()
	v.SetTagName("binding")
	return v
}

// Username dasar (huruf dan angka) harus lolos.
func TestRegisterRequestUsernameAlfanumerikSah(t *testing.T) {
	if err := validatorBinding().Struct(registerValid()); err != nil {
		t.Fatalf("username huruf+angka seharusnya sah, dapat galat: %v", err)
	}
}

// Titik, garis bawah, dan strip harus lolos binding DTO.
// Bentuk karakternya diperiksa service layer, bukan di sini.
func TestRegisterRequestUsernameBersimbolLolosBinding(t *testing.T) {
	for _, username := range []string{
		"budi.santoso",
		"budi_santoso",
		"budi-santoso",
		"budi.s._-9",
	} {
		req := registerValid()
		req.Username = username
		if err := validatorBinding().Struct(req); err != nil {
			t.Errorf("username %q seharusnya lolos binding, dapat galat: %v", username, err)
		}
	}
}

// Batas panjang tetap dijaga di DTO.
func TestRegisterRequestUsernamePanjangDijaga(t *testing.T) {
	req := registerValid()
	req.Username = "ab" // kurang dari 3
	if err := validatorBinding().Struct(req); err == nil {
		t.Error("username 2 karakter seharusnya ditolak")
	}

	req.Username = "abcdefghijklmnopqrstuvwxyz0123456" // 33 karakter
	if err := validatorBinding().Struct(req); err == nil {
		t.Error("username 33 karakter seharusnya ditolak")
	}
}

// Username boleh kosong (klien lama belum mengirimnya).
func TestRegisterRequestUsernameBolehKosong(t *testing.T) {
	req := registerValid()
	req.Username = ""
	if err := validatorBinding().Struct(req); err != nil {
		t.Fatalf("username kosong seharusnya sah, dapat galat: %v", err)
	}
}
