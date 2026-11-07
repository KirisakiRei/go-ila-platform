package service

import (
	"context"
	"strings"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
)

// SessionService mengurus daftar perangkat yang sedang masuk.
type SessionService interface {
	// List menampilkan sesi aktif milik user; currentID menandai perangkat ini.
	List(ctx context.Context, userID, scope, currentID string) ([]dto.SessionResponse, error)
	// Revoke mencabut satu sesi milik user.
	Revoke(ctx context.Context, userID, sessionID string) error
}

type sessionService struct {
	sessions SessionRepository
	revoker  SessionRevoker
	audit    AuditRepository
	now      func() time.Time
}

// SessionRepository adalah kebutuhan service atas penyimpanan sesi.
type SessionRepository interface {
	ListByUser(ctx context.Context, userID, scope string) ([]model.Session, error)
	Revoke(ctx context.Context, id, userID string) error
	RevokeAll(ctx context.Context, userID, scope string) error
	Touch(ctx context.Context, id string, at time.Time) error
}

// SessionRevoker mencabut seluruh token akun. Dipakai saat satu perangkat
// dicabut, karena verifikasi per permintaan memakai session_version.
type SessionRevoker interface {
	RevokeSessions(ctx context.Context, userID string) error
}

// NewSessionService merangkai layanan sesi.
func NewSessionService(sessions SessionRepository, revoker SessionRevoker, audit AuditRepository) SessionService {
	return &sessionService{
		sessions: sessions,
		revoker:  revoker,
		audit:    audit,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// List mengembalikan sesi aktif beserta penanda perangkat saat ini.
func (s *sessionService) List(ctx context.Context, userID, scope, currentID string) ([]dto.SessionResponse, error) {
	sessions, err := s.sessions.ListByUser(ctx, userID, scope)
	if err != nil {
		return nil, err
	}

	out := make([]dto.SessionResponse, 0, len(sessions))
	for i := range sessions {
		item := sessions[i]
		out = append(out, dto.SessionResponse{
			ID:          item.ID,
			DeviceLabel: deviceLabel(item.DeviceLabel, item.UserAgent),
			UserAgent:   item.UserAgent,
			LastSeenAt:  item.LastSeenAt.Format(time.RFC3339),
			LastSeen:    dto.NewTimePair(item.LastSeenAt),
			CreatedAt:   item.CreatedAt.Format(time.RFC3339),
			Created:     dto.NewTimePair(item.CreatedAt),
			ExpiresAt:   item.ExpiresAt.Format(time.RFC3339),
			Expires:     dto.NewTimePair(item.ExpiresAt),
			Current:     item.ID == currentID,
		})
	}
	return out, nil
}

// Revoke mencabut satu sesi. Sesi milik orang lain tidak bisa disentuh.
//
// Token perangkat lain tidak bisa dibatalkan satu per satu karena verifikasi
// memakai session_version; karena itu pencabutan menaikkan versi sehingga
// seluruh token lama ditolak. Perangkat ini akan otomatis masuk ulang.
func (s *sessionService) Revoke(ctx context.Context, userID, sessionID string) error {
	if err := s.sessions.Revoke(ctx, sessionID, userID); err != nil {
		return err
	}

	if s.revoker != nil {
		if err := s.revoker.RevokeSessions(ctx, userID); err != nil {
			return err
		}
	}

	s.recordAudit(ctx, userID, "auth.session.revoked", sessionID)
	return nil
}

func (s *sessionService) recordAudit(ctx context.Context, userID, event, target string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, &model.AuditLog{
		UserID:    userID,
		Event:     event,
		Entity:    "session",
		EntityID:  target,
		CreatedAt: s.now(),
	})
}

// deviceLabel menyusun sebutan ramah dari user agent bila belum ada.
func deviceLabel(stored, userAgent string) string {
	if strings.TrimSpace(stored) != "" {
		return stored
	}
	return detectDevice(userAgent)
}

// detectDevice mengenali peramban dan sistem operasi tanpa pustaka tambahan.
// Hasilnya perkiraan; tujuannya membantu user mengenali perangkatnya sendiri.
func detectDevice(userAgent string) string {
	ua := strings.ToLower(userAgent)
	if ua == "" {
		return "Perangkat tidak dikenal"
	}

	browser := "Peramban lain"
	switch {
	case strings.Contains(ua, "edg/"):
		browser = "Edge"
	case strings.Contains(ua, "chrome/") && !strings.Contains(ua, "chromium"):
		browser = "Chrome"
	case strings.Contains(ua, "firefox/"):
		browser = "Firefox"
	case strings.Contains(ua, "safari/") && !strings.Contains(ua, "chrome"):
		browser = "Safari"
	}

	platform := "perangkat lain"
	switch {
	case strings.Contains(ua, "windows"):
		platform = "Windows"
	case strings.Contains(ua, "android"):
		platform = "Android"
	case strings.Contains(ua, "iphone"), strings.Contains(ua, "ipad"):
		platform = "iOS"
	case strings.Contains(ua, "mac os x"):
		platform = "macOS"
	case strings.Contains(ua, "linux"):
		platform = "Linux"
	}

	return browser + " di " + platform
}
