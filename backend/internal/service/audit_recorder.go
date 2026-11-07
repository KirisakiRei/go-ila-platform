package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"ila/backend/internal/audit"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// MinReasonLength adalah panjang alasan minimal untuk aksi destruktif.
// Alasan ditampilkan ke pengguna, jadi harus bisa dibaca sebagai kalimat.
const MinReasonLength = 10

// AdminAudit mencatat aksi admin pada bentuk baris standar.
// Seluruh aksi admin lewat sini agar bentuknya seragam dan rahasia tersaring.
type AdminAudit interface {
	Record(ctx context.Context, entry AuditEntry) error
}

// AuditEntry adalah masukan satu baris audit.
type AuditEntry struct {
	ActorID    string
	ActorType  string
	Action     audit.Action
	TargetType string
	TargetID   string
	Reason     string
	Before     map[string]any
	After      map[string]any
	RequestID  string
	// Force menandai aksi sensitif yang wajib tercatat walau alasan kosong.
	Force bool
}

// recorderWriter adalah penyimpanan baris audit.
type recorderWriter interface {
	Record(ctx context.Context, entry *model.AuditLog) error
}

type auditRecorder struct {
	store recorderWriter
	now   func() time.Time
}

// NewAuditRecorder merangkai perekam audit.
func NewAuditRecorder(store recorderWriter) AdminAudit {
	return &auditRecorder{store: store, now: func() time.Time { return time.Now().UTC() }}
}

// ValidateReason memastikan alasan cukup panjang untuk aksi destruktif.
func ValidateReason(action audit.Action, reason string) error {
	trimmed := strings.TrimSpace(reason)
	if !audit.Destructive(action) {
		return nil
	}
	if len([]rune(trimmed)) < MinReasonLength {
		return validation("alasan wajib diisi minimal 10 karakter")
	}
	return nil
}

// Record menulis satu baris audit. Kegagalan menulis dikembalikan agar
// pemanggil membatalkan transaksi: aksi tanpa jejak tidak boleh terjadi.
func (r *auditRecorder) Record(ctx context.Context, entry AuditEntry) error {
	if entry.Action == "" {
		return errors.Join(apperror.ErrBadRequest, errors.New("aksi audit wajib diisi"))
	}
	if err := ValidateReason(entry.Action, entry.Reason); err != nil {
		return err
	}

	actorType := entry.ActorType
	if actorType == "" {
		actorType = "admin"
	}

	row := &model.AuditLog{
		UserID:      entry.ActorID,
		Event:       string(entry.Action),
		Entity:      entry.TargetType,
		EntityID:    entry.TargetID,
		ActorID:     entry.ActorID,
		ActorType:   actorType,
		TargetType:  entry.TargetType,
		TargetID:    entry.TargetID,
		Action:      string(entry.Action),
		Reason:      strings.TrimSpace(entry.Reason),
		BeforeState: audit.Scrub(entry.Before),
		AfterState:  audit.Scrub(entry.After),
		RequestID:   entry.RequestID,
		CreatedAt:   r.now(),
	}
	if err := r.store.Record(ctx, row); err != nil {
		return err
	}
	return nil
}
