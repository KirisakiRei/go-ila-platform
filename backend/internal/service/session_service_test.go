package service_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/service"
	"ila/backend/pkg/apperror"
)

type fakeSessionRepo struct {
	sessions []model.Session
	revoked  []string
}

func (f *fakeSessionRepo) ListByUser(_ context.Context, userID, scope string) ([]model.Session, error) {
	var out []model.Session
	for _, s := range f.sessions {
		if s.UserID == userID && s.Scope == scope {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeSessionRepo) Revoke(_ context.Context, id, userID string) error {
	for i := range f.sessions {
		if f.sessions[i].ID == id && f.sessions[i].UserID == userID {
			f.revoked = append(f.revoked, id)
			return nil
		}
	}
	// Repository sungguhan mengembalikan ErrNotFound; samakan agar tes
	// tidak memberi rasa aman palsu.
	return apperror.ErrNotFound
}

func (f *fakeSessionRepo) RevokeAll(_ context.Context, _, _ string) error { return nil }
func (f *fakeSessionRepo) Touch(_ context.Context, _ string, _ time.Time) error {
	return nil
}

// Sesi milik user lain tidak boleh dicabut lewat ID-nya.
func TestCabutSesiMilikOrangLainDitolak(t *testing.T) {
	repo := &fakeSessionRepo{
		sessions: []model.Session{{ID: "s-1", UserID: "pemilik", Scope: "user"}},
	}
	svc := service.NewSessionService(repo, nil, nil)

	_ = svc.Revoke(context.Background(), "penyusup", "s-1")

	if len(repo.revoked) != 0 {
		t.Fatal("sesi milik orang lain tidak boleh tercabut")
	}
}

// Sesi perangkat ini harus ditandai agar UI bisa membedakannya.
func TestDaftarSesiMenandaiPerangkatIni(t *testing.T) {
	repo := &fakeSessionRepo{
		sessions: []model.Session{
			{ID: "s-1", UserID: "u1", Scope: "user", LastSeenAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Hour)},
			{ID: "s-2", UserID: "u1", Scope: "user", LastSeenAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Hour)},
		},
	}
	svc := service.NewSessionService(repo, nil, nil)

	items, err := svc.List(context.Background(), "u1", "user", "s-2")
	if err != nil {
		t.Fatalf("List galat: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("jumlah sesi = %d, ingin 2", len(items))
	}

	var current int
	for _, item := range items {
		if item.Current {
			current++
			if item.ID != "s-2" {
				t.Fatalf("sesi ini seharusnya s-2, dapat %s", item.ID)
			}
		}
		// Waktu harus tersaji dalam dua zona.
		if item.LastSeen.WIB == "" || item.LastSeen.NY == "" {
			t.Fatal("waktu sesi wajib memuat WIB dan NY")
		}
	}
	if current != 1 {
		t.Fatalf("sesi bertanda ini = %d, ingin 1", current)
	}
}

// User agent dikenali menjadi sebutan yang bisa dibaca manusia.
func TestLabelPerangkatDikenali(t *testing.T) {
	repo := &fakeSessionRepo{
		sessions: []model.Session{{
			ID: "s-1", UserID: "u1", Scope: "user",
			UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0 Safari/537.36",
			LastSeenAt: time.Now().UTC(),
			ExpiresAt:  time.Now().Add(time.Hour),
		}},
	}
	svc := service.NewSessionService(repo, nil, nil)

	items, _ := svc.List(context.Background(), "u1", "user", "")
	if len(items) != 1 {
		t.Fatalf("jumlah sesi = %d, ingin 1", len(items))
	}
	if !strings.Contains(items[0].DeviceLabel, "Chrome") {
		t.Fatalf("label = %q, ingin memuat Chrome", items[0].DeviceLabel)
	}
	if !strings.Contains(items[0].DeviceLabel, "Windows") {
		t.Fatalf("label = %q, ingin memuat Windows", items[0].DeviceLabel)
	}
}

// fakeRevoker mencatat permintaan pencabutan seluruh sesi.
type fakeRevoker struct {
	users []string
}

func (f *fakeRevoker) RevokeSessions(_ context.Context, userID string) error {
	f.users = append(f.users, userID)
	return nil
}

// Mencabut satu perangkat harus menaikkan session_version, karena verifikasi
// per permintaan memakai nilai itu — tanpa ini token perangkat itu tetap sah.
func TestCabutSesiMenaikkanVersiSesi(t *testing.T) {
	repo := &fakeSessionRepo{
		sessions: []model.Session{{ID: "s-1", UserID: "u1", Scope: "user"}},
	}
	revoker := &fakeRevoker{}
	svc := service.NewSessionService(repo, revoker, nil)

	if err := svc.Revoke(context.Background(), "u1", "s-1"); err != nil {
		t.Fatalf("Revoke galat: %v", err)
	}

	if len(revoker.users) != 1 || revoker.users[0] != "u1" {
		t.Fatalf("pencabutan versi sesi tidak dipanggil: %v", revoker.users)
	}
}

// Sesi milik orang lain tidak boleh memicu pencabutan apa pun.
func TestCabutSesiAsingTidakMenaikkanVersi(t *testing.T) {
	repo := &fakeSessionRepo{
		sessions: []model.Session{{ID: "s-1", UserID: "pemilik", Scope: "user"}},
	}
	revoker := &fakeRevoker{}
	svc := service.NewSessionService(repo, revoker, nil)

	_ = svc.Revoke(context.Background(), "penyusup", "s-1")

	if len(revoker.users) != 0 {
		t.Fatalf("versi sesi tidak boleh dinaikkan untuk penyusup: %v", revoker.users)
	}
}
