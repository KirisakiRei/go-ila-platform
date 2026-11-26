package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/model"
	"ila/backend/internal/service"
	"ila/backend/internal/telegram"
)

// fakeTelegramEditor mencatat pemanggilan penggantian pesan dan callback.
type fakeTelegramEditor struct {
	answered []string
	edits    []string
	sent     []string
}

func (f *fakeTelegramEditor) Send(_ context.Context, chatID, text string) error {
	f.sent = append(f.sent, text)
	return nil
}

func (f *fakeTelegramEditor) AnswerCallback(_ context.Context, callbackID, _ string) error {
	f.answered = append(f.answered, callbackID)
	return nil
}

func (f *fakeTelegramEditor) EditMessageText(_ context.Context, _ string, _ int64, text string, _ [][]telegram.InlineButton) error {
	f.edits = append(f.edits, text)
	return nil
}

func (f *fakeTelegramEditor) SendChatAction(_ context.Context, _, _ string) error {
	return nil
}

func (f *fakeTelegramEditor) SendMessageReturningID(_ context.Context, _ string, msg telegram.Message) (int64, error) {
	f.sent = append(f.sent, msg.Text)
	return 1, nil
}

// fakeApprove hanya mencatat token yang dipakai.
type fakeApprove struct {
	sudahDipanggil bool
	hasil          *service.ApproveResult
	err            error
}

func (f *fakeApprove) Resolve(context.Context, string, string) (*service.ApprovePreview, error) {
	return nil, nil
}

func (f *fakeApprove) Confirm(context.Context, string, string) (*service.ApproveResult, error) {
	f.sudahDipanggil = true
	return f.hasil, f.err
}

func (f *fakeApprove) Decline(context.Context, string, string) (*service.ApproveResult, error) {
	f.sudahDipanggil = true
	return &service.ApproveResult{Status: model.SignalStatusSkipped}, f.err
}

func (f *fakeApprove) Issue(context.Context, string, string) (*service.ApproveIssue, error) {
	return nil, nil
}

func kirimCallback(t *testing.T, h *TelegramHandler, data string, chatID int64) *httptest.ResponseRecorder {
	t.Helper()
	return kirimCallbackText(t, h, data, chatID, "")
}

func kirimCallbackText(t *testing.T, h *TelegramHandler, data string, chatID int64, text string) *httptest.ResponseRecorder {
	t.Helper()
	payload, _ := json.Marshal(map[string]any{
		"callback_query": map[string]any{
			"id":   "cb-1",
			"data": data,
			"message": map[string]any{
				"message_id": 99,
				"text":       text,
				"chat":       map[string]any{"id": chatID},
			},
		},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r := gin.New()
	r.POST("/webhook", h.Webhook)
	r.ServeHTTP(w, req)
	return w
}

func TestCallbackEksekusiMemakaiToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sender := &fakeTelegramEditor{}
	svc := &fakeTelegramService{
		resolveChatFn: func(context.Context, string) (*model.User, error) {
			return &model.User{Base: model.Base{ID: "u-1"}}, nil
		},
	}
	approve := &fakeApprove{hasil: &service.ApproveResult{Status: model.SignalStatusConsumed}}
	h := NewTelegramHandler(svc, "bot", "", sender).WithMenu(nil, approve, "", nil)

	w := kirimCallback(t, h, telegram.CallbackEksekusi+"token-abc", 123)
	if w.Code != http.StatusOK {
		t.Fatalf("webhook harus membalas 200, dapat %d", w.Code)
	}
	if !approve.sudahDipanggil {
		t.Fatal("tombol eksekusi harus memanggil konfirmasi")
	}
	if len(sender.edits) == 0 {
		t.Fatal("pesan harus diganti setelah tombol ditekan")
	}
	if len(sender.answered) == 0 {
		t.Fatal("callback harus selalu dijawab agar spinner berhenti")
	}
}

func TestCallbackChatAsingTidakMengeksekusi(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sender := &fakeTelegramEditor{}
	svc := &fakeTelegramService{
		resolveChatFn: func(context.Context, string) (*model.User, error) {
			return nil, errors.New("tidak terhubung")
		},
	}
	approve := &fakeApprove{hasil: &service.ApproveResult{Status: model.SignalStatusConsumed}}
	h := NewTelegramHandler(svc, "bot", "", sender).WithMenu(nil, approve, "", nil)

	kirimCallback(t, h, telegram.CallbackEksekusi+"token-abc", 999)

	if approve.sudahDipanggil {
		t.Fatal("chat yang belum terhubung tidak boleh memicu eksekusi")
	}
	if len(sender.answered) == 0 {
		t.Fatal("callback tetap harus dijawab")
	}
}

func TestCallbackEksekusiDitolakMenampilkanAlasan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sender := &fakeTelegramEditor{}
	svc := &fakeTelegramService{
		resolveChatFn: func(context.Context, string) (*model.User, error) {
			return &model.User{Base: model.Base{ID: "u-1"}}, nil
		},
	}
	approve := &fakeApprove{hasil: &service.ApproveResult{
		Status:  model.SignalStatusRejected,
		Message: "Harga sudah bergerak terlalu jauh.",
	}}
	h := NewTelegramHandler(svc, "bot", "", sender).WithMenu(nil, approve, "", nil)

	kirimCallback(t, h, telegram.CallbackEksekusi+"token-abc", 123)

	if len(sender.edits) == 0 {
		t.Fatal("alasan penolakan harus tampil di pesan")
	}
	if !bytes.Contains([]byte(sender.edits[0]), []byte("terlalu jauh")) {
		t.Fatalf("alasan penolakan hilang dari pesan: %q", sender.edits[0])
	}
}

func TestCallbackRetainsSetupCycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, action, status string
		err                  error
	}{
		{"confirmed", telegram.CallbackEksekusi + "token-abc", model.SignalStatusConsumed, nil},
		{"rejected", telegram.CallbackEksekusi + "token-abc", model.SignalStatusRejected, nil},
		{"confirm-expired", telegram.CallbackEksekusi + "token-abc", "", errors.New("expired")},
		{"confirm-empty", telegram.CallbackEksekusi, "", nil},
		{"declined", telegram.CallbackLewati + "token-abc", model.SignalStatusSkipped, nil},
		{"decline-expired", telegram.CallbackLewati + "token-abc", "", errors.New("expired")},
		{"decline-empty", telegram.CallbackLewati, "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sender := &fakeTelegramEditor{}
			svc := &fakeTelegramService{resolveChatFn: func(context.Context, string) (*model.User, error) {
				return &model.User{Base: model.Base{ID: "u-1"}}, nil
			}}
			approve := &fakeApprove{hasil: &service.ApproveResult{Status: tc.status, Message: "Ditolak"}, err: tc.err}
			h := NewTelegramHandler(svc, "bot", "", sender).WithMenu(nil, approve, "", nil)
			kirimCallbackText(t, h, tc.action, 123, "🟢 EURUSD · H4 ➔ M1\nTahap 4/6 · Siap masuk\n")
			if len(sender.edits) != 1 || !strings.Contains(sender.edits[0], "EURUSD · H4 ➔ M1\n") || !strings.Contains(sender.edits[0], "Setup: H4 ➔ M1\nCycle: DAILY\n") {
				t.Fatalf("konteks callback hilang: %v", sender.edits)
			}
			if strings.Contains(sender.edits[0], "token-abc") {
				t.Fatal("token tidak boleh muncul di pesan")
			}
		})
	}
}

type menuNavigationFake struct{ page int }

func (m *menuNavigationFake) Pantau(ctx context.Context, u *model.User, link string, page int) (telegram.Message, error) {
	m.page = page
	return telegram.Message{Text: "Pantau"}, nil
}
func (*menuNavigationFake) Riwayat(context.Context, *model.User, int) (telegram.Message, error) {
	return telegram.Message{Text: "Riwayat"}, nil
}
func (*menuNavigationFake) HariIni(context.Context, *model.User, string) (telegram.Message, error) {
	return telegram.Message{Text: "Hari ini"}, nil
}
func TestPantauPaginationAndHelpWorkWithoutApprovalService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sender := &fakeTelegramEditor{}
	svc := &fakeTelegramService{resolveChatFn: func(context.Context, string) (*model.User, error) { return &model.User{Base: model.Base{ID: "u"}}, nil }}
	menu := &menuNavigationFake{}
	h := NewTelegramHandler(svc, "bot", "", sender).WithMenu(menu, nil, "https://rekavia.com", nil)
	kirimCallback(t, h, "ph:2", 123)
	if menu.page != 2 || len(sender.edits) != 1 {
		t.Fatalf("pagination ignored: page=%d edits=%v", menu.page, sender.edits)
	}
	kirimCallback(t, h, telegram.CallbackBantuan, 123)
	if len(sender.edits) != 2 {
		t.Fatal("inline help ignored")
	}
	kirimCallback(t, h, telegram.CallbackNoop, 123)
	if len(sender.edits) != 2 || len(sender.answered) != 3 {
		t.Fatalf("no-op must acknowledge without editing: %+v", sender)
	}
}
