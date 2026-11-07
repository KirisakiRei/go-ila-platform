package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/model"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/apperror"
)

// ApprovalTokenTTL adalah masa berlaku token deep-link konfirmasi entry.
// Lebih pendek daripada TTL sinyal agar tidak ada tautan basi yang tersisa.
const ApprovalTokenTTL = 5 * time.Minute

// ApproveResult adalah jawaban halaman konfirmasi setelah tombol ditekan.
type ApproveResult struct {
	SignalID string `json:"signal_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

// ApproveService memvalidasi token konfirmasi lalu meneruskan eksekusi.
//
// Seluruh syarat trading diperiksa ulang di Runner, bukan di sini: keadaan
// bisa berubah antara halaman dirender dan tombol ditekan (plans/13 §4.6).
type ApproveService interface {
	// Resolve menampilkan detail sinyal untuk halaman konfirmasi.
	Resolve(ctx context.Context, userID, rawToken string) (*ApprovePreview, error)
	// Confirm memakai token lalu meminta Runner mengeksekusi.
	Confirm(ctx context.Context, userID, rawToken string) (*ApproveResult, error)
	// Decline menandai sinyal dilewati user.
	Decline(ctx context.Context, userID, rawToken string) (*ApproveResult, error)
}

// ApprovePreview adalah ringkasan yang ditampilkan sebelum konfirmasi.
type ApprovePreview struct {
	SignalID        string    `json:"signal_id"`
	Symbol          string    `json:"symbol"`
	Direction       string    `json:"direction"`
	StrategyVersion string    `json:"strategy_version"`
	EntryPrice      float64   `json:"entry_price"`
	StopLoss        float64   `json:"stop_loss"`
	TakeProfit      float64   `json:"take_profit"`
	RiskReward      float64   `json:"risk_reward"`
	ExpiresAt       time.Time `json:"expires_at"`
	BrokerAccountID string    `json:"broker_account_id"`
	CanConfirm      bool      `json:"can_confirm"`
	BlockedReason   string    `json:"blocked_reason,omitempty"`
}

// ApprovalRunner adalah operasi Runner yang dipanggil backend-api.
type ApprovalRunner interface {
	Approve(ctx context.Context, userID, signalID, accountID string) (string, error)
	Skip(ctx context.Context, userID, signalID string) error
}

// ApprovalTokenRepository mengelola token konfirmasi.
type ApprovalTokenStore interface {
	FindByHash(ctx context.Context, hash string) (*model.ApprovalToken, error)
	MarkUsed(ctx context.Context, id string, at time.Time) error
}

// UserSignalStore membaca status sinyal per user.
type UserSignalStore interface {
	FindUserSignal(ctx context.Context, userID, signalID string) (*model.UserSignal, error)
}

// SignalStore membaca sinyal.
type SignalStore interface {
	FindSignal(ctx context.Context, signalID string) (*model.Signal, error)
}

// AuditRecorder mencatat keputusan user.
type AuditRecorder interface {
	Record(ctx context.Context, entry *model.AuditLog) error
}

type approveService struct {
	tokens  ApprovalTokenStore
	users   UserSignalStore
	signals SignalStore
	audit   AuditRecorder
	runner  ApprovalRunner
	clock   func() time.Time
}

// NewApproveService merangkai layanan konfirmasi entry.
func NewApproveService(
	tokens ApprovalTokenStore,
	users UserSignalStore,
	signals SignalStore,
	audit AuditRecorder,
	runner ApprovalRunner,
) ApproveService {
	return &approveService{
		tokens:  tokens,
		users:   users,
		signals: signals,
		audit:   audit,
		runner:  runner,
		clock:   func() time.Time { return time.Now().UTC() },
	}
}

// HashApprovalToken menghasilkan hash yang disimpan untuk verifikasi.
func HashApprovalToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *approveService) Resolve(ctx context.Context, userID, rawToken string) (*ApprovePreview, error) {
	_, signal, err := s.validate(ctx, userID, rawToken)
	if err != nil {
		return nil, err
	}

	preview := &ApprovePreview{
		SignalID:        signal.SignalID,
		Symbol:          signal.Symbol,
		Direction:       signal.Direction,
		StrategyVersion: signal.StrategyVersion,
		EntryPrice:      signal.EntryPrice,
		StopLoss:        signal.StopLoss,
		TakeProfit:      signal.TakeProfit,
		RiskReward:      signal.RiskReward,
		ExpiresAt:       signal.ExpiresAt,
		BrokerAccountID: tokenBrokerAccount(ctx, s.users, userID, signal.SignalID),
	}

	// Alasan tombol nonaktif selalu ditampilkan, bukan dibiarkan kosong.
	if reason := s.blockedReason(ctx, userID, signal); reason != "" {
		preview.CanConfirm = false
		preview.BlockedReason = reason
	} else {
		preview.CanConfirm = true
	}

	return preview, nil
}

func (s *approveService) Confirm(ctx context.Context, userID, rawToken string) (*ApproveResult, error) {
	token, signal, err := s.validate(ctx, userID, rawToken)
	if err != nil {
		return nil, err
	}

	if reason := s.blockedReason(ctx, userID, signal); reason != "" {
		return nil, errors.Join(apperror.ErrConflict, errors.New(reason))
	}

	accountID := tokenBrokerAccount(ctx, s.users, userID, signal.SignalID)
	if accountID == "" {
		return nil, errors.Join(apperror.ErrConflict, errors.New("Akun broker belum terhubung atau kredensial tidak valid."))
	}

	// Token dipakai lebih dulu: bila eksekusi gagal, tautan tetap tidak bisa
	// dipakai ulang. Kegagalan dicatat, bukan diulang otomatis.
	if err := s.tokens.MarkUsed(ctx, token.ID, s.clock()); err != nil {
		return nil, err
	}

	reason, runErr := s.runner.Approve(ctx, userID, signal.SignalID, accountID)
	if runErr != nil {
		return nil, runErr
	}

	s.recordAudit(ctx, userID, signal.SignalID, "signal.approve", reason)

	if reason != "" {
		return &ApproveResult{
			SignalID: signal.SignalID,
			Status:   model.SignalStatusRejected,
			Message:  reason,
		}, nil
	}

	return &ApproveResult{
		SignalID: signal.SignalID,
		Status:   model.SignalStatusConsumed,
		Message:  "Eksekusi terkirim. Status akhir mengikuti broker.",
	}, nil
}

func (s *approveService) Decline(ctx context.Context, userID, rawToken string) (*ApproveResult, error) {
	token, signal, err := s.validate(ctx, userID, rawToken)
	if err != nil {
		return nil, err
	}

	if err := s.tokens.MarkUsed(ctx, token.ID, s.clock()); err != nil {
		return nil, err
	}

	if err := s.runner.Skip(ctx, userID, signal.SignalID); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, userID, signal.SignalID, "signal.skip", "dilewati user")

	return &ApproveResult{
		SignalID: signal.SignalID,
		Status:   model.SignalStatusSkipped,
		Message:  "Sinyal dilewati. Kuota harian tidak terpakai.",
	}, nil
}

// validate memastikan token sah, milik user, dan belum kedaluwarsa.
func (s *approveService) validate(ctx context.Context, userID, rawToken string) (*model.ApprovalToken, *model.Signal, error) {
	if rawToken == "" {
		return nil, nil, apperror.ErrBadRequest
	}

	token, err := s.tokens.FindByHash(ctx, HashApprovalToken(rawToken))
	if err != nil {
		return nil, nil, err
	}

	// Token terikat ke user: token sah milik orang lain tidak boleh dipakai.
	if token.UserID != userID {
		return nil, nil, apperror.ErrForbidden
	}
	if token.UsedAt != nil {
		return nil, nil, errors.Join(apperror.ErrConflict, errors.New("Tautan konfirmasi sudah dipakai."))
	}
	if s.clock().After(token.ExpiresAt) {
		return nil, nil, errors.Join(apperror.ErrConflict, errors.New("Tautan konfirmasi kedaluwarsa. Buka dashboard untuk melihat sinyal."))
	}

	signal, err := s.signals.FindSignal(ctx, token.SignalID)
	if err != nil {
		return nil, nil, err
	}

	return token, signal, nil
}

// blockedReason mengembalikan alasan tombol konfirmasi harus nonaktif.
func (s *approveService) blockedReason(ctx context.Context, userID string, signal *model.Signal) string {
	if !s.clock().Before(signal.ExpiresAt) {
		return "Sinyal sudah kedaluwarsa (batas waktu habis)."
	}

	switch signal.Status {
	case model.SignalStatusCancelled:
		return "Setup ini tidak lagi valid sebelum entry."
	case model.SignalStatusSuperseded:
		return "Ada setup yang lebih baru untuk pair ini."
	case model.SignalStatusConsumed:
		return "Sinyal ini sudah dieksekusi sebelumnya."
	case model.SignalStatusExpired:
		return "Sinyal sudah kedaluwarsa (batas waktu habis)."
	}

	us, err := s.users.FindUserSignal(ctx, userID, signal.SignalID)
	if err != nil {
		return ""
	}

	switch us.Status {
	case model.SignalStatusConsumed:
		return "Sinyal ini sudah dieksekusi sebelumnya."
	case model.SignalStatusSkipped:
		return "Sinyal ini sudah Anda lewati."
	case model.SignalStatusRejected:
		if us.RejectReason != "" {
			return us.RejectReason
		}
		return "Sinyal ini tertahan aturan harian."
	case model.SignalStatusCancelled, model.SignalStatusSuperseded, model.SignalStatusExpired:
		return "Setup ini tidak lagi berlaku."
	}

	return ""
}

func (s *approveService) recordAudit(ctx context.Context, userID, signalID, event, note string) {
	entry := &model.AuditLog{
		UserID:   userID,
		Event:    event,
		Entity:   "signal",
		EntityID: signalID,
		Metadata: fmt.Sprintf(`{"note":%q}`, note),
	}
	// Kegagalan audit tidak membatalkan aksi user, tetapi tidak boleh disembunyikan.
	_ = s.audit.Record(ctx, entry)
}

func tokenBrokerAccount(ctx context.Context, users UserSignalStore, userID, signalID string) string {
	us, err := users.FindUserSignal(ctx, userID, signalID)
	if err != nil {
		return ""
	}
	return us.BrokerAccountID
}
