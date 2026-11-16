package handler

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"ila/backend/internal/model"
	"ila/backend/internal/service"
	"ila/backend/internal/telegram"
)

// callbackUpdate adalah bagian callback_query pada payload Telegram.
type callbackUpdate struct {
	ID   string `json:"id"`
	Data string `json:"data"`
	// From dipakai hanya sebagai catatan; hak akses tetap ditentukan oleh
	// chat_id yang terhubung ke akun.
	From struct {
		ID int64 `json:"id"`
	} `json:"from"`
	Message struct {
		MessageID int64 `json:"message_id"`
		Chat      struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
}

// WithMenu menyuntikkan penyusun halaman menu, layanan konfirmasi entry,
// alamat dashboard, dan logger.
//
// Menu hanya aktif bila klien Telegram mendukung penggantian pesan; tanpa itu
// tombol inline tidak bisa diperbarui dan webhook tetap membalas 200.
func (h *TelegramHandler) WithMenu(menu service.TelegramMenuService, approve service.ApproveService, dashboardURL string, logger *slog.Logger) *TelegramHandler {
	h.menu = menu
	h.approve = approve
	h.baseURL = dashboardURL
	if logger != nil {
		h.logger = logger
	}
	return h
}

// handleCallback memproses tekanan tombol inline pada pesan bot.
//
// Callback hanya menunjuk aksi; identitas pemilik akun selalu dicari dari
// chat_id yang terdaftar, bukan dari isi callback_data.
func (h *TelegramHandler) handleCallback(ctx context.Context, cb *callbackUpdate) {
	if h.edit == nil {
		return
	}
	chatID := strconv.FormatInt(cb.Message.Chat.ID, 10)

	// Spinner pada tombol harus selalu ditutup, termasuk saat pemrosesan gagal.
	defer func() {
		if err := h.edit.AnswerCallback(ctx, cb.ID, ""); err != nil && h.logger != nil {
			h.logger.WarnContext(ctx, "gagal menutup indikator tombol telegram", slog.String("error", err.Error()))
		}
	}()

	if h.approve == nil {
		h.edit.AnswerCallback(ctx, cb.ID, "Menu belum aktif")
		return
	}

	user, err := h.telegram.ResolveChat(ctx, chatID)
	if err != nil {
		h.tolakCallback(ctx, cb, chatID)
		return
	}
	messageID := cb.Message.MessageID

	// Aksi yang butuh data jurnal hanya dilayani bila penyusun halaman ada.
	butuhMenu := func() bool {
		if h.menu != nil {
			return true
		}
		h.edit.AnswerCallback(ctx, cb.ID, "Menu belum aktif")
		return false
	}

	switch aksi := strings.TrimSpace(cb.Data); {
	case strings.HasPrefix(aksi, telegram.CallbackEksekusi):
		h.confirmEntry(ctx, user, chatID, messageID, strings.TrimPrefix(aksi, telegram.CallbackEksekusi))
	case strings.HasPrefix(aksi, telegram.CallbackLewati):
		h.declineEntry(ctx, user, chatID, messageID, strings.TrimPrefix(aksi, telegram.CallbackLewati))
	case strings.HasPrefix(aksi, telegram.CallbackRiwayat):
		if !butuhMenu() {
			return
		}
		offset, _ := strconv.Atoi(strings.TrimPrefix(aksi, telegram.CallbackRiwayat))
		h.gantiPesan(ctx, chatID, messageID, func() (telegram.Message, error) {
			return h.menu.Riwayat(ctx, user, offset)
		})
	case aksi == telegram.CallbackMenu:
		h.ketikPesan(ctx, chatID, messageID, telegram.FormatMenuMessage("", ""), nil)
	case aksi == telegram.CallbackPantau || aksi == telegram.CallbackSegarkan:
		if !butuhMenu() {
			return
		}
		h.gantiPesan(ctx, chatID, messageID, func() (telegram.Message, error) {
			return h.menu.Pantau(ctx, user, h.dashboardLink())
		})
	case aksi == telegram.CallbackHariIni:
		if !butuhMenu() {
			return
		}
		h.gantiPesan(ctx, chatID, messageID, func() (telegram.Message, error) {
			return h.menu.HariIni(ctx, user, h.dashboardLink())
		})
	case aksi == telegram.CallbackPengaturan:
		h.ketikPesan(ctx, chatID, messageID, telegram.FormatPengaturanMessage(h.dashboardLink()), nil)
	default:
		h.edit.AnswerCallback(ctx, cb.ID, "Aksi tidak dikenal")
	}
}

// confirmEntry mengeksekusi setup lewat jalur yang sama dengan halaman
// konfirmasi web. Seluruh syarat trading tetap diperiksa ulang di Runner.
func (h *TelegramHandler) confirmEntry(ctx context.Context, user *model.User, chatID string, messageID int64, token string) {
	if token == "" {
		h.ketikPesan(ctx, chatID, messageID, telegram.FormatKedaluwarsaTautan(""), nil)
		return
	}
	hasil, err := h.approve.Confirm(ctx, user.ID, token)
	if err != nil {
		h.logInfo(ctx, "eksekusi dari telegram gagal", err)
		h.ketikPesan(ctx, chatID, messageID, telegram.FormatKedaluwarsaTautan(""), nil)
		return
	}
	if hasil.Status == model.SignalStatusRejected {
		h.ketikPesan(ctx, chatID, messageID, telegram.FormatEksekusiGagal("", "", "", hasil.Message), nil)
		return
	}
	h.ketikPesan(ctx, chatID, messageID, telegram.Message{
		Text: "⚡ Eksekusi dikirim\n\nEntry sudah diteruskan ke broker. Rincian order menyusul di chat ini begitu konfirmasi diterima.\n",
	}, nil)
}

// declineEntry menandai setup dilewati dan melepas tombolnya.
func (h *TelegramHandler) declineEntry(ctx context.Context, user *model.User, chatID string, messageID int64, token string) {
	if token == "" {
		h.ketikPesan(ctx, chatID, messageID, telegram.FormatKedaluwarsaTautan(""), nil)
		return
	}
	if _, err := h.approve.Decline(ctx, user.ID, token); err != nil {
		h.logInfo(ctx, "melewati setup dari telegram gagal", err)
		h.ketikPesan(ctx, chatID, messageID, telegram.FormatKedaluwarsaTautan(""), nil)
		return
	}
	h.ketikPesan(ctx, chatID, messageID, telegram.FormatDilewatiMessage("", "", ""), nil)
}

// gantiPesan menyusun halaman lalu mengganti isi pesan yang ditekan.
func (h *TelegramHandler) gantiPesan(ctx context.Context, chatID string, messageID int64, susun func() (telegram.Message, error)) {
	pesan, err := susun()
	if err != nil {
		h.logInfo(ctx, "menyusun halaman menu gagal", err)
		h.ketikPesan(ctx, chatID, messageID, telegram.Message{Text: "Halaman belum bisa dimuat. Coba lagi sebentar lagi."}, nil)
		return
	}
	h.ketikPesan(ctx, chatID, messageID, pesan, pesan.Buttons)
}

// ketikPesan mengganti isi pesan lama. Pesan yang terlalu tua untuk diedit
// dianggap bukan kegagalan: yang dipakai adalah salinan baru.
func (h *TelegramHandler) ketikPesan(ctx context.Context, chatID string, messageID int64, pesan telegram.Message, buttons [][]telegram.InlineButton) {
	if buttons == nil {
		buttons = [][]telegram.InlineButton{}
	}
	if err := h.edit.EditMessageText(ctx, chatID, messageID, pesan.Text, buttons); err != nil {
		h.logInfo(ctx, "mengganti pesan telegram gagal", err)
		if _, kirimErr := h.edit.SendMessageReturningID(ctx, chatID, pesan); kirimErr != nil {
			h.logInfo(ctx, "mengirim pesan pengganti gagal", kirimErr)
		}
	}
}

// tolakCallback menjawab tombol dari chat yang belum terhubung.
func (h *TelegramHandler) tolakCallback(ctx context.Context, cb *callbackUpdate, chatID string) {
	_ = h.edit.AnswerCallback(ctx, cb.ID, "Akun belum terhubung")
	if h.sender == nil {
		return
	}
	_ = h.sender.Send(ctx, chatID, "Chat ini belum terhubung ke akun BOT Trading ILA. Hubungkan lewat menu Pengaturan di dashboard.")
}

// handleMenuTeks membalas tekanan tombol keyboard tetap.
//
// Tombol keyboard mengirim teks biasa, sehingga halamannya dikirim sebagai
// pesan baru, bukan penggantian pesan lama.
func (h *TelegramHandler) handleMenuTeks(ctx context.Context, chatID, aksi string) {
	if h.menu == nil {
		return
	}
	user, err := h.telegram.ResolveChat(ctx, chatID)
	if err != nil {
		if h.sender != nil {
			_ = h.sender.Send(ctx, chatID, "Chat ini belum terhubung ke akun BOT Trading ILA. Hubungkan lewat menu Pengaturan di dashboard.")
		}
		return
	}

	var pesan telegram.Message
	switch {
	case aksi == telegram.CallbackPantau:
		pesan, err = h.menu.Pantau(ctx, user, h.dashboardLink())
	case aksi == telegram.CallbackHariIni:
		pesan, err = h.menu.HariIni(ctx, user, h.dashboardLink())
	case strings.HasPrefix(aksi, telegram.CallbackRiwayat):
		pesan, err = h.menu.Riwayat(ctx, user, 0)
	case aksi == telegram.CallbackPengaturan:
		pesan = telegram.FormatPengaturanMessage(h.dashboardLink())
	default:
		pesan = telegram.FormatMenuMessage("", "")
	}
	if err != nil {
		h.logInfo(ctx, "menyusun halaman menu gagal", err)
		pesan = telegram.Message{Text: "Halaman belum bisa dimuat. Coba lagi sebentar lagi."}
	}
	if h.edit != nil {
		kb := telegram.KeyboardUtama()
		pesan.Keyboard = &kb
		if _, err := h.edit.SendMessageReturningID(ctx, chatID, pesan); err != nil {
			h.logInfo(ctx, "mengirim halaman menu gagal", err)
		}
	}
}

// pasangKeyboard mengirim pesan pembuka sekaligus memasang menu tetap.
//
// Telegram menempelkan keyboard pada pesan, sehingga menu hilang bila pesan
// terakhir bot tidak membawanya. Semua balasan menu memakai helper ini.
// Kanal yang tidak mendukung keyboard tetap menerima teksnya saja.
func (h *TelegramHandler) pasangKeyboard(ctx context.Context, chatID, teks string) {
	if h.edit != nil {
		kb := telegram.KeyboardUtama()
		pesan := telegram.Message{Text: teks, Keyboard: &kb}
		if _, err := h.edit.SendMessageReturningID(ctx, chatID, pesan); err != nil {
			h.logInfo(ctx, "memasang keyboard menu gagal", err)
		}
		return
	}
	if h.sender != nil {
		_ = h.sender.Send(ctx, chatID, teks)
	}
}

// dashboardLink menyusun alamat dashboard untuk tombol tautan.
func (h *TelegramHandler) dashboardLink() string {
	return h.baseURL
}

func (h *TelegramHandler) logInfo(ctx context.Context, pesan string, err error) {
	if h.logger != nil && err != nil {
		h.logger.WarnContext(ctx, pesan, slog.String("error", err.Error()))
	}
}
