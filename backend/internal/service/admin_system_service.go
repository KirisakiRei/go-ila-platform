package service

import (
	"context"
	"errors"
	"strings"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/settings"
	"ila/backend/pkg/apperror"
)

// AdminSystemService mengatur flag operasional dari panel admin.
//
// Tujuannya memberi rem darurat tanpa SSH: mematikan eksekusi, memaksa
// simulasi, atau menurunkan batas harian saat pasar berjalan aneh
// (plans/11 Fase I).
type AdminSystemService interface {
	Settings(ctx context.Context) ([]settings.Snapshot, error)
	UpdateSetting(ctx context.Context, actor AdminActor, key string, req dto.SystemSettingRequest) (*settings.Snapshot, error)
	ResetSetting(ctx context.Context, actor AdminActor, key string, req dto.SystemSettingRequest) error
}

type adminSystemService struct {
	flags *settings.Service
	store SettingsStore
	audit AdminAudit
}

// SettingsStore adalah kebutuhan penyimpanan override milik service.
type SettingsStore interface {
	Rows(ctx context.Context) ([]model.SystemSetting, error)
	Delete(ctx context.Context, key string) error
}

// NewAdminSystemService merangkai pengelola flag sistem.
func NewAdminSystemService(flags *settings.Service, store SettingsStore, recorder AdminAudit) AdminSystemService {
	return &adminSystemService{flags: flags, store: store, audit: recorder}
}

func (s *adminSystemService) Settings(ctx context.Context) ([]settings.Snapshot, error) {
	return s.flags.Snapshot(ctx), nil
}

func (s *adminSystemService) UpdateSetting(ctx context.Context, actor AdminActor, key string, req dto.SystemSettingRequest) (*settings.Snapshot, error) {
	key = strings.TrimSpace(key)
	if err := s.confirm(key, req); err != nil {
		return nil, err
	}

	before := s.flags.Value(ctx, key)
	if err := settings.Validate(key, req.Value); err != nil {
		return nil, validation(settingMessage(err))
	}

	if err := s.flags.Write(ctx, key, strings.TrimSpace(req.Value), strings.TrimSpace(req.Reason), actor.ID); err != nil {
		return nil, err
	}
	s.flags.Invalidate()

	if err := s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.SystemFlagChange,
		TargetType: "system_setting", TargetID: key,
		Reason: req.Reason, RequestID: actor.RequestID,
		Before: map[string]any{"key": key, "value": before},
		After:  map[string]any{"key": key, "value": strings.TrimSpace(req.Value)},
	}); err != nil {
		return nil, err
	}

	for _, item := range s.flags.Snapshot(ctx) {
		if item.Key == key {
			return &item, nil
		}
	}
	return nil, apperror.ErrNotFound
}

// ResetSetting menghapus override sehingga nilai environment berlaku lagi.
func (s *adminSystemService) ResetSetting(ctx context.Context, actor AdminActor, key string, req dto.SystemSettingRequest) error {
	key = strings.TrimSpace(key)
	if _, ok := settings.Lookup(key); !ok {
		return apperror.ErrNotFound
	}
	if err := s.confirm(key, req); err != nil {
		return err
	}
	if err := ValidateReason(audit.SystemFlagChange, req.Reason); err != nil {
		return err
	}

	before := s.flags.Value(ctx, key)
	if err := s.store.Delete(ctx, key); err != nil {
		return err
	}
	s.flags.Invalidate()

	after := s.flags.Value(ctx, key)
	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.SystemFlagChange,
		TargetType: "system_setting", TargetID: key,
		Reason: req.Reason, RequestID: actor.RequestID,
		Before: map[string]any{"key": key, "value": before, "source": "database"},
		After:  map[string]any{"key": key, "value": after, "source": "env"},
	})
}

// confirm mewajibkan pelaku mengetik ulang nama flag, sehingga flag berbahaya
// tidak berubah karena salah klik (plans/11 I4).
func (s *adminSystemService) confirm(key string, req dto.SystemSettingRequest) error {
	_, ok := settings.Lookup(key)
	if !ok {
		return apperror.ErrNotFound
	}
	if strings.TrimSpace(req.Confirm) != key {
		return validation("ketik ulang nama flag untuk konfirmasi")
	}
	// Aksi ini selalu berdampak ke jalur trading, jadi alasan wajib.
	if strings.TrimSpace(req.Reason) == "" {
		return validation("alasan perubahan wajib diisi")
	}
	return nil
}

// settingMessage menerjemahkan kesalahan validasi flag ke bahasa pengguna.
func settingMessage(err error) string {
	switch {
	case errors.Is(err, settings.ErrUnknownKey):
		return "flag tidak dikenal"
	case errors.Is(err, settings.ErrEmptyValue):
		return "nilai tidak boleh kosong"
	case errors.Is(err, settings.ErrOutOfRange):
		return "nilai di luar rentang yang diizinkan"
	default:
		return "nilai tidak sah untuk flag ini"
	}
}
