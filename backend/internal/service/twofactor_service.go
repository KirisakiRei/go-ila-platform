package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/totp"
	"ila/backend/pkg/apperror"
)

// Batas dan umur sesuai plans/05.
const (
	// MaxChallengeAttempts adalah jumlah kode salah sebelum tantangan hangus.
	MaxChallengeAttempts = 5
	// ChallengeTTL adalah umur tantangan antara password dan kode 2FA.
	ChallengeTTL = 5 * time.Minute
	// SetupTTL adalah umur izin aktivasi 2FA (menunggu scan QR).
	SetupTTL = 10 * time.Minute
	// RecoveryCodeCount adalah jumlah kode pemulihan sekali pakai.
	RecoveryCodeCount = 8
)

// TwoFactorPurpose membedakan pintu user dan pintu admin. Secret disimpan pada
// tabel yang sama; pemisahannya cukup lewat tantangan login.
const (
	PurposeUser  = "user"
	PurposeAdmin = "admin"
)

// SecretSealer mengenkripsi secret TOTP sebelum menyentuh database.
type SecretSealer interface {
	Version() int
	SealString(plaintext, ownerID string) (string, error)
	OpenString(encoded, ownerID string) (string, error)
}

// TwoFactorService mengelola TOTP: aktivasi, verifikasi saat masuk, kode
// pemulihan, dan penonaktifan (plans/05).
type TwoFactorService interface {
	// Status melaporkan keadaan 2FA sebuah akun.
	Status(ctx context.Context, userID string) (*TwoFactorStatus, error)
	// Enabled menandai apakah 2FA aktif dan menuntut kode saat masuk.
	Enabled(ctx context.Context, userID string) (bool, error)
	// Setup membuat secret baru yang belum aktif beserta tautan otentikator.
	Setup(ctx context.Context, userID, account string) (*TwoFactorSetup, error)
	// Activate memverifikasi kode dari aplikasi lalu mengaktifkan 2FA.
	Activate(ctx context.Context, userID, code string) ([]string, error)
	// Verify memeriksa kode saat masuk. Nil yang dikembalikan berarti kode sah.
	Verify(ctx context.Context, userID, code string) error
	// ConsumeRecovery memakai satu kode pemulihan.
	ConsumeRecovery(ctx context.Context, userID, code string) (bool, error)
	// Disable mematikan 2FA setelah kode/–recovery terbukti sah.
	Disable(ctx context.Context, userID, code string) error
	// RegenerateRecovery membuat set kode pemulihan baru.
	RegenerateRecovery(ctx context.Context, userID, code string) ([]string, error)
}

// TwoFactorStatus adalah ringkasan keadaan 2FA untuk ditampilkan.
type TwoFactorStatus struct {
	Enabled           bool
	ActivatedAt       *time.Time
	LastUsedAt        *time.Time
	RecoveryRemaining int64
}

// TwoFactorSetup adalah data untuk menampilkan QR di antarmuka.
type TwoFactorSetup struct {
	Secret          string
	ProvisioningURL string
	ExpiresAt       time.Time
}

type twoFactorService struct {
	repo   TwoFactorRepository
	sealer SecretSealer
	// sessions mencabut seluruh sesi lama saat 2FA berubah. Tanpa ini, token
	// yang dicuri sebelum 2FA dipasang tetap berlaku sampai kedaluwarsa.
	sessions SessionRevoker
	now      func() time.Time
}

// TwoFactorRepository adalah kebutuhan service atas penyimpanan 2FA.
type TwoFactorRepository interface {
	FindByUser(ctx context.Context, userID string) (*model.TwoFactor, error)
	Upsert(ctx context.Context, settings *model.TwoFactor) error
	Deactivate(ctx context.Context, userID string) error
	MarkUsed(ctx context.Context, userID string, step int64, at time.Time) error
	ReplaceRecoveryCodes(ctx context.Context, userID string, hashes []string) error
	ConsumeRecoveryCode(ctx context.Context, userID, hash string, at time.Time) (bool, error)
	CountUnusedRecoveryCodes(ctx context.Context, userID string) (int64, error)
}

// NewTwoFactorService merangkai layanan 2FA.
// revoker boleh nil saat pengujian; bila ada, sesi lama dicabut setiap 2FA
// diaktifkan, dimatikan, atau kode pemulihannya diganti (plans/03 §5).
func NewTwoFactorService(repo TwoFactorRepository, sealer SecretSealer, revoker SessionRevoker) TwoFactorService {
	return &twoFactorService{
		repo:     repo,
		sealer:   sealer,
		sessions: revoker,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// revokeOldSessions mencabut token lama. Kegagalan dicatat lewat error agar
// pemanggil tahu sesi belum benar-benar mati.
func (s *twoFactorService) revokeOldSessions(ctx context.Context, userID string) error {
	if s.sessions == nil {
		return nil
	}
	return s.sessions.RevokeSessions(ctx, userID)
}

// Enabled menandai 2FA yang sudah aktif. Akun tanpa 2FA bukan kesalahan.
func (s *twoFactorService) Enabled(ctx context.Context, userID string) (bool, error) {
	settings, err := s.repo.FindByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return settings.ActivatedAt != nil, nil
}

// Status membaca keadaan 2FA. Akun tanpa 2FA bukan kesalahan.
func (s *twoFactorService) Status(ctx context.Context, userID string) (*TwoFactorStatus, error) {
	settings, err := s.repo.FindByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return &TwoFactorStatus{Enabled: false}, nil
		}
		return nil, err
	}

	if settings.ActivatedAt == nil {
		return &TwoFactorStatus{Enabled: false}, nil
	}

	remaining, err := s.repo.CountUnusedRecoveryCodes(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &TwoFactorStatus{
		Enabled:           true,
		ActivatedAt:       settings.ActivatedAt,
		LastUsedAt:        settings.LastUsedAt,
		RecoveryRemaining: remaining,
	}, nil
}

// Setup membuat secret baru. Secret lama (bila ada) tidak tersentuh sampai
// aktivasi berhasil, sehingga percobaan aktivasi yang gagal tidak mematikan
// 2FA yang sedang berlaku.
func (s *twoFactorService) Setup(ctx context.Context, userID, account string) (*TwoFactorSetup, error) {
	secret, err := totp.NewSecret()
	if err != nil {
		return nil, err
	}

	encrypted, err := s.sealer.SealString(secret, userID)
	if err != nil {
		return nil, err
	}

	now := s.now()
	// ActivatedAt sengaja kosong: secret belum berlaku sampai kode terbukti.
	settings := &model.TwoFactor{
		UserID:        userID,
		SecretEnc:     encrypted,
		SecretVersion: s.sealer.Version(),
		ActivatedAt:   nil,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.repo.Upsert(ctx, settings); err != nil {
		return nil, err
	}

	return &TwoFactorSetup{
		Secret:          secret,
		ProvisioningURL: totp.ProvisioningURL(secret, account, ""),
		ExpiresAt:       now.Add(SetupTTL),
	}, nil
}

// Activate memverifikasi kode dari aplikasi otentikator lalu menyimpan 8 kode
// pemulihan. Kode pemulihan hanya dikembalikan sekali.
func (s *twoFactorService) Activate(ctx context.Context, userID, code string) ([]string, error) {
	settings, err := s.repo.FindByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, validation("mulai penyiapan 2FA terlebih dahulu")
		}
		return nil, err
	}

	secret, err := s.sealer.OpenString(settings.SecretEnc, userID)
	if err != nil {
		return nil, err
	}

	now := s.now()
	ok, err := totp.Validate(secret, code, now)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, validation("kode 2FA tidak sesuai")
	}

	settings.ActivatedAt = &now
	settings.LastUsedStep = totp.Step(now)
	settings.LastUsedAt = &now
	settings.UpdatedAt = now
	settings.SecretVersion = s.sealer.Version()
	if err := s.repo.Upsert(ctx, settings); err != nil {
		return nil, err
	}
	// 2FA baru berlaku: token yang terbit sebelum faktor ini ada tidak boleh
	// tetap sah, termasuk token yang mungkin sudah dicuri.
	if err := s.revokeOldSessions(ctx, userID); err != nil {
		return nil, err
	}

	return s.issueRecoveryCodes(ctx, userID)
}

// Verify memeriksa kode saat masuk, sekaligus menolak kode yang dipakai ulang.
func (s *twoFactorService) Verify(ctx context.Context, userID, code string) error {
	settings, err := s.repo.FindByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return apperror.ErrNotFound
		}
		return err
	}
	if settings.ActivatedAt == nil {
		return apperror.ErrNotFound
	}

	secret, err := s.sealer.OpenString(settings.SecretEnc, userID)
	if err != nil {
		return err
	}

	now := s.now()
	ok, err := totp.Validate(secret, code, now)
	if err != nil {
		return err
	}
	if !ok {
		return validation("kode 2FA tidak sesuai")
	}

	// Kode yang sudah dipakai pada jendela yang sama tidak diterima lagi.
	step := totp.Step(now)
	if settings.LastUsedStep >= step {
		return validation("kode 2FA sudah dipakai; tunggu kode berikutnya")
	}

	return s.repo.MarkUsed(ctx, userID, step, now)
}

// ConsumeRecovery memakai satu kode pemulihan. Kode yang sah langsung hangus.
func (s *twoFactorService) ConsumeRecovery(ctx context.Context, userID, code string) (bool, error) {
	hash := HashRecoveryCode(code)
	if hash == "" {
		return false, nil
	}
	return s.repo.ConsumeRecoveryCode(ctx, userID, hash, s.now())
}

// Disable mematikan 2FA setelah kode (atau kode pemulihan) terbukti sah.
func (s *twoFactorService) Disable(ctx context.Context, userID, code string) error {
	if strings.TrimSpace(code) == "" {
		return validation("masukkan kode 2FA atau kode pemulihan")
	}

	// Coba kode aplikasi lebih dulu, lalu kode pemulihan.
	if err := s.Verify(ctx, userID, code); err != nil {
		if !errors.Is(err, apperror.ErrBadRequest) {
			return err
		}
		used, recErr := s.ConsumeRecovery(ctx, userID, code)
		if recErr != nil {
			return recErr
		}
		if !used {
			return validation("kode 2FA atau kode pemulihan tidak sesuai")
		}
	}

	if err := s.repo.Deactivate(ctx, userID); err != nil {
		return err
	}
	// Mematikan 2FA mengubah syarat masuk, jadi sesi lama dicabut.
	return s.revokeOldSessions(ctx, userID)
}

// RegenerateRecovery mengganti seluruh kode pemulihan dengan set baru.
func (s *twoFactorService) RegenerateRecovery(ctx context.Context, userID, code string) ([]string, error) {
	if strings.TrimSpace(code) == "" {
		return nil, validation("masukkan kode 2FA")
	}
	if err := s.Verify(ctx, userID, code); err != nil {
		return nil, err
	}
	if err := s.revokeOldSessions(ctx, userID); err != nil {
		return nil, err
	}
	return s.issueRecoveryCodes(ctx, userID)
}

// issueRecoveryCodes membuat kode pemulihan baru dan hanya mengembalikan
// bentuk plaintextnya sekali.
func (s *twoFactorService) issueRecoveryCodes(ctx context.Context, userID string) ([]string, error) {
	codes := make([]string, 0, RecoveryCodeCount)
	hashes := make([]string, 0, RecoveryCodeCount)

	for i := 0; i < RecoveryCodeCount; i++ {
		code, err := newRecoveryCode()
		if err != nil {
			return nil, err
		}
		codes = append(codes, code)
		hashes = append(hashes, HashRecoveryCode(code))
	}

	if err := s.repo.ReplaceRecoveryCodes(ctx, userID, hashes); err != nil {
		return nil, err
	}
	return codes, nil
}

// newRandomBytes membuat nilai acak sepanjang n byte.
func newRandomBytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return nil, errors.New("gagal membuat nilai acak")
	}
	return buf, nil
}

// newRecoveryCode membuat kode berbentuk xxxx-xxxx-xxxx dari 8 byte acak
// (13 karakter base32, diambil 12 agar pas tiga kelompok).
func newRecoveryCode() (string, error) {
	buf, err := newRandomBytes(8)
	if err != nil {
		return "", err
	}
	encoded := strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf))
	// Ambil 12 karakter pertama: cukup acak, mudah diketik ulang.
	if len(encoded) < 12 {
		return "", errors.New("kode pemulihan terlalu pendek")
	}
	raw := encoded[:12]
	return raw[0:4] + "-" + raw[4:8] + "-" + raw[8:12], nil
}

// HashRecoveryCode menormalkan lalu meng-hash kode pemulihan. Yang disimpan
// hanya hash, sehingga bocornya database tidak memberi kode yang sah.
func HashRecoveryCode(code string) string {
	normalized := strings.ToLower(strings.TrimSpace(code))
	normalized = strings.ReplaceAll(normalized, "-", "")
	if len(normalized) != 12 {
		return ""
	}
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
