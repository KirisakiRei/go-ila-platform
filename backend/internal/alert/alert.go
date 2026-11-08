// Package alert mengirim peringatan operasional ke admin lewat Telegram.
//
// Anti-banjir sengaja di sisi pengirim: satu pemicu paling banyak satu pesan
// per 15 menit, sehingga ponsel admin tidak bergetar terus saat gangguan
// berkepanjangan (plans/18 §2).
package alert

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// Cooldown adalah jeda minimum antarpesan untuk pemicu yang sama.
const Cooldown = 15 * time.Minute

// Trigger menandai sumber peringatan. Nilainya stabil agar riwayat bisa dibaca.
type Trigger string

const (
	EngineDisconnected   Trigger = "engine_disconnected"
	ExecutionFailing     Trigger = "execution_failing"
	BrokerErrors         Trigger = "broker_errors"
	WebhookSignature     Trigger = "webhook_signature"
	EmailQueueStuck      Trigger = "email_queue_stuck"
	UntrackedPosition    Trigger = "untracked_position"
	SlowQuery            Trigger = "slow_query"
	DiskAlmostFull       Trigger = "disk_almost_full"
	NotifyQueueFull      Trigger = "notify_queue_full"
	AdminLoginBruteforce Trigger = "admin_login_bruteforce"
	// MarketScheduleAnomaly menandai engine yang melaporkan status pasar di
	// luar jadwal NY, biasanya karena zona waktu salah (plans/22 §3).
	MarketScheduleAnomaly Trigger = "market_schedule_anomaly"
)

// Label menjelaskan pemicu dalam bahasa manusia untuk pesan Telegram.
var Label = map[Trigger]string{
	EngineDisconnected:    "Engine terputus",
	ExecutionFailing:      "Eksekusi gagal berulang",
	BrokerErrors:          "Banyak akun broker bermasalah",
	WebhookSignature:      "Webhook dengan tanda tangan salah",
	EmailQueueStuck:       "Antrean surel macet",
	UntrackedPosition:     "Posisi tidak terlacak",
	SlowQuery:             "Query lambat berulang",
	DiskAlmostFull:        "Disk hampir penuh",
	NotifyQueueFull:       "Antrean notifikasi penuh",
	AdminLoginBruteforce:  "Login admin gagal berulang",
	MarketScheduleAnomaly: "Status pasar tidak sesuai jadwal",
}

// Sender mengirim teks ke satu chat.
type Sender interface {
	Send(ctx context.Context, chatID, text string) error
}

// Notifier mengirim peringatan dengan anti-banjir per pemicu.
type Notifier struct {
	sender Sender
	chatID string
	logger *slog.Logger
	now    func() time.Time

	mu   sync.Mutex
	last map[Trigger]time.Time
	// aktif menandai pemicu yang pesannya belum diberi kabar pulih.
	aktif map[Trigger]bool
}

// New membuat notifier admin. Chat kosong berarti peringatan dimatikan,
// bukan error: sistem tetap berjalan tanpa kanal admin.
func New(sender Sender, chatID string, logger *slog.Logger) *Notifier {
	return &Notifier{
		sender: sender,
		chatID: strings.TrimSpace(chatID),
		logger: logger,
		now:    func() time.Time { return time.Now().UTC() },
		last:   make(map[Trigger]time.Time),
		aktif:  make(map[Trigger]bool),
	}
}

// Enabled menandai kanal peringatan siap dipakai.
func (n *Notifier) Enabled() bool {
	return n != nil && n.sender != nil && n.chatID != ""
}

// Fire mengirim peringatan bila cooldown pemicu sudah lewat.
// Nilai kembali menunjukkan apakah pesan benar-benar dikirim.
func (n *Notifier) Fire(ctx context.Context, trigger Trigger, detail string) bool {
	if !n.Enabled() {
		return false
	}

	n.mu.Lock()
	now := n.now()
	if last, ok := n.last[trigger]; ok && now.Sub(last) < Cooldown {
		n.mu.Unlock()
		return false
	}
	n.last[trigger] = now
	n.aktif[trigger] = true
	n.mu.Unlock()

	text := fmt.Sprintf("⚠️ %s\n%s\n\nWaktu: %s", Label[trigger], strings.TrimSpace(detail), now.Format(time.RFC3339))
	if err := n.sender.Send(ctx, n.chatID, text); err != nil {
		n.logger.Error("peringatan admin gagal dikirim", "trigger", string(trigger), "error", err.Error())
		return false
	}
	return true
}

// Recovered memberi kabar pulih satu kali setelah peringatan terkirim.
// Pemicu yang tidak pernah aktif tidak menghasilkan pesan apa pun.
func (n *Notifier) Recovered(ctx context.Context, trigger Trigger, detail string) {
	if !n.Enabled() {
		return
	}

	n.mu.Lock()
	if !n.aktif[trigger] {
		n.mu.Unlock()
		return
	}
	delete(n.aktif, trigger)
	// Cooldown dikosongkan agar gangguan berikutnya langsung dilaporkan.
	delete(n.last, trigger)
	n.mu.Unlock()

	text := fmt.Sprintf("✅ %s pulih\n%s\n\nWaktu: %s", Label[trigger], strings.TrimSpace(detail), n.now().Format(time.RFC3339))
	if err := n.sender.Send(ctx, n.chatID, text); err != nil {
		n.logger.Error("kabar pulih gagal dikirim", "trigger", string(trigger), "error", err.Error())
	}
}
