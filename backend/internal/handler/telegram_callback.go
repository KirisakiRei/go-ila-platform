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

	// Acknowledge before database or broker work so the Telegram spinner ends promptly.
	if err := h.edit.AnswerCallback(ctx, cb.ID, ""); err != nil {
		h.logInfo(ctx, "menjawab tombol Telegram gagal", err)
	}
	if cb.Data == telegram.CallbackNoop {
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
		if h.approve == nil {
			h.kirimTeks(ctx, chatID, "Konfirmasi entry belum tersedia. Silakan buka dashboard.")
			return
		}
		h.confirmEntry(ctx, user, chatID, messageID, strings.TrimPrefix(aksi, telegram.CallbackEksekusi))
	case strings.HasPrefix(aksi, telegram.CallbackLewati):
		if h.approve == nil {
			h.kirimTeks(ctx, chatID, "Keputusan entry belum dapat diproses. Silakan buka dashboard.")
			return
		}
		h.declineEntry(ctx, user, chatID, messageID, strings.TrimPrefix(aksi, telegram.CallbackLewati))
	case strings.HasPrefix(aksi, telegram.CallbackRiwayat):
		if !butuhMenu() {
			return
		}
		offset, _ := strconv.Atoi(strings.TrimPrefix(aksi, telegram.CallbackRiwayat))
		h.gantiPesan(ctx, chatID, messageID, func() (telegram.Message, error) {
			return h.menu.Riwayat(ctx, user, offset)
		})
	case strings.HasPrefix(aksi, telegram.CallbackSimulasi):
		reader, ok := h.menu.(interface {
			Simulasi(context.Context, *model.User, string, int) (telegram.Message, error)
		})
		if !ok {
			h.kirimTeks(ctx, chatID, "Riwayat simulasi belum tersedia. Silakan buka dashboard.")
			return
		}
		page, _ := strconv.Atoi(strings.TrimPrefix(aksi, telegram.CallbackSimulasi))
		h.gantiPesan(ctx, chatID, messageID, func() (telegram.Message, error) { return reader.Simulasi(ctx, user, h.dashboardLink(), page) })
	case strings.HasPrefix(aksi, "view:"):
		reader, ok := h.menu.(interface {
			Detail(context.Context, *model.User, string, string, string) (telegram.Message, error)
		})
		if !ok {
			return
		}
		parts := strings.SplitN(aksi, ":", 3)
		if len(parts) != 3 {
			return
		}
		h.gantiPesan(ctx, chatID, messageID, func() (telegram.Message, error) {
			return reader.Detail(ctx, user, h.dashboardLink(), parts[1], parts[2])
		})
	case aksi == telegram.CallbackBantuan:
		h.gantiPesan(ctx, chatID, messageID, func() (telegram.Message, error) {
			msg := telegram.FormatBantuanMessage()
			if h.baseURL != "" {
				msg.Buttons = append([][]telegram.InlineButton{{{Text: "Laporkan masalah", URL: telegram.AppURL(h.baseURL, "/laporkan-bug"), Style: "primary"}}}, msg.Buttons...)
			}
			return msg, nil
		})
	case aksi == telegram.CallbackPanduan:
		h.gantiPesan(ctx, chatID, messageID, func() (telegram.Message, error) { return telegram.FormatPanduanMessage(), nil })
	case aksi == telegram.CallbackRiwayatMenu:
		h.gantiPesan(ctx, chatID, messageID, func() (telegram.Message, error) { return telegram.FormatRiwayatMenu(), nil })
	case aksi == telegram.CallbackMenu:
		h.gantiPesan(ctx, chatID, messageID, func() (telegram.Message, error) {
			return telegram.FormatMenuMessage(namaTampil(user), paketTampil(user), h.dashboardLink()), nil
		})
	case aksi == telegram.CallbackPantau || aksi == telegram.CallbackSegarkan:
		if !butuhMenu() {
			return
		}
		h.gantiPesan(ctx, chatID, messageID, func() (telegram.Message, error) {
			return h.menu.Pantau(ctx, user, h.dashboardLink(), 1)
		})
	case strings.HasPrefix(aksi, telegram.CallbackPantauHalaman):
		if !butuhMenu() {
			return
		}
		halaman, _ := strconv.Atoi(strings.TrimPrefix(cb.Data, telegram.CallbackPantauHalaman))
		h.gantiPesan(ctx, chatID, messageID, func() (telegram.Message, error) {
			return h.menu.Pantau(ctx, user, h.dashboardLink(), halaman)
		})
	case aksi == telegram.CallbackHariIni:
		if !butuhMenu() {
			return
		}
		h.gantiPesan(ctx, chatID, messageID, func() (telegram.Message, error) {
			return h.menu.HariIni(ctx, user, h.dashboardLink())
		})
	case aksi == telegram.CallbackPengaturan:
		h.gantiPesan(ctx, chatID, messageID, func() (telegram.Message, error) {
			return telegram.FormatPengaturanMessage(h.pengaturanLink()), nil
		})
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
		h.ketikPesan(ctx, chatID, messageID, telegram.Message{Text: "Halaman belum dapat dimuat. Silakan coba kembali melalui menu.", Buttons: [][]telegram.InlineButton{{{Text: telegram.LabelMenu, CallbackData: telegram.CallbackMenu}}}}, [][]telegram.InlineButton{{{Text: telegram.LabelMenu, CallbackData: telegram.CallbackMenu}}})
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
		if strings.Contains(err.Error(), "message is not modified") {
			return
		}
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
	// Panduan dan menu utama tidak menyentuh data akun, jadi tetap dilayani
	// walau chat belum terhubung. Hanya halaman berisi data yang ditolak.
	switch aksi {
	case telegram.CallbackBantuan:
		h.kirimPesanMenu(ctx, chatID, telegram.FormatBantuanMessage())
		return
	case telegram.CallbackPengaturan:
		h.kirimPesanMenu(ctx, chatID, telegram.FormatPengaturanMessage(h.pengaturanLink()))
		return
	}

	if h.menu == nil {
		h.kirimPesanMenu(ctx, chatID, telegram.FormatMenuMessage("", "", h.dashboardLink()))
		return
	}

	user, err := h.telegram.ResolveChat(ctx, chatID)
	if err != nil {
		h.kirimTeks(ctx, chatID, "Chat ini belum terhubung ke akun BOT Trading ILA. Hubungkan lewat menu Pengaturan di dashboard, lalu buka /start dari tautan yang tersedia.")
		return
	}

	var pesan telegram.Message
	switch {
	case aksi == telegram.CallbackPantau:
		pesan, err = h.menu.Pantau(ctx, user, h.dashboardLink(), 1)
	case aksi == telegram.CallbackHariIni:
		pesan, err = h.menu.HariIni(ctx, user, h.dashboardLink())
	case aksi == telegram.CallbackRiwayatMenu:
		pesan = telegram.FormatRiwayatMenu()
	case strings.HasPrefix(aksi, telegram.CallbackRiwayat):
		pesan, err = h.menu.Riwayat(ctx, user, 0)
	default:
		pesan = telegram.FormatMenuMessage(namaTampil(user), paketTampil(user), h.dashboardLink())
	}
	if err != nil {
		h.logInfo(ctx, "menyusun halaman menu gagal", err)
		pesan = telegram.Message{Text: "Halaman belum bisa dimuat. Coba lagi sebentar lagi."}
	}
	h.kirimPesanMenu(ctx, chatID, pesan)
}

// kirimPesanMenu mengirim halaman menu sebagai pesan baru.
//
// Tombol inline dibiarkan menempel di balon chat. Keyboard bawah layar tidak
// ikut dikirim karena Telegram hanya menerima satu reply_markup per pesan.
func (h *TelegramHandler) kirimPesanMenu(ctx context.Context, chatID string, pesan telegram.Message) {
	if h.edit == nil {
		if h.sender != nil {
			_ = h.sender.Send(ctx, chatID, pesan.Text)
		}
		return
	}
	if _, err := h.edit.SendMessageReturningID(ctx, chatID, pesan); err != nil {
		h.logInfo(ctx, "mengirim halaman menu gagal", err)
	}
}

// namaTampil mengambil nama depan untuk sapaan.
//
// Nama diambil apa adanya dari akun, dipotong di spasi pertama supaya sapaan
// tetap pendek di layar ponsel.
func namaTampil(user *model.User) string {
	if user == nil {
		return ""
	}
	nama := strings.TrimSpace(user.Name)
	if nama == "" {
		return strings.TrimSpace(user.Username)
	}
	if depan, _, ada := strings.Cut(nama, " "); ada {
		return depan
	}
	return nama
}

// paketTampil menyusun keterangan paket untuk halaman menu.
func paketTampil(user *model.User) string {
	if user == nil {
		return ""
	}
	tier := strings.ToUpper(strings.TrimSpace(user.Tier))
	switch tier {
	case "", "FREE":
		return "Free"
	case "TRIAL":
		return "Trial"
	}
	return strings.ToUpper(tier[:1]) + tier[1:]
}

// kirimTeks mengirim balasan singkat dengan tombol menu utama.
//
// Tombol ditempel di balon chat agar trader selalu punya jalan kembali ke
// menu tanpa mengetik perintah.
func (h *TelegramHandler) kirimTeks(ctx context.Context, chatID, teks string) {
	h.kirimPesanMenu(ctx, chatID, telegram.Message{
		Text:    teks,
		Buttons: [][]telegram.InlineButton{{{Text: telegram.LabelMenu, CallbackData: telegram.CallbackMenu}}},
	})
}

// dashboardLink menyusun alamat dashboard untuk tombol tautan.
// dashboardLink menunjuk halaman utama aplikasi web.
func (h *TelegramHandler) dashboardLink() string {
	return telegram.AppURL(h.baseURL, "/dashboard")
}

// pengaturanLink menunjuk halaman pengaturan akun di web.
func (h *TelegramHandler) pengaturanLink() string {
	return telegram.AppURL(h.baseURL, "/pengaturan")
}

func (h *TelegramHandler) logInfo(ctx context.Context, pesan string, err error) {
	if h.logger != nil && err != nil {
		h.logger.WarnContext(ctx, pesan, slog.String("error", err.Error()))
	}
}
