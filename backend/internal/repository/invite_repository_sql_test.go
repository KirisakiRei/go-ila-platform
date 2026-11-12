package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ila/backend/internal/model"
)

// dbKeringKapSQL menyiapkan basis data tanpa Postgres dan mengembalikan
// penangkap SQL untuk operasi update.
//
// Tanpa Postgres sungguhan, tes repository biasanya ter-skip — dan justru di
// situlah bug SQL bersembunyi. Callback ini memeriksa SQL yang benar-benar
// disusun fungsi repository aslinya. Pola yang sama sudah dipakai
// TestListForUserTidakMemakaiDistinct untuk menangkap SQLSTATE 42P10.
func dbKeringKapSQL(t *testing.T) (*gorm.DB, *string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:invite?mode=memory&cache=shared"), &gorm.Config{
		DryRun: true,
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("membuka basis data: %v", err)
	}

	var sql string
	db.Callback().Update().After("gorm:update").Register("uji:tangkap_sql_update", func(d *gorm.DB) {
		sql = d.Statement.SQL.String()
	})
	return db, &sql
}

// TestKlaimKuotaMenyertakanSyaratKuotaDiSQL mengunci hal yang TIDAK bisa
// ditangkap tiruan di memori: syarat kuota wajib ada di klausa WHERE.
//
// Kalau syarat itu hilang, UPDATE memotong kuota tanpa memeriksa sisa — dan
// satu tautan berkuota 1 bisa menghasilkan akun tanpa batas. Tiruan di memori
// tetap hijau untuk kerusakan itu, karena tiruannya menegakkan syarat sendiri.
func TestKlaimKuotaMenyertakanSyaratKuotaDiSQL(t *testing.T) {
	db, sql := dbKeringKapSQL(t)
	repo := NewInviteRepository(db)

	// Gagal di driver tidak masalah: yang diuji adalah SQL yang disusun.
	_ = repo.KlaimKuota(context.Background(), "inv-1", time.Now().UTC())

	if *sql == "" {
		t.Fatalf("SQL tidak tertangkap; uji tidak dapat menyimpulkan apa pun")
	}
	for _, wajib := range []string{"used_count < max_uses", "is_active", "expires_at"} {
		if !strings.Contains(*sql, wajib) {
			t.Fatalf("SQL tidak memuat syarat %q:\n%s", wajib, *sql)
		}
	}
}

// TestKlaimKuotaMenambahDiDatabase memastikan pemotongan dilakukan database,
// bukan dihitung di Go dari nilai lama (yang bisa balapan antar dua permintaan).
func TestKlaimKuotaMenambahDiDatabase(t *testing.T) {
	db, sql := dbKeringKapSQL(t)
	repo := NewInviteRepository(db)

	_ = repo.KlaimKuota(context.Background(), "inv-1", time.Now().UTC())

	if *sql == "" {
		t.Fatalf("SQL tidak tertangkap")
	}
	if !strings.Contains(*sql, "used_count + 1") {
		t.Fatalf("pemotongan kuota harus dihitung database:\n%s", *sql)
	}
}

// TestKembalikanKuotaTidakTurunDiBawahNol menjaga kolom tetap wajar walau
// dipanggil dua kali karena galat berantai.
func TestKembalikanKuotaTidakTurunDiBawahNol(t *testing.T) {
	db, sql := dbKeringKapSQL(t)
	repo := NewInviteRepository(db)

	_ = repo.KembalikanKuota(context.Background(), "inv-1")

	if *sql == "" {
		t.Fatalf("SQL tidak tertangkap")
	}
	if !strings.Contains(*sql, "used_count > 0") {
		t.Fatalf("pengembalian kuota harus dibatasi used_count > 0:\n%s", *sql)
	}
}

// TestModelUndanganTertabel memastikan nama tabel sesuai rancangan.
func TestModelUndanganTertabel(t *testing.T) {
	if got := (model.InviteCode{}).TableName(); got != "invite_code" {
		t.Fatalf("TableName = %q", got)
	}
	if got := (model.InviteRedemption{}).TableName(); got != "invite_redemption" {
		t.Fatalf("TableName = %q", got)
	}
}
