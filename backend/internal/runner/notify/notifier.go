// Package notify menjembatani perubahan status Runner ke kanal notifikasi.
//
// Semua pengiriman berjalan di antrean terpisah: kegagalan Telegram tidak
// pernah menghalangi eksekusi (plans/13 §7).
package notify

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/telegram"
	"ila/backend/pkg/redact"
)

// UserLookup menyediakan tujuan notifikasi per user.
type UserLookup interface {
	FindUserByID(ctx context.Context, userID string) (*model.User, error)
}

// ClaimCheck memastikan satu pesan tidak terkirim dua kali.
type ClaimCheck interface {
	// ClaimMessage mengembalikan true bila pesan baru dan boleh dikirim.
	ClaimMessage(ctx context.Context, userID, channel, messageKey string) (bool, error)
	MarkMessageSent(ctx context.Context, messageKey string) error
	MarkMessageFailed(ctx context.Context, messageKey, reason string) error
}

// MessageRecorder menyimpan jejak pesan bot yang masih bertombol.
//
// Jejak ini dipakai untuk membersihkan tombol saat setup tidak berlaku lagi.
type MessageRecorder interface {
	Simpan(ctx context.Context, pesan *model.TelegramMessage) error
}

// Notifier mengirim notifikasi lewat antrean.
type Notifier struct {
	users    UserLookup
	claims   ClaimCheck
	tg       *telegram.Client
	logger   *slog.Logger
	queue    chan job
	recorder MessageRecorder
	wg       sync.WaitGroup
}

type job struct {
	userID     string
	messageKey string
	// signalID diisi bila pesan memuat tombol yang terikat ke satu sinyal.
	signalID string
	build    func(chatID string) telegram.Message
}

// New membuat notifier dengan sejumlah pekerja tetap.
//
// recorder boleh nil: tanpa pencatat, pesan tetap terkirim tetapi tombolnya
// tidak bisa dibersihkan otomatis.
func New(users UserLookup, claims ClaimCheck, tg *telegram.Client, recorder MessageRecorder, logger *slog.Logger, workers int) *Notifier {
	if workers < 1 {
		workers = 1
	}

	n := &Notifier{
		users:    users,
		claims:   claims,
		tg:       tg,
		logger:   logger,
		queue:    make(chan job, 256),
		recorder: recorder,
	}

	for i := 0; i < workers; i++ {
		n.wg.Add(1)
		go n.worker()
	}
	return n
}

// Close menutup antrean dan menunggu pesan yang tersisa selesai.
func (n *Notifier) Close() {
	close(n.queue)
	n.wg.Wait()
}

// Enqueue memasukkan pesan ke antrean tanpa memblokir pemanggil.
// Antrean penuh berarti pesan dibuang dan dicatat, bukan membuat Runner macet.
func (n *Notifier) Enqueue(userID, messageKey string, build func(chatID string) telegram.Message) {
	n.EnqueueSignal(userID, messageKey, "", build)
}

// EnqueueSignal sama dengan Enqueue, tetapi menandai pesan sebagai milik satu
// sinyal agar tombolnya bisa dibersihkan belakangan.
func (n *Notifier) EnqueueSignal(userID, messageKey, signalID string, build func(chatID string) telegram.Message) {
	if n == nil || n.tg == nil {
		return
	}
	select {
	case n.queue <- job{userID: userID, messageKey: messageKey, signalID: signalID, build: build}:
	default:
		n.logger.Warn("antrean notifikasi penuh, pesan dilewati",
			"user_id", userID, "message_key", messageKey)
	}
}

func (n *Notifier) worker() {
	defer n.wg.Done()

	for j := range n.queue {
		n.deliver(j)
	}
}

func (n *Notifier) deliver(j job) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := n.users.FindUserByID(ctx, j.userID)
	if err != nil {
		n.logger.Warn("penerima notifikasi tidak ditemukan", "user_id", j.userID)
		return
	}

	// User tidak menghubungkan Telegram atau mematikan notifikasi:
	// bukan kegagalan, hanya tidak ada tujuan.
	if user.TelegramChatID == nil || *user.TelegramChatID == "" || !user.NotifyEnabled {
		return
	}

	if n.claims != nil {
		claimed, claimErr := n.claims.ClaimMessage(ctx, j.userID, "telegram", j.messageKey)
		if claimErr != nil {
			n.logger.Error("gagal mengklaim pesan", "error", claimErr.Error())
			return
		}
		if !claimed {
			return
		}
	}

	message := j.build(*user.TelegramChatID)

	// Pesan yang memuat istilah internal diganti versi aman sebelum dikirim.
	if telegram.ContainsForbiddenTerms(message.Text) {
		message.Text = telegram.Sanitize(message.Text)
	}

	messageID, err := n.tg.SendMessageReturningID(ctx, *user.TelegramChatID, message)
	if err != nil {
		// Teks error disaring dahulu: kegagalan jaringan bisa memuat URL yang
		// berisi token bot, dan nilainya ikut tercatat di log serta database.
		safeError := redact.Error(err)
		n.logger.Warn("gagal mengirim notifikasi",
			"user_id", j.userID, "message_key", j.messageKey, "error", safeError)
		if n.claims != nil {
			_ = n.claims.MarkMessageFailed(ctx, j.messageKey, safeError)
		}
		return
	}
	if n.claims != nil {
		_ = n.claims.MarkMessageSent(ctx, j.messageKey)
	}
	// Hanya pesan bertombol yang perlu diingat: tombol itulah yang harus
	// dilepas saat setup batal atau kedaluwarsa.
	if n.recorder != nil && len(message.Buttons) > 0 && messageID != 0 {
		jejak := &model.TelegramMessage{
			UserID:    j.userID,
			ChatID:    *user.TelegramChatID,
			MessageID: messageID,
			SignalID:  j.signalID,
			Kind:      model.TelegramPesanSetup,
		}
		if err := n.recorder.Simpan(ctx, jejak); err != nil {
			n.logger.Warn("gagal mencatat jejak pesan bertombol",
				"user_id", j.userID, "message_key", j.messageKey, "error", redact.Error(err))
		}
	}
}
