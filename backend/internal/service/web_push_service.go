package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"sync"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/telegram"
	"ila/backend/internal/webpush"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/cryptox"
)

const vapidOwnerContext = "webpush-vapid-v1"

// WebPushPayload adalah muatan JSON yang dikirim terenkripsi ke Service Worker.
type WebPushPayload struct {
	Title              string `json:"title"`
	Body               string `json:"body"`
	URL                string `json:"url"`
	Tag                string `json:"tag,omitempty"`
	Kind               string `json:"kind"` // entry | exit | test
	RequireInteraction bool   `json:"require_interaction,omitempty"`
	Timestamp          int64  `json:"timestamp"`
}

// VAPIDKeyPair menyimpan pasangan kunci VAPID aktif.
type VAPIDKeyPair struct {
	PublicKey  string
	PrivateKey string
	Subject    string
}

// WebPushService mengatur pendaftaran perangkat, kunci VAPID, dan uji kirim.
type WebPushService interface {
	EnsureVAPID(ctx context.Context) (VAPIDKeyPair, error)
	Status(ctx context.Context, userID string, query dto.PushPaginationQuery) (*dto.PushStatusResponse, *dto.PushPaginationMeta, error)
	Subscribe(ctx context.Context, userID, fallbackUA string, req dto.PushSubscribeRequest) (*dto.PushSubscriptionItem, error)
	UpdatePreferences(ctx context.Context, userID, id string, req dto.PushUpdatePreferencesRequest) (*dto.PushSubscriptionItem, error)
	Delete(ctx context.Context, userID, id string) error
	Unsubscribe(ctx context.Context, userID string, req dto.PushUnsubscribeRequest) error
	SendTest(ctx context.Context, userID string) (*dto.PushTestResponse, error)
}

// WebPushServiceDeps menampung dependensi layanan Web Push.
type WebPushServiceDeps struct {
	Repo            repository.WebPushRepository
	Sealer          *cryptox.Sealer
	Client          *webpush.Client
	EnvPublicKey    string
	EnvPrivateKey   string
	Subject         string
	SkipHostCheck   bool // Hanya untuk pengujian lokal httptest
}

type webPushService struct {
	repo          repository.WebPushRepository
	sealer        *cryptox.Sealer
	client        *webpush.Client
	envPub        string
	envPriv       string
	subject       string
	skipHostCheck bool

	mu         sync.Mutex
	cachedKeys *VAPIDKeyPair
	schemaDone bool
}

// NewWebPushService menyiapkan layanan Web Push.
func NewWebPushService(deps WebPushServiceDeps) WebPushService {
	client := deps.Client
	if client == nil {
		client = webpush.NewClient(10 * time.Second)
	}
	return &webPushService{
		repo:          deps.Repo,
		sealer:        deps.Sealer,
		client:        client,
		envPub:        strings.TrimSpace(deps.EnvPublicKey),
		envPriv:       strings.TrimSpace(deps.EnvPrivateKey),
		subject:       webpush.NormalizeSubject(deps.Subject),
		skipHostCheck: deps.SkipHostCheck,
	}
}

func (s *webPushService) ensureReady(ctx context.Context) error {
	s.mu.Lock()
	done := s.schemaDone
	s.mu.Unlock()
	if done {
		return nil
	}
	if err := s.repo.EnsureSchema(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	s.schemaDone = true
	s.mu.Unlock()
	return nil
}

// EnsureVAPID memastikan pasangan kunci VAPID tersedia dan konsisten di seluruh proses.
func (s *webPushService) EnsureVAPID(ctx context.Context) (VAPIDKeyPair, error) {
	if err := s.ensureReady(ctx); err != nil {
		return VAPIDKeyPair{}, err
	}

	s.mu.Lock()
	if s.cachedKeys != nil {
		cp := *s.cachedKeys
		s.mu.Unlock()
		return cp, nil
	}
	s.mu.Unlock()

	if s.envPriv != "" {
		_, derivedPub, err := webpush.ParsePrivateKey(s.envPriv)
		if err != nil {
			return VAPIDKeyPair{}, fmt.Errorf("VAPID_PRIVATE_KEY tidak sah: %w", err)
		}
		pub := s.envPub
		if pub == "" {
			pub = derivedPub
		}
		pair := VAPIDKeyPair{PublicKey: pub, PrivateKey: s.envPriv, Subject: s.subject}
		s.cachePair(pair)
		return pair, nil
	}

	existing, err := s.repo.GetVAPID(ctx)
	if err != nil && !errors.Is(err, apperror.ErrNotFound) {
		return VAPIDKeyPair{}, err
	}
	if existing != nil {
		priv, openErr := s.unsealPrivateKey(existing.PrivateKeySealed)
		if openErr == nil {
			if _, derivedPub, parseErr := webpush.ParsePrivateKey(priv); parseErr == nil {
				pair := VAPIDKeyPair{PublicKey: derivedPub, PrivateKey: priv, Subject: s.subject}
				s.cachePair(pair)
				return pair, nil
			}
		}
	}

	pub, priv, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		return VAPIDKeyPair{}, err
	}
	sealed, keyVer, err := s.sealPrivateKey(priv)
	if err != nil {
		return VAPIDKeyPair{}, err
	}

	saved, err := s.repo.SaveVAPIDIfAbsent(ctx, &model.WebPushVAPID{
		ID:               1,
		PublicKey:        pub,
		PrivateKeySealed: sealed,
		KeyVersion:       keyVer,
	})
	if err != nil {
		return VAPIDKeyPair{}, err
	}

	finalPriv, err := s.unsealPrivateKey(saved.PrivateKeySealed)
	if err != nil {
		return VAPIDKeyPair{}, err
	}
	_, finalPub, err := webpush.ParsePrivateKey(finalPriv)
	if err != nil {
		return VAPIDKeyPair{}, err
	}

	pair := VAPIDKeyPair{PublicKey: finalPub, PrivateKey: finalPriv, Subject: s.subject}
	s.cachePair(pair)
	return pair, nil
}

func (s *webPushService) cachePair(pair VAPIDKeyPair) {
	s.mu.Lock()
	s.cachedKeys = &pair
	s.mu.Unlock()
}

func (s *webPushService) sealPrivateKey(priv string) (string, int, error) {
	if s.sealer == nil {
		return "plain:" + priv, 0, nil
	}
	enc, err := s.sealer.SealString(priv, vapidOwnerContext)
	if err != nil {
		return "", 0, err
	}
	return "enc:" + enc, s.sealer.Version(), nil
}

func (s *webPushService) unsealPrivateKey(stored string) (string, error) {
	if strings.HasPrefix(stored, "plain:") {
		return strings.TrimPrefix(stored, "plain:"), nil
	}
	raw := strings.TrimPrefix(stored, "enc:")
	if s.sealer == nil {
		return "", errors.New("master key belum siap untuk membuka kunci VAPID")
	}
	return s.sealer.OpenString(raw, vapidOwnerContext)
}

func (s *webPushService) Status(ctx context.Context, userID string, query dto.PushPaginationQuery) (*dto.PushStatusResponse, *dto.PushPaginationMeta, error) {
	keys, err := s.EnsureVAPID(ctx)
	if err != nil {
		return nil, nil, err
	}

	page := query.Page
	if page < 1 {
		page = 1
	}
	perPage := query.PerPage
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}

	rows, total, err := s.repo.ListByUser(ctx, userID, page, perPage)
	if err != nil {
		return nil, nil, err
	}

	items := make([]dto.PushSubscriptionItem, 0, len(rows))
	for i := range rows {
		items = append(items, mapSubscriptionDTO(&rows[i]))
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	if totalPages < 1 {
		totalPages = 1
	}

	meta := &dto.PushPaginationMeta{
		CurrentPage: page,
		PerPage:     perPage,
		TotalItems:  total,
		TotalPages:  totalPages,
	}

	return &dto.PushStatusResponse{
		Supported:      keys.PublicKey != "",
		VAPIDPublicKey: keys.PublicKey,
		Subscriptions:  items,
	}, meta, nil
}

func (s *webPushService) Subscribe(ctx context.Context, userID, fallbackUA string, req dto.PushSubscribeRequest) (*dto.PushSubscriptionItem, error) {
	if err := s.ensureReady(ctx); err != nil {
		return nil, err
	}

	if !s.skipHostCheck {
		if err := webpush.ValidateEndpoint(req.Endpoint); err != nil {
			return nil, errors.Join(apperror.ErrBadRequest, err)
		}
	}

	normPub, normAuth, err := webpush.ValidateSubscriptionKeys(req.Keys.P256dh, req.Keys.Auth)
	if err != nil {
		return nil, errors.Join(apperror.ErrBadRequest, err)
	}

	notifyEntry := true
	if req.NotifyEntry != nil {
		notifyEntry = *req.NotifyEntry
	}
	notifyExit := true
	if req.NotifyExit != nil {
		notifyExit = *req.NotifyExit
	}

	label := strings.TrimSpace(req.DeviceLabel)
	if label == "" {
		label = summarizeUserAgent(fallbackUA)
	}
	if len(label) > 120 {
		label = label[:120]
	}

	endpoint := strings.TrimSpace(req.Endpoint)
	saved, err := s.repo.Upsert(ctx, &model.WebPushSubscription{
		UserID:       userID,
		EndpointHash: webpush.EndpointHash(endpoint),
		Endpoint:     endpoint,
		P256dh:       normPub,
		Auth:         normAuth,
		DeviceLabel:  label,
		NotifyEntry:  notifyEntry,
		NotifyExit:   notifyExit,
	})
	if err != nil {
		return nil, err
	}

	item := mapSubscriptionDTO(saved)
	return &item, nil
}

func (s *webPushService) UpdatePreferences(ctx context.Context, userID, id string, req dto.PushUpdatePreferencesRequest) (*dto.PushSubscriptionItem, error) {
	if req.NotifyEntry == nil && req.NotifyExit == nil {
		return nil, errors.Join(apperror.ErrBadRequest, errors.New("tidak ada preferensi yang diubah"))
	}
	if err := s.ensureReady(ctx); err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdatePreferences(ctx, userID, id, req.NotifyEntry, req.NotifyExit)
	if err != nil {
		return nil, err
	}
	item := mapSubscriptionDTO(updated)
	return &item, nil
}

func (s *webPushService) Delete(ctx context.Context, userID, id string) error {
	if err := s.ensureReady(ctx); err != nil {
		return err
	}
	return s.repo.DeleteByID(ctx, userID, id)
}

func (s *webPushService) Unsubscribe(ctx context.Context, userID string, req dto.PushUnsubscribeRequest) error {
	if err := s.ensureReady(ctx); err != nil {
		return err
	}
	hash := webpush.EndpointHash(req.Endpoint)
	return s.repo.DeleteByEndpointHash(ctx, userID, hash)
}

func (s *webPushService) SendTest(ctx context.Context, userID string) (*dto.PushTestResponse, error) {
	keys, err := s.EnsureVAPID(ctx)
	if err != nil {
		return nil, err
	}

	subs, _, err := s.repo.ListByUser(ctx, userID, 1, repository.MaxPushDevicesPerUser)
	if err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, errors.Join(apperror.ErrBadRequest, errors.New("belum ada perangkat aktif yang terdaftar untuk notifikasi Web Push"))
	}

	payloadBytes, err := json.Marshal(WebPushPayload{
		Title:     telegram.Sanitize("Tes Notifikasi · ILA Platform"),
		Body:      telegram.Sanitize("Notifikasi push perangkat ini aktif! Peringatan entry dan penutupan posisi (TP/SL/Manual) siap dikirim."),
		URL:       "/app/pengaturan",
		Tag:       "ila-push-test",
		Kind:      "test",
		Timestamp: time.Now().UTC().UnixMilli(),
	})
	if err != nil {
		return nil, err
	}

	res := &dto.PushTestResponse{}
	now := time.Now().UTC()
	for i := range subs {
		sub := &subs[i]
		_, sendErr := s.client.Send(ctx, webpush.Subscription{
			Endpoint: sub.Endpoint,
			P256dh:   sub.P256dh,
			Auth:     sub.Auth,
		}, payloadBytes, webpush.SendOptions{
			VAPIDPublicKey:    keys.PublicKey,
			VAPIDPrivateKey:   keys.PrivateKey,
			VAPIDSubject:      keys.Subject,
			TTLSeconds:        300,
			Urgency:           "high",
			Topic:             "ila-test",
			SkipHostAllowlist: s.skipHostCheck,
		})
		if sendErr != nil {
			if errors.Is(sendErr, webpush.ErrSubscriptionExpired) {
				res.Expired++
				_ = s.repo.DeleteExpiredByID(ctx, sub.ID)
				continue
			}
			res.Failed++
			continue
		}
		res.Delivered++
		_ = s.repo.TouchLastSent(ctx, sub.ID, now)
	}

	return res, nil
}

func mapSubscriptionDTO(sub *model.WebPushSubscription) dto.PushSubscriptionItem {
	var lastSent *string
	if sub.LastSentAt != nil && !sub.LastSentAt.IsZero() {
		formatted := sub.LastSentAt.UTC().Format(time.RFC3339)
		lastSent = &formatted
	}
	host := ""
	if parsed, err := url.Parse(sub.Endpoint); err == nil {
		host = parsed.Host
	}
	label := sub.DeviceLabel
	if label == "" {
		label = "Browser Web"
	}
	return dto.PushSubscriptionItem{
		ID:           sub.ID,
		DeviceLabel:  label,
		NotifyEntry:  sub.NotifyEntry,
		NotifyExit:   sub.NotifyExit,
		LastSentAt:   lastSent,
		CreatedAt:    sub.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    sub.UpdatedAt.UTC().Format(time.RFC3339),
		EndpointHost: host,
	}
}

func summarizeUserAgent(ua string) string {
	lower := strings.ToLower(ua)
	browser := "Browser"
	switch {
	case strings.Contains(lower, "edg/"):
		browser = "Edge"
	case strings.Contains(lower, "firefox/"):
		browser = "Firefox"
	case strings.Contains(lower, "chrome/") && !strings.Contains(lower, "edg/"):
		browser = "Chrome"
	case strings.Contains(lower, "safari/") && !strings.Contains(lower, "chrome/"):
		browser = "Safari"
	}

	osName := "Perangkat"
	switch {
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad"):
		osName = "iOS"
	case strings.Contains(lower, "android"):
		osName = "Android"
	case strings.Contains(lower, "windows"):
		osName = "Windows"
	case strings.Contains(lower, "mac os"):
		osName = "macOS"
	case strings.Contains(lower, "linux"):
		osName = "Linux"
	}
	return browser + " · " + osName
}
