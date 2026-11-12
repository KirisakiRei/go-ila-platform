package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// Panjang dan umur bawaan undangan.
const (
	// InviteMasaBerlakuBawaan dipakai bila admin tidak mengisi tanggal.
	InviteMasaBerlakuBawaan = 7 * 24 * time.Hour
	// InviteMasaBerlakuMaksimum menahan salah ketik tahun.
	InviteMasaBerlakuMaksimum = 365 * 24 * time.Hour
	// inviteByteKode: 32 byte acak = 256 bit, jauh di atas tebakan.
	inviteByteKode = 32
)

// ErrUndanganTidakSah adalah penolakan tunggal untuk undangan yang tidak bisa
// dipakai, APA PUN sebabnya (tidak ada, dicabut, kedaluwarsa, kuota habis).
//
// Satu pesan untuk semua sebab dengan sengaja: bila berbeda, orang bisa
// menebak-nebak kode sambil membaca mana yang "hampir benar".
var ErrUndanganTidakSah = errors.Join(
	apperror.ErrForbidden,
	errors.New("Undangan tidak berlaku. Minta tautan baru ke admin."),
)

// ErrUndanganAlamatLain dipisah karena bisa diperbaiki pengguna.
var ErrUndanganAlamatLain = errors.Join(
	apperror.ErrForbidden,
	errors.New("Undangan ini ditujukan untuk alamat surel lain."),
)

// InviteStore adalah kebutuhan penyimpanan undangan.
type InviteStore interface {
	Create(ctx context.Context, invite *model.InviteCode) error
	FindByHash(ctx context.Context, hash string) (*model.InviteCode, error)
	FindByID(ctx context.Context, id string) (*model.InviteCode, error)
	List(ctx context.Context, status string, limit int) ([]model.InviteCode, error)
	Revoke(ctx context.Context, id string, now time.Time) error
	KlaimKuota(ctx context.Context, id string, now time.Time) error
	KembalikanKuota(ctx context.Context, id string) error
	CatatPemakaian(ctx context.Context, redemption *model.InviteRedemption) error
	ListPemakaian(ctx context.Context, inviteID string) ([]model.InviteRedemption, error)
}

// InviteService menangani pemeriksaan dan pemakaian undangan oleh pengguna.
type InviteService interface {
	// Periksa hanya membaca; tidak memakai kuota.
	Periksa(ctx context.Context, code string) (*dto.InvitePublic, error)
	// Klaim memakai kuota lalu membuat akun.
	Klaim(ctx context.Context, code string, req dto.RegisterRequest) (*dto.UserResponse, error)
}

// AdminInviteService menangani pembuatan dan pencabutan undangan.
type AdminInviteService interface {
	Create(ctx context.Context, actor AdminActor, req dto.CreateInviteRequest) (*dto.CreatedInvite, error)
	List(ctx context.Context, status string) ([]dto.InviteItem, error)
	Revoke(ctx context.Context, actor AdminActor, inviteID, reason string) error
}

type inviteService struct {
	store InviteStore
	auth  AuthService
	audit AdminAudit
	clock func() time.Time
}

// NewInviteService merangkai layanan undangan.
func NewInviteService(store InviteStore, auth AuthService, recorder AdminAudit) *inviteService {
	return &inviteService{
		store: store,
		auth:  auth,
		audit: recorder,
		clock: func() time.Time { return time.Now().UTC() },
	}
}

// ---------------------------------------------------------------------------
// Pembuatan kode
// ---------------------------------------------------------------------------

// inviteEncoding memakai base32 tanpa padding. Base32 dipilih karena hanya
// memuat huruf besar dan angka, sehingga tautan tidak rusak saat disalin dari
// aplikasi chat yang mengubah huruf atau memotong di tanda baca.
var inviteEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// HashInviteCode mengembalikan SHA-256 heksadesimal dari kode yang sudah
// dinormalkan. Kode mentah tidak pernah disimpan.
func HashInviteCode(code string) string {
	sum := sha256.Sum256([]byte(normalizeInviteCode(code)))
	return hex.EncodeToString(sum[:])
}

// normalizeInviteCode membuang spasi dan menyamakan huruf besar/kecil, supaya
// kode yang diketik ulang dengan salah spasi tetap diterima.
func normalizeInviteCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
}

// generateInviteCode menghasilkan kode acak kriptografis.
func generateInviteCode() (string, error) {
	buf := make([]byte, inviteByteKode)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("membuat kode undangan: %w", err)
	}
	return inviteEncoding.EncodeToString(buf), nil
}

// PrefixUndangan mengambil empat karakter awal untuk ditampilkan di daftar.
func PrefixUndangan(code string) string {
	if len(code) < 4 {
		return code
	}
	return code[:4]
}

// ---------------------------------------------------------------------------
// Admin
// ---------------------------------------------------------------------------

// Create membuat undangan baru dan mengembalikan kode mentah SEKALI.
func (s *inviteService) Create(ctx context.Context, actor AdminActor, req dto.CreateInviteRequest) (*dto.CreatedInvite, error) {
	tier := strings.TrimSpace(req.Tier)
	if tier == "" {
		tier = model.TierPlus
	}
	if !tierDikenal(tier) {
		return nil, validation("tier tidak dikenal")
	}

	// Kuota wajib >= 1. Tidak ada nilai yang berarti "tanpa batas": kuota adalah
	// satu-satunya batas nyata selain masa berlaku.
	if req.MaxUses < 1 {
		return nil, validation("kuota minimal 1; tautan tanpa batas tidak diizinkan")
	}
	if req.MaxUses > 10000 {
		return nil, validation("kuota maksimal 10000")
	}

	now := s.clock()
	expires := req.ExpiresAt
	if expires.IsZero() {
		expires = now.Add(InviteMasaBerlakuBawaan)
	}
	if !expires.After(now) {
		return nil, validation("masa berlaku harus di masa depan")
	}
	if expires.After(now.Add(InviteMasaBerlakuMaksimum)) {
		return nil, validation("masa berlaku maksimal 365 hari")
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email != "" && !strings.Contains(email, "@") {
		return nil, validation("alamat surel tidak sah")
	}

	code, err := generateInviteCode()
	if err != nil {
		return nil, err
	}

	invite := &model.InviteCode{
		CodeHash:    HashInviteCode(code),
		Prefix:      PrefixUndangan(code),
		EmailTujuan: email,
		Tier:        tier,
		MaxUses:     req.MaxUses,
		ExpiresAt:   expires,
		IsActive:    true,
		Note:        strings.TrimSpace(req.Note),
		CreatedBy:   actor.ID,
	}
	if err := s.store.Create(ctx, invite); err != nil {
		return nil, err
	}

	// Audit tidak menyimpan kode mentah: jejaknya cukup dari id undangan.
	if s.audit != nil {
		if err := s.audit.Record(ctx, AuditEntry{
			ActorID: actor.ID, Action: audit.InviteCreate,
			TargetType: "invite", TargetID: invite.ID,
			RequestID: actor.RequestID,
			After:     map[string]any{"tier": tier, "max_uses": req.MaxUses, "email": email},
		}); err != nil {
			return nil, err
		}
	}

	return &dto.CreatedInvite{
		InviteItem: toInviteItem(*invite, now),
		// Kode mentah hanya ada di balasan ini. Tidak bisa dilihat lagi.
		Code: code,
		URL:  "/daftar?invite=" + code,
	}, nil
}

// List membaca daftar undangan untuk admin, tanpa kode mentah.
func (s *inviteService) List(ctx context.Context, status string) ([]dto.InviteItem, error) {
	rows, err := s.store.List(ctx, status, 100)
	if err != nil {
		return nil, err
	}
	now := s.clock()
	out := make([]dto.InviteItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, toInviteItem(row, now))
	}
	return out, nil
}

// Revoke mencabut undangan yang belum atau sudah terpakai sebagian.
func (s *inviteService) Revoke(ctx context.Context, actor AdminActor, inviteID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return validation("alasan pencabutan wajib diisi")
	}
	if err := s.store.Revoke(ctx, inviteID, s.clock()); err != nil {
		return err
	}
	if s.audit != nil {
		if err := s.audit.Record(ctx, AuditEntry{
			ActorID: actor.ID, Action: audit.InviteRevoke,
			TargetType: "invite", TargetID: inviteID,
			RequestID: actor.RequestID,
			After:     map[string]any{"reason": reason},
		}); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Pengguna
// ---------------------------------------------------------------------------

// Periksa membaca keadaan undangan tanpa memakai kuota.
//
// Dipakai halaman daftar untuk memutuskan apakah formulir ditampilkan. Hasilnya
// TIDAK boleh dipercaya saat submit: Klaim memeriksa ulang semuanya.
func (s *inviteService) Periksa(ctx context.Context, code string) (*dto.InvitePublic, error) {
	if strings.TrimSpace(code) == "" {
		return nil, ErrUndanganTidakSah
	}

	invite, err := s.store.FindByHash(ctx, HashInviteCode(code))
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, ErrUndanganTidakSah
		}
		return nil, err
	}

	now := s.clock()
	if !invite.DapatDipakai(now) {
		return nil, ErrUndanganTidakSah
	}

	// Email tujuan tidak pernah dikirim ke publik: tautan bisa disebar, dan
	// menampilkan alamat orang lain di halaman publik membocorkan data pribadi.
	return &dto.InvitePublic{
		Valid:         true,
		Tier:          invite.Tier,
		ExpiresAt:     invite.ExpiresAt,
		SisaKuota:     invite.SisaKuota(),
		EmailTerkunci: invite.EmailTujuan != "",
	}, nil
}

// Klaim memakai satu kuota lalu membuat akun.
//
// Urutannya penting dan disengaja:
//  1. Periksa undangan sekali lagi (bukan percaya hasil Periksa halaman).
//  2. Potong kuota dengan UPDATE bersyarat — atomik di database.
//  3. Baru buat akun lewat jalur registrasi yang sama dengan pendaftaran biasa.
//  4. Bila pembuatan akun gagal, kuota dikembalikan.
//
// Kuota dipotong lebih dulu supaya dua orang yang mengklaim slot terakhir
// bersamaan tidak sama-sama berhasil. Kegagalan di langkah 3 tidak boleh
// menghabiskan jatah orang lain, jadi langkah 4 wajib.
func (s *inviteService) Klaim(ctx context.Context, code string, req dto.RegisterRequest) (*dto.UserResponse, error) {
	if strings.TrimSpace(code) == "" {
		return nil, ErrUndanganTidakSah
	}

	invite, err := s.store.FindByHash(ctx, HashInviteCode(code))
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, ErrUndanganTidakSah
		}
		return nil, err
	}

	now := s.clock()
	if !invite.DapatDipakai(now) {
		return nil, ErrUndanganTidakSah
	}

	// Undangan terikat alamat: hanya alamat itu yang boleh memakainya.
	if invite.EmailTujuan != "" && !strings.EqualFold(invite.EmailTujuan, strings.TrimSpace(req.Email)) {
		return nil, ErrUndanganAlamatLain
	}

	// Potong kuota. Gagal di sini berarti undangan habis/dicabut/kedaluwarsa,
	// termasuk bila keadaan berubah sejak Periksa di halaman.
	if err := s.store.KlaimKuota(ctx, invite.ID, now); err != nil {
		if errors.Is(err, apperror.ErrConflict) {
			return nil, ErrUndanganTidakSah
		}
		return nil, err
	}

	user, regErr := s.auth.Register(ctx, req)
	if regErr != nil {
		// Pendaftaran tidak jadi: kuota dikembalikan agar tidak terbuang.
		// Kegagalan pengembalian tidak boleh menutupi galat aslinya, tetapi
		// dicatat lewat pembungkus agar terlihat di log.
		if undoErr := s.store.KembalikanKuota(ctx, invite.ID); undoErr != nil {
			return nil, errors.Join(regErr, fmt.Errorf("mengembalikan kuota: %w", undoErr))
		}
		return nil, regErr
	}

	// Tingkat dan verifikasi surel ditetapkan di sini, bukan saat registrasi
	// biasa. Memegang tautan undangan membuktikan kepemilikan alamat tujuannya,
	// jadi verifikasi surel tidak perlu diminta lagi.
	if err := s.terapkanHakUndangan(ctx, user.ID, invite); err != nil {
		return nil, err
	}

	// Hak undangan sudah tertulis di database. Salinan yang dikembalikan masih
	// membawa tier lama (free) dari saat akun dibuat, jadi tier di sini harus
	// ikut diperbarui: balasan ini yang dibaca klien, dan free akan menyesatkan
	// pengguna yang baru saja masuk lewat undangan plus.
	user.Tier = invite.Tier

	if s.audit != nil {
		_ = s.audit.Record(ctx, AuditEntry{
			ActorID: user.ID, Action: audit.InviteRedeem,
			TargetType: "invite", TargetID: invite.ID,
			After: map[string]any{"user_id": user.ID, "tier": invite.Tier},
		})
	}

	return user, nil
}

// terapkanHakUndangan menulis tier, verifikasi surel, dan catatan pemakaian.
func (s *inviteService) terapkanHakUndangan(ctx context.Context, userID string, invite *model.InviteCode) error {
	if updater, ok := s.auth.(interface {
		TerapkanUndangan(ctx context.Context, userID, tier string, at time.Time) error
	}); ok {
		if err := updater.TerapkanUndangan(ctx, userID, invite.Tier, s.clock()); err != nil {
			return err
		}
	}

	// Catatan pemakaian gagal tidak membatalkan akun yang sudah jadi; tetapi
	// harus terlihat, karena tanpa baris ini admin kehilangan jejak asal akun.
	_ = s.store.CatatPemakaian(ctx, &model.InviteRedemption{
		InviteID: invite.ID,
		UserID:   userID,
		Email:    invite.EmailTujuan,
		UsedAt:   s.clock(),
	})
	return nil
}

// ---------------------------------------------------------------------------
// Pembantu
// ---------------------------------------------------------------------------

func tierDikenal(tier string) bool {
	switch tier {
	case model.TierFree, model.TierBasic, model.TierPlus:
		return true
	}
	return false
}

func toInviteItem(invite model.InviteCode, now time.Time) dto.InviteItem {
	return dto.InviteItem{
		ID:        invite.ID,
		Prefix:    invite.Prefix,
		Email:     invite.EmailTujuan,
		Tier:      invite.Tier,
		MaxUses:   invite.MaxUses,
		UsedCount: invite.UsedCount,
		SisaKuota: invite.SisaKuota(),
		ExpiresAt: invite.ExpiresAt,
		Status:    invite.Status(now),
		Note:      invite.Note,
		CreatedAt: invite.CreatedAt,
	}
}

// pastikan inviteService memenuhi kedua antarmuka.
var (
	_ InviteService      = (*inviteService)(nil)
	_ AdminInviteService = (*inviteService)(nil)
)
