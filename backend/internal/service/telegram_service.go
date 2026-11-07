package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// OnboardingTokenTTL adalah masa berlaku token /start Telegram (plans/13 §3.2).
const OnboardingTokenTTL = 5 * time.Minute

// TelegramService mengelola penghubungan chat Telegram ke akun.
//
// Bot hanya membalas kalimat pendek untuk chat yang belum terhubung: tidak ada
// data sinyal, status akun, atau info bisnis apa pun (plans/13 §3.4).
type TelegramService interface {
	// CreateLink membuat token sekali pakai dan tautan t.me untuk user.
	CreateLink(ctx context.Context, userID string, botUsername string) (*TelegramLink, error)
	// HandleStart memverifikasi token lalu menyimpan chat_id ke user.
	HandleStart(ctx context.Context, rawToken, chatID string) (*TelegramStartResult, error)
	// Disconnect memutus chat dari akun.
	Disconnect(ctx context.Context, userID string) error
	// ResolveChat mencari user pemilik chat, atau NotFound bila belum terhubung.
	ResolveChat(ctx context.Context, chatID string) (*model.User, error)
}

// TelegramLink adalah tautan onboarding yang dikirim ke dashboard.
type TelegramLink struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// TelegramStartResult adalah hasil penghubungan chat.
type TelegramStartResult struct {
	UserID  string `json:"user_id"`
	ChatID  string `json:"chat_id"`
	Message string `json:"message"`
}

// OnboardingTokenStore menyimpan token onboarding.
type OnboardingTokenStore interface {
	Create(ctx context.Context, token *model.OnboardingToken) error
	FindByHash(ctx context.Context, hash string) (*model.OnboardingToken, error)
	MarkUsed(ctx context.Context, id string, at time.Time) error
}

type telegramService struct {
	users  UserRepository
	tokens OnboardingTokenStore
	audit  AuditRepository
	clock  func() time.Time
}

// NewTelegramService membuat service penghubungan Telegram.
func NewTelegramService(users UserRepository, tokens OnboardingTokenStore, audit AuditRepository) TelegramService {
	return &telegramService{
		users:  users,
		tokens: tokens,
		audit:  audit,
		clock:  func() time.Time { return time.Now().UTC() },
	}
}

// CreateLink membuat token 32 byte; yang disimpan hanya hash-nya.
func (s *telegramService) CreateLink(ctx context.Context, userID string, botUsername string) (*TelegramLink, error) {
	if _, err := s.users.FindByID(ctx, userID); err != nil {
		return nil, err
	}

	raw, err := randomToken()
	if err != nil {
		return nil, err
	}

	now := s.clock()
	expiresAt := now.Add(OnboardingTokenTTL)

	record := &model.OnboardingToken{
		TokenHash: hashOnboarding(raw),
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
	if err := s.tokens.Create(ctx, record); err != nil {
		return nil, err
	}

	bot := strings.TrimPrefix(strings.TrimSpace(botUsername), "@")
	if bot == "" {
		return nil, fmt.Errorf("%w: username bot belum dikonfigurasi", apperror.ErrServiceUnavailable)
	}

	return &TelegramLink{
		URL:       fmt.Sprintf("https://t.me/%s?start=%s", bot, raw),
		ExpiresAt: expiresAt,
	}, nil
}

// HandleStart memverifikasi token dan menyimpan chat_id ke pemiliknya.
func (s *telegramService) HandleStart(ctx context.Context, rawToken, chatID string) (*TelegramStartResult, error) {
	rawToken = strings.TrimSpace(rawToken)
	chatID = strings.TrimSpace(chatID)
	if rawToken == "" || chatID == "" {
		return nil, apperror.ErrBadRequest
	}

	record, err := s.tokens.FindByHash(ctx, hashOnboarding(rawToken))
	if err != nil {
		return nil, err
	}

	now := s.clock()
	if record.UsedAt != nil {
		return nil, errors.Join(apperror.ErrConflict, errors.New("tautan ini sudah dipakai"))
	}
	if now.After(record.ExpiresAt) {
		return nil, errors.Join(apperror.ErrConflict, errors.New("tautan sudah kedaluwarsa"))
	}

	// Chat hanya boleh terikat pada satu akun; chat yang sudah dipakai dipindah.
	if existing, findErr := s.users.FindByTelegramChatID(ctx, chatID); findErr == nil && existing.ID != record.UserID {
		existing.TelegramChatID = nil
		existing.TelegramLinkedAt = nil
		if updateErr := s.users.Update(ctx, existing); updateErr != nil {
			return nil, updateErr
		}
	}

	user, err := s.users.FindByID(ctx, record.UserID)
	if err != nil {
		return nil, err
	}

	user.TelegramChatID = &chatID
	user.TelegramLinkedAt = &now
	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}

	if err := s.tokens.MarkUsed(ctx, record.ID, now); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, user.ID, "telegram.link", "user", user.ID)

	return &TelegramStartResult{
		UserID:  user.ID,
		ChatID:  chatID,
		Message: "Terhubung. Notifikasi setup akan dikirim ke chat ini.",
	}, nil
}

// Disconnect melepas chat dari akun.
func (s *telegramService) Disconnect(ctx context.Context, userID string) error {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	user.TelegramChatID = nil
	user.TelegramLinkedAt = nil
	if err := s.users.Update(ctx, user); err != nil {
		return err
	}

	s.recordAudit(ctx, userID, "telegram.unlink", "user", userID)
	return nil
}

// ResolveChat mencari pemilik chat yang sudah terhubung.
func (s *telegramService) ResolveChat(ctx context.Context, chatID string) (*model.User, error) {
	if strings.TrimSpace(chatID) == "" {
		return nil, apperror.ErrNotFound
	}
	return s.users.FindByTelegramChatID(ctx, chatID)
}

func (s *telegramService) recordAudit(ctx context.Context, userID, event, entity, entityID string) {
	_ = s.audit.Record(ctx, &model.AuditLog{
		UserID:    userID,
		Event:     event,
		Entity:    entity,
		EntityID:  entityID,
		CreatedAt: s.clock(),
	})
}

// randomToken membuat token 32 byte acak berbentuk base64url.
func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("membuat token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashOnboarding(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
