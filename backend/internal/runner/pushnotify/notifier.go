// Package pushnotify mengirim notifikasi Web Push di latar belakang tanpa
// pernah menahan jalannya evaluasi sinyal maupun eksekusi order di Runner.
package pushnotify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/service"
	"ila/backend/internal/telegram"
	"ila/backend/internal/webpush"
)

// Kind mengelompokkan jenis notifikasi agar cocok dengan saklar per perangkat.
type Kind string

const (
	KindEntry Kind = "entry"
	KindExit  Kind = "exit"
)

// Job merepresentasikan satu kejadian Web Push untuk satu user.
type Job struct {
	UserID     string
	MessageKey string
	Kind       Kind
	TTLSeconds int
	Topic      string
	Build      func(ctx context.Context) service.WebPushPayload
}

// VAPIDProvider menyediakan pasangan kunci VAPID aktif.
type VAPIDProvider interface {
	EnsureVAPID(ctx context.Context) (service.VAPIDKeyPair, error)
}

// Notifier menjalankan antrean worker terbatas untuk Web Push.
type Notifier struct {
	pushRepo      repository.WebPushRepository
	notifRepo     repository.NotificationRepository
	vapid         VAPIDProvider
	client        *webpush.Client
	logger        *slog.Logger
	skipHostCheck bool

	queue chan Job
	wg    sync.WaitGroup
	once  sync.Once
}

// New menyiapkan Notifier Web Push dengan jumlah worker tetap.
func New(
	pushRepo repository.WebPushRepository,
	notifRepo repository.NotificationRepository,
	vapid VAPIDProvider,
	client *webpush.Client,
	logger *slog.Logger,
	workers int,
) *Notifier {
	if workers < 1 {
		workers = 2
	}
	if client == nil {
		client = webpush.NewClient(10 * time.Second)
	}
	if logger == nil {
		logger = slog.Default()
	}

	n := &Notifier{
		pushRepo:  pushRepo,
		notifRepo: notifRepo,
		vapid:     vapid,
		client:    client,
		logger:    logger,
		queue:     make(chan Job, 256),
	}

	for i := 0; i < workers; i++ {
		n.wg.Add(1)
		go n.worker()
	}
	return n
}

// Enqueue menaruh pekerjaan ke antrean secara non-blocking.
func (n *Notifier) Enqueue(job Job) {
	if n == nil || job.UserID == "" || job.Build == nil {
		return
	}
	select {
	case n.queue <- job:
	default:
		n.logger.Warn("antrean Web Push penuh; notifikasi dilewati",
			"user_id", job.UserID, "kind", string(job.Kind))
	}
}

// WithSkipHostAllowlist menonaktifkan pemeriksaan domain gateway HANYA pada pengujian httptest.
func (n *Notifier) WithSkipHostAllowlist(skip bool) *Notifier {
	n.skipHostCheck = skip
	return n
}

// Close menutup antrean dan menunggu seluruh worker selesai.
func (n *Notifier) Close() {
	if n == nil {
		return
	}
	n.once.Do(func() {
		close(n.queue)
		n.wg.Wait()
	})
}

func (n *Notifier) worker() {
	defer n.wg.Done()
	for job := range n.queue {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		n.deliver(ctx, job)
		cancel()
	}
}

func (n *Notifier) deliver(ctx context.Context, job Job) {
	if err := n.pushRepo.EnsureSchema(ctx); err != nil {
		n.logger.Warn("skema Web Push belum siap", "error", err.Error())
		return
	}

	subs, err := n.pushRepo.ListActiveByUser(ctx, job.UserID, repository.MaxPushDevicesPerUser)
	if err != nil {
		n.logger.Warn("gagal membaca langganan Web Push user",
			"user_id", job.UserID, "error", err.Error())
		return
	}

	targets := make([]model.WebPushSubscription, 0, len(subs))
	for i := range subs {
		sub := subs[i]
		if job.Kind == KindEntry && sub.NotifyEntry {
			targets = append(targets, sub)
		} else if job.Kind == KindExit && sub.NotifyExit {
			targets = append(targets, sub)
		}
	}
	// Jangan mengklaim baris notification_log bila user belum punya perangkat aktif.
	if len(targets) == 0 {
		return
	}

	msgKey := safeWebPushKey(job.MessageKey)
	if n.notifRepo != nil {
		claimed, claimErr := n.notifRepo.ClaimMessage(ctx, job.UserID, "webpush", msgKey)
		if claimErr != nil {
			n.logger.Warn("gagal mengklaim idempotensi Web Push",
				"user_id", job.UserID, "error", claimErr.Error())
			return
		}
		if !claimed {
			return
		}
	}

	keys, err := n.vapid.EnsureVAPID(ctx)
	if err != nil {
		if n.notifRepo != nil {
			_ = n.notifRepo.MarkMessageFailed(ctx, msgKey, err.Error())
		}
		n.logger.Error("kunci VAPID tidak siap di Runner", "error", err.Error())
		return
	}

	payload := job.Build(ctx)
	payload.Title = telegram.Sanitize(strings.TrimSpace(payload.Title))
	payload.Body = telegram.Sanitize(strings.TrimSpace(payload.Body))
	if payload.Title == "" && payload.Body == "" {
		if n.notifRepo != nil {
			_ = n.notifRepo.MarkMessageSent(ctx, msgKey)
		}
		return
	}
	if payload.Timestamp == 0 {
		payload.Timestamp = time.Now().UTC().UnixMilli()
	}
	if payload.Kind == "" {
		payload.Kind = string(job.Kind)
	}

	rawJSON, err := json.Marshal(payload)
	if err != nil {
		if n.notifRepo != nil {
			_ = n.notifRepo.MarkMessageFailed(ctx, msgKey, err.Error())
		}
		return
	}

	delivered := 0
	expired := 0
	var lastErr error
	now := time.Now().UTC()

	for i := range targets {
		sub := &targets[i]
		_, sendErr := n.client.Send(ctx, webpush.Subscription{
			Endpoint: sub.Endpoint,
			P256dh:   sub.P256dh,
			Auth:     sub.Auth,
		}, rawJSON, webpush.SendOptions{
			VAPIDPublicKey:    keys.PublicKey,
			VAPIDPrivateKey:   keys.PrivateKey,
			VAPIDSubject:      keys.Subject,
			TTLSeconds:        job.TTLSeconds,
			Urgency:           "high",
			Topic:             safeTopic(job.Topic),
			SkipHostAllowlist: n.skipHostCheck,
		})
		if sendErr != nil {
			if errors.Is(sendErr, webpush.ErrSubscriptionExpired) {
				expired++
				_ = n.pushRepo.DeleteExpiredByID(ctx, sub.ID)
				continue
			}
			lastErr = sendErr
			continue
		}
		delivered++
		_ = n.pushRepo.TouchLastSent(ctx, sub.ID, now)
	}

	if n.notifRepo != nil {
		if delivered > 0 || expired == len(targets) {
			_ = n.notifRepo.MarkMessageSent(ctx, msgKey)
		} else if lastErr != nil {
			_ = n.notifRepo.MarkMessageFailed(ctx, msgKey, lastErr.Error())
		}
	}
}

// safeWebPushKey menjamin awalan `wp:` agar tidak pernah bentrok dengan kunci
// kanal `telegram` pada indeks unik `notification_log.message_key`, serta
// menjaga panjang <= 115 karakter (kolom varchar(120)).
func safeWebPushKey(raw string) string {
	key := strings.TrimSpace(raw)
	if !strings.HasPrefix(key, "wp:") {
		key = "wp:" + key
	}
	if len(key) <= 115 {
		return key
	}
	sum := sha256.Sum256([]byte(key))
	return "wp:h:" + hex.EncodeToString(sum[:24])
}

func safeTopic(raw string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, strings.TrimSpace(raw))
	if len(cleaned) > 32 {
		return cleaned[:32]
	}
	return cleaned
}
