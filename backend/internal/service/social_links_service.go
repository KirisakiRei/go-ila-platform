package service

import (
	"context"
	"net/url"
	"strings"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/settings"
)

// Kunci penyimpanan tautan media sosial publik di tabel system_setting.
// Sengaja tidak masuk settings.Catalog agar tidak muncul di halaman danger-flag.
const (
	KeySocialDiscordURL  = "SOCIAL_DISCORD_URL"
	KeySocialXURL        = "SOCIAL_X_URL"
	KeySocialTelegramURL = "SOCIAL_TELEGRAM_URL"
)

// SocialLinksStore menyimpan dan menghapus override kunci tautan sosmed.
type SocialLinksStore interface {
	Save(ctx context.Context, key, value, reason, actorID string) error
	Delete(ctx context.Context, key string) error
}

// SocialLinksService mengelola tautan Discord, X, dan Telegram dari dashboard admin.
type SocialLinksService interface {
	Get(ctx context.Context) dto.SocialLinks
	Update(ctx context.Context, actor AdminActor, req dto.SocialLinks) (dto.SocialLinks, error)
}

type socialLinksService struct {
	flags *settings.Service
	store SocialLinksStore
	audit AdminAudit
}

// NewSocialLinksService merangkai pengelola tautan media sosial.
func NewSocialLinksService(flags *settings.Service, store SocialLinksStore, recorder AdminAudit) SocialLinksService {
	return &socialLinksService{
		flags: flags,
		store: store,
		audit: recorder,
	}
}

// Get membaca tautan sosmed aktif dari cache pengaturan sistem.
func (s *socialLinksService) Get(ctx context.Context) dto.SocialLinks {
	if s.flags == nil {
		return dto.SocialLinks{}
	}
	return dto.SocialLinks{
		Discord:  s.flags.Value(ctx, KeySocialDiscordURL),
		X:        s.flags.Value(ctx, KeySocialXURL),
		Telegram: s.flags.Value(ctx, KeySocialTelegramURL),
	}
}

// Update memvalidasi dan menyimpan tautan Discord, X, dan Telegram.
// Nilai kosong menghapus tautan sehingga ikon terkait tidak tampil di situs publik.
func (s *socialLinksService) Update(ctx context.Context, actor AdminActor, req dto.SocialLinks) (dto.SocialLinks, error) {
	clean := dto.SocialLinks{
		Discord:  strings.TrimSpace(req.Discord),
		X:        strings.TrimSpace(req.X),
		Telegram: strings.TrimSpace(req.Telegram),
	}

	if err := validateSocialURL("Discord", clean.Discord); err != nil {
		return dto.SocialLinks{}, err
	}
	if err := validateSocialURL("X", clean.X); err != nil {
		return dto.SocialLinks{}, err
	}
	if err := validateSocialURL("Telegram", clean.Telegram); err != nil {
		return dto.SocialLinks{}, err
	}

	before := s.Get(ctx)
	pairs := []struct {
		key   string
		value string
	}{
		{KeySocialDiscordURL, clean.Discord},
		{KeySocialXURL, clean.X},
		{KeySocialTelegramURL, clean.Telegram},
	}
	for _, item := range pairs {
		if item.value == "" {
			if err := s.store.Delete(ctx, item.key); err != nil {
				return dto.SocialLinks{}, err
			}
		} else {
			if err := s.store.Save(ctx, item.key, item.value, "pembaruan tautan media sosial", actor.ID); err != nil {
				return dto.SocialLinks{}, err
			}
		}
	}

	if s.flags != nil {
		s.flags.Invalidate()
	}

	if s.audit != nil {
		if err := s.audit.Record(ctx, AuditEntry{
			ActorID:    actor.ID,
			Action:     audit.SocialLinksUpdate,
			TargetType: "social_links",
			TargetID:   "public",
			RequestID:  actor.RequestID,
			Before:     map[string]any{"discord": before.Discord, "x": before.X, "telegram": before.Telegram},
			After:      map[string]any{"discord": clean.Discord, "x": clean.X, "telegram": clean.Telegram},
		}); err != nil {
			return dto.SocialLinks{}, err
		}
	}

	return clean, nil
}

// validateSocialURL menolak skema non-HTTPS agar tautan berbahaya (mis. javascript:)
// tidak pernah bisa disimpan atau dirender di halaman publik.
func validateSocialURL(label, raw string) error {
	if raw == "" {
		return nil
	}
	if len(raw) > 255 {
		return validation("tautan " + label + " terlalu panjang (maksimal 255 karakter)")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || strings.TrimSpace(parsed.Host) == "" {
		return validation("tautan " + label + " harus berawal https:// dan memiliki domain yang valid")
	}
	return nil
}
