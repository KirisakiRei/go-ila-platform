package email

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"ila/backend/internal/model"
)

// OutboxStore adalah kebutuhan antrean surel oleh worker.
type OutboxStore interface {
	Pending(ctx context.Context, limit int) ([]model.EmailOutbox, error)
	MarkSent(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, reason string, retryAt time.Time, final bool) error
	// TokenFor mengambil token mentah sekali pakai milik satu baris antrean.
	TokenFor(outboxID string) (string, bool)
}

// MaxAttempts membatasi percobaan sebelum pesan dianggap gagal permanen.
// Pesan yang gagal tetap tersimpan agar bisa diperiksa operator.
const MaxAttempts = 5

// RetryDelay adalah jeda antarpercobaan; cukup longgar agar tidak membanjiri SMTP.
const RetryDelay = 5 * time.Minute

// Worker mengirim antrean surel secara berkala.
type Worker struct {
	store  OutboxStore
	sender Sender
	logger *slog.Logger
	now    func() time.Time
}

// NewWorker merangkai pengirim antrean.
func NewWorker(store OutboxStore, sender Sender, logger *slog.Logger) *Worker {
	return &Worker{store: store, sender: sender, logger: logger, now: func() time.Time { return time.Now().UTC() }}
}

// Run mengirim antrean tiap interval sampai konteks selesai.
// Interval sengaja jarang: surel transaksional tidak butuh latensi rendah.
func (w *Worker) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.Drain(ctx)
		}
	}
}

// Drain mengirim seluruh pesan yang siap, satu per satu.
// Kegagalan satu pesan tidak menghentikan pesan lain.
func (w *Worker) Drain(ctx context.Context) {
	rows, err := w.store.Pending(ctx, 20)
	if err != nil {
		w.logger.Error("membaca antrean surel gagal", "error", err.Error())
		return
	}

	for _, row := range rows {
		if ctx.Err() != nil {
			return
		}
		msg, err := Render(row, w.store)
		if err != nil {
			// Pesan yang tidak bisa dirender tidak akan pernah berhasil:
			// langsung ditandai gagal agar tidak berputar.
			_ = w.store.MarkFailed(ctx, row.ID, err.Error(), w.now(), true)
			continue
		}
		if err := w.sender.Send(ctx, msg); err != nil {
			final := row.Attempts+1 >= MaxAttempts
			if markErr := w.store.MarkFailed(ctx, row.ID, err.Error(), w.now().Add(RetryDelay), final); markErr != nil {
				w.logger.Error("menandai surel gagal", "error", markErr.Error())
			}
			w.logger.Warn("pengiriman surel gagal",
				"template", row.Template, "attempt", row.Attempts+1, "final", final)
			continue
		}
		if err := w.store.MarkSent(ctx, row.ID); err != nil {
			w.logger.Error("menandai surel terkirim", "error", err.Error())
		}
	}
}

// Render menyusun judul dan isi berdasarkan template.
// Data yang tidak dikenal menghasilkan kesalahan, bukan surel kosong.
//
// Tautan dibentuk di sini dari token yang dititipkan di memori, bukan dibaca
// dari kolom payload, sehingga token tidak pernah tersimpan di basis data.
func Render(row model.EmailOutbox, tokens TokenSource) (Message, error) {
	var payload struct {
		TokenID string `json:"token_id"`
		Name    string `json:"name"`
		Link    string `json:"link"`
	}
	if err := json.Unmarshal([]byte(row.Payload), &payload); err != nil {
		return Message{}, err
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		name = "Pengguna"
	}

	link := payload.Link
	if link == "" && tokens != nil {
		raw, ok := tokens.TokenFor(row.ID)
		if !ok {
			// Token sudah hangus atau proses pernah berhenti: kirim ulang
			// lebih aman daripada mengirim tautan rusak.
			return Message{}, errTokenGone
		}
		link = LinkFor(row.Template, raw)
	}

	switch row.Template {
	case "reset_password":
		return Message{
			To:      row.ToEmail,
			Subject: "Atur ulang kata sandi Imaginary Liquidity",
			Body: "Halo " + name + ",\n\n" +
				"Kami menerima permintaan untuk mengatur ulang kata sandi akun Anda.\n" +
				"Buka tautan berikut dalam 30 menit ke depan:\n\n" +
				link + "\n\n" +
				"Setelah kata sandi diganti, seluruh perangkat akan keluar otomatis.\n" +
				"Bila Anda tidak meminta ini, abaikan pesan ini; kata sandi tidak berubah.\n",
		}, nil
	case "verify_email":
		return Message{
			To:      row.ToEmail,
			Subject: "Verifikasi alamat email Imaginary Liquidity",
			Body: "Halo " + name + ",\n\n" +
				"Konfirmasi alamat email Anda dengan membuka tautan berikut:\n\n" +
				link + "\n\n" +
				"Tautan berlaku 24 jam. Selama email belum terverifikasi, akun tetap " +
				"bisa digunakan untuk melihat dashboard, namun trial, koneksi broker, " +
				"dan pembayaran belum tersedia.\n",
		}, nil
	default:
		return Message{}, errUnknownTemplate
	}
}

// TokenSource menyediakan token mentah sekali pakai per baris antrean.
type TokenSource interface {
	TokenFor(outboxID string) (string, bool)
}

// LinkFor menyusun tautan bertoken untuk template tertentu.
func LinkFor(template, rawToken string) string {
	switch template {
	case "reset_password":
		return PublicBaseURL + "/reset-password?token=" + rawToken
	case "verify_email":
		return PublicBaseURL + "/verifikasi?token=" + rawToken
	default:
		return ""
	}
}

// PublicBaseURL adalah alamat situs publik untuk tautan surel.
// Diisi saat start dari konfigurasi.
var PublicBaseURL = "http://localhost:3000"

// errTokenGone menandai token yang sudah tidak tersedia di memori.
var errTokenGone = &renderError{"tautan kedaluwarsa; minta ulang"}

// errUnknownTemplate menandai template yang tidak dikenali kode.
var errUnknownTemplate = &renderError{"template surel tidak dikenal"}

type renderError struct{ message string }

func (e *renderError) Error() string { return e.message }
