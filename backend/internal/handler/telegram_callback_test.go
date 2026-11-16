package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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

func (f *fakeTelegramEditor) SendMessageReturningID(_ context.Context, _ string, msg telegram.Message) (int64, error) {
	f.sent = append(f.sent, msg.Text)
	return 1, nil
}

func (f *fakeTelegramEditor) SendKeyboard(_ context.Context, _ string, text string, _ telegram.ReplyKeyboardMarkup) error {
	f.sent = append(f.sent, text)
	return nil
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
	payload, _ := json.Marshal(map[string]any{
		"callback_query": map[string]any{
			"id":   "cb-1",
			"data": data,
			"message": map[string]any{
				"message_id": 99,
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

func TestTeksKeyboardDikenali(t *testing.T) {
	for _, label := range []string{telegram.LabelPantau, telegram.LabelRiwayat, telegram.LabelHariIni, telegram.LabelPengaturan} {
		if _, kenal := telegram.AksiDariTeks(label); !kenal {
			t.Fatalf("label %q harus dikenali sebagai menu", label)
		}
	}
	if _, kenal := telegram.AksiDariTeks("halo apa kabar"); kenal {
		t.Fatal("teks biasa tidak boleh dianggap perintah menu")
	}
}
