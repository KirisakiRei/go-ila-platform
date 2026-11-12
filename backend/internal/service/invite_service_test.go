package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// ---------------------------------------------------------------------------
// Tiruan
// ---------------------------------------------------------------------------

// inviteStoreTiruan menyimpan undangan di memori dan meniru UPDATE bersyarat.
type inviteStoreTiruan struct {
	mu        sync.Mutex
	rows      map[string]*model.InviteCode
	pemakaian []model.InviteRedemption
}

func newInviteStoreTiruan() *inviteStoreTiruan {
	return &inviteStoreTiruan{rows: map[string]*model.InviteCode{}}
}

func (s *inviteStoreTiruan) Create(_ context.Context, invite *model.InviteCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if invite.ID == "" {
		invite.ID = fmt.Sprintf("inv-%d", len(s.rows)+1)
	}
	salin := *invite
	s.rows[invite.ID] = &salin
	return nil
}

func (s *inviteStoreTiruan) FindByHash(_ context.Context, hash string) (*model.InviteCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, row := range s.rows {
		if row.CodeHash == hash {
			salin := *row
			return &salin, nil
		}
	}
	return nil, apperror.ErrNotFound
}

func (s *inviteStoreTiruan) FindByID(_ context.Context, id string) (*model.InviteCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[id]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	salin := *row
	return &salin, nil
}

func (s *inviteStoreTiruan) List(context.Context, string, int) ([]model.InviteCode, error) {
	return nil, nil
}

// KlaimKuota meniru UPDATE ... WHERE used_count < max_uses: satu-satunya titik
// yang menentukan apakah kuota bisa dilampaui. Diuji dengan banyak goroutine.
func (s *inviteStoreTiruan) KlaimKuota(_ context.Context, id string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[id]
	if !ok {
		return apperror.ErrNotFound
	}
	if !row.IsActive || !now.Before(row.ExpiresAt) || row.UsedCount >= row.MaxUses {
		return apperror.ErrConflict
	}
	row.UsedCount++
	return nil
}

func (s *inviteStoreTiruan) KembalikanKuota(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[id]
	if !ok {
		return apperror.ErrNotFound
	}
	if row.UsedCount > 0 {
		row.UsedCount--
	}
	return nil
}

func (s *inviteStoreTiruan) Revoke(_ context.Context, id string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[id]
	if !ok {
		return apperror.ErrNotFound
	}
	row.IsActive = false
	return nil
}

func (s *inviteStoreTiruan) CatatPemakaian(_ context.Context, r *model.InviteRedemption) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pemakaian = append(s.pemakaian, *r)
	return nil
}

func (s *inviteStoreTiruan) ListPemakaian(context.Context, string) ([]model.InviteRedemption, error) {
	return s.pemakaian, nil
}

// authTiruan membuat akun tiruan dan bisa dipaksa gagal.
type authTiruan struct {
	mu          sync.Mutex
	dibuat      int
	gagalDengan error
	diterapkan  []string
}

func (a *authTiruan) Register(_ context.Context, req dto.RegisterRequest) (*dto.UserResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.gagalDengan != nil {
		return nil, a.gagalDengan
	}
	a.dibuat++
	return &dto.UserResponse{ID: fmt.Sprintf("user-%d", a.dibuat), Email: req.Email}, nil
}

func (a *authTiruan) Login(context.Context, dto.LoginRequest) (*dto.LoginResponse, error) {
	return nil, nil
}
func (a *authTiruan) Profile(context.Context, string) (*dto.UserResponse, error) { return nil, nil }
func (a *authTiruan) ChangePassword(context.Context, string, dto.ChangePasswordRequest) error {
	return nil
}
func (a *authTiruan) LogoutAll(context.Context, string) error            { return nil }
func (a *authTiruan) SetIdleWarning(context.Context, string, bool) error { return nil }
func (a *authTiruan) VerifyTwoFactor(context.Context, dto.VerifyTwoFactorRequest, LoginContext) (*dto.LoginResponse, error) {
	return nil, nil
}
func (a *authTiruan) TerapkanUndangan(_ context.Context, userID, tier string, _ time.Time) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.diterapkan = append(a.diterapkan, userID+":"+tier)
	return nil
}

// ---------------------------------------------------------------------------
// Pembantu
// ---------------------------------------------------------------------------

// buatUndangan membuat undangan lewat service dan mengembalikan kode mentahnya.
func buatUndangan(t *testing.T, svc *inviteService, req dto.CreateInviteRequest) *dto.CreatedInvite {
	t.Helper()
	created, err := svc.Create(context.Background(), AdminActor{ID: "admin-1"}, req)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	return created
}

func payloadDaftar(email string) dto.RegisterRequest {
	return dto.RegisterRequest{Name: "Uji Undangan", Email: email, Password: "rahasia-panjang-123", AcceptTerms: true}
}

// ---------------------------------------------------------------------------
// Tes
// ---------------------------------------------------------------------------

// TestUndanganSahMembuatAkunTierPlus mengunci jalur bahagia.
func TestUndanganSahMembuatAkunTierPlus(t *testing.T) {
	store := newInviteStoreTiruan()
	auth := &authTiruan{}
	svc := NewInviteService(store, auth, nil)

	created := buatUndangan(t, svc, dto.CreateInviteRequest{MaxUses: 1, Tier: "plus"})
	if created.Code == "" || !strings.Contains(created.URL, created.Code) {
		t.Fatalf("kode tautan tidak terbentuk: %+v", created)
	}

	user, err := svc.Klaim(context.Background(), created.Code, payloadDaftar("budi@contoh.id"))
	if err != nil {
		t.Fatalf("Klaim error: %v", err)
	}
	if user == nil || auth.dibuat != 1 {
		t.Fatalf("akun tidak dibuat: dibuat=%d", auth.dibuat)
	}

	// Hak undangan: tier plus diterapkan ke akun.
	if len(auth.diterapkan) != 1 || !strings.HasSuffix(auth.diterapkan[0], ":plus") {
		t.Fatalf("tier undangan tidak diterapkan: %v", auth.diterapkan)
	}

	// Balasan ke klien harus membawa tier yang SAMA dengan database. Objek ini
	// dibentuk sebelum hak undangan ditulis, jadi tanpa pembaruan eksplisit ia
	// masih berkata "free" dan menyesatkan pengguna yang baru masuk.
	if user.Tier != "plus" {
		t.Fatalf("tier pada balasan Klaim = %q; ingin \"plus\"", user.Tier)
	}

	// Kuota terpakai satu.
	row, _ := store.FindByHash(context.Background(), HashInviteCode(created.Code))
	if row.UsedCount != 1 {
		t.Fatalf("used_count = %d; ingin 1", row.UsedCount)
	}
	if len(store.pemakaian) != 1 {
		t.Fatalf("pemakaian tidak tercatat: %d", len(store.pemakaian))
	}
}

// TestKodeMentahTidakDisimpan mengunci janji keamanan: database hanya menyimpan hash.
func TestKodeMentahTidakDisimpan(t *testing.T) {
	store := newInviteStoreTiruan()
	svc := NewInviteService(store, &authTiruan{}, nil)
	created := buatUndangan(t, svc, dto.CreateInviteRequest{MaxUses: 5})

	for _, row := range store.rows {
		if row.CodeHash == created.Code {
			t.Fatal("kode mentah tersimpan di kolom hash")
		}
		if strings.Contains(row.CodeHash, created.Code) {
			t.Fatal("kode mentah muncul di dalam nilai tersimpan")
		}
	}
	if len(created.Code) < 40 {
		t.Fatalf("kode terlalu pendek: %d karakter", len(created.Code))
	}
}

// TestKuotaTidakBisaDilampauiBersamaan adalah tes terpenting: dua orang
// mengklaim slot terakhir serentak. Harus tepat satu yang berhasil.
func TestKuotaTidakBisaDilampauiBersamaan(t *testing.T) {
	store := newInviteStoreTiruan()
	auth := &authTiruan{}
	svc := NewInviteService(store, auth, nil)
	created := buatUndangan(t, svc, dto.CreateInviteRequest{MaxUses: 1})

	const percobaan = 20
	var wg sync.WaitGroup
	var berhasil int32
	var mu sync.Mutex

	for i := 0; i < percobaan; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := svc.Klaim(context.Background(), created.Code, payloadDaftar(fmt.Sprintf("u%d@contoh.id", n)))
			if err == nil {
				mu.Lock()
				berhasil++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	if berhasil != 1 {
		t.Fatalf("berhasil = %d akun; ingin tepat 1 (kuota 1)", berhasil)
	}
	if auth.dibuat != 1 {
		t.Fatalf("akun dibuat = %d; ingin 1", auth.dibuat)
	}
}

// TestKuotaKembaliBilaPendaftaranGagal mengunci saran no. 3: pendaftaran yang
// tidak jadi tidak boleh menghabiskan jatah orang lain.
func TestKuotaKembaliBilaPendaftaranGagal(t *testing.T) {
	store := newInviteStoreTiruan()
	auth := &authTiruan{gagalDengan: apperror.ErrAlreadyExists}
	svc := NewInviteService(store, auth, nil)
	created := buatUndangan(t, svc, dto.CreateInviteRequest{MaxUses: 2})

	_, err := svc.Klaim(context.Background(), created.Code, payloadDaftar("sudah@contoh.id"))
	if !errors.Is(err, apperror.ErrAlreadyExists) {
		t.Fatalf("error = %v; ingin ErrAlreadyExists", err)
	}

	row, _ := store.FindByHash(context.Background(), HashInviteCode(created.Code))
	if row.UsedCount != 0 {
		t.Fatalf("used_count = %d; ingin 0 (kuota harus kembali)", row.UsedCount)
	}
}

// TestUndanganKedaluwarsaDitolak tanpa menyentuh akun.
func TestUndanganKedaluwarsaDitolak(t *testing.T) {
	store := newInviteStoreTiruan()
	auth := &authTiruan{}
	svc := NewInviteService(store, auth, nil)
	created := buatUndangan(t, svc, dto.CreateInviteRequest{MaxUses: 5})

	// Majukan jam service melewati masa berlaku.
	svc.clock = func() time.Time { return time.Now().UTC().Add(8 * 24 * time.Hour) }

	_, err := svc.Klaim(context.Background(), created.Code, payloadDaftar("a@contoh.id"))
	if !errors.Is(err, ErrUndanganTidakSah) {
		t.Fatalf("error = %v; ingin ErrUndanganTidakSah", err)
	}
	if auth.dibuat != 0 {
		t.Fatal("akun dibuat padahal undangan kedaluwarsa")
	}
}

// TestUndanganDicabutDitolak.
func TestUndanganDicabutDitolak(t *testing.T) {
	store := newInviteStoreTiruan()
	svc := NewInviteService(store, &authTiruan{}, nil)
	created := buatUndangan(t, svc, dto.CreateInviteRequest{MaxUses: 5})

	if err := svc.Revoke(context.Background(), AdminActor{ID: "admin-1"}, created.ID, "bocor ke publik"); err != nil {
		t.Fatalf("Revoke error: %v", err)
	}

	if _, err := svc.Klaim(context.Background(), created.Code, payloadDaftar("a@contoh.id")); !errors.Is(err, ErrUndanganTidakSah) {
		t.Fatalf("error = %v; ingin ErrUndanganTidakSah", err)
	}
}

// TestUndanganAlamatLainDitolak memakai pesan berbeda karena bisa diperbaiki.
func TestUndanganAlamatLainDitolak(t *testing.T) {
	store := newInviteStoreTiruan()
	auth := &authTiruan{}
	svc := NewInviteService(store, auth, nil)
	created := buatUndangan(t, svc, dto.CreateInviteRequest{MaxUses: 1, Email: "budi@contoh.id"})

	_, err := svc.Klaim(context.Background(), created.Code, payloadDaftar("oranglain@contoh.id"))
	if !errors.Is(err, ErrUndanganAlamatLain) {
		t.Fatalf("error = %v; ingin ErrUndanganAlamatLain", err)
	}
	if auth.dibuat != 0 {
		t.Fatal("akun dibuat untuk alamat yang tidak diundang")
	}

	// Alamat yang benar tetap bisa, walau beda huruf besar/kecil.
	if _, err := svc.Klaim(context.Background(), created.Code, payloadDaftar("BUDI@contoh.id")); err != nil {
		t.Fatalf("alamat yang diundang ditolak: %v", err)
	}
}

// TestKuotaWajibPositif menutup celah "0 = tanpa batas".
func TestKuotaWajibPositif(t *testing.T) {
	svc := NewInviteService(newInviteStoreTiruan(), &authTiruan{}, nil)

	if _, err := svc.Create(context.Background(), AdminActor{ID: "a"}, dto.CreateInviteRequest{MaxUses: 0}); err == nil {
		t.Fatal("kuota 0 diterima; seharusnya ditolak")
	}
	if _, err := svc.Create(context.Background(), AdminActor{ID: "a"}, dto.CreateInviteRequest{MaxUses: -5}); err == nil {
		t.Fatal("kuota negatif diterima")
	}
}

// TestMasaBerlakuWajibDiMasaDepan menutup undangan yang lahir sudah mati.
func TestMasaBerlakuWajibDiMasaDepan(t *testing.T) {
	svc := NewInviteService(newInviteStoreTiruan(), &authTiruan{}, nil)

	_, err := svc.Create(context.Background(), AdminActor{ID: "a"}, dto.CreateInviteRequest{
		MaxUses: 1, ExpiresAt: time.Now().Add(-time.Hour),
	})
	if err == nil {
		t.Fatal("masa berlaku lampau diterima")
	}
}

// TestPeriksaTidakMemakaiKuota: membuka halaman daftar tidak boleh membakar jatah.
func TestPeriksaTidakMemakaiKuota(t *testing.T) {
	store := newInviteStoreTiruan()
	svc := NewInviteService(store, &authTiruan{}, nil)
	created := buatUndangan(t, svc, dto.CreateInviteRequest{MaxUses: 1})

	for i := 0; i < 5; i++ {
		if _, err := svc.Periksa(context.Background(), created.Code); err != nil {
			t.Fatalf("Periksa error: %v", err)
		}
	}

	row, _ := store.FindByHash(context.Background(), HashInviteCode(created.Code))
	if row.UsedCount != 0 {
		t.Fatalf("used_count = %d setelah 5 pemeriksaan; ingin 0", row.UsedCount)
	}
}

// TestPeriksaTidakMembocorkanEmail: halaman publik tidak boleh menampilkan
// alamat surel tujuan.
func TestPeriksaTidakMembocorkanEmail(t *testing.T) {
	svc := NewInviteService(newInviteStoreTiruan(), &authTiruan{}, nil)
	created := buatUndangan(t, svc, dto.CreateInviteRequest{MaxUses: 1, Email: "rahasia@contoh.id"})

	info, err := svc.Periksa(context.Background(), created.Code)
	if err != nil {
		t.Fatalf("Periksa error: %v", err)
	}
	if !info.EmailTerkunci {
		t.Fatal("email_terkunci seharusnya true")
	}
}
