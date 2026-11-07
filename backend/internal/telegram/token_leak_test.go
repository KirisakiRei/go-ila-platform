package telegram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const fakeToken = "123456:TOKEN-SANGAT-RAHASIA"

// proxyGagal meneruskan ke host yang menolak koneksi sehingga http.Do gagal
// dan pesan error memuat URL lengkap beserta token.
func proxyGagal(t *testing.T) *Client {
	t.Helper()

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Menutup koneksi tanpa balasan membuat klien menerima kegagalan jaringan.
		hj, _ := w.(http.Hijacker)
		conn, _, err := hj.Hijack()
		if err == nil {
			_ = conn.Close()
		}
	}))
	t.Cleanup(bad.Close)

	client := NewClient(Config{BotToken: fakeToken, Timeout: 3 * time.Second, MinInterval: time.Millisecond}, nil)
	client.cfg.BotToken = fakeToken
	return client
}

// TestSendTidakMembocorkanToken memastikan error yang dikembalikan tidak
// memuat token bot, karena error itu dicatat ke log dan ke database.
func TestSendTidakMembocorkanToken(t *testing.T) {
	client := proxyGagal(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := client.Send(ctx, "999", "halo")
	if err == nil {
		t.Fatal("ingin error karena koneksi diputus")
	}

	if strings.Contains(err.Error(), fakeToken) {
		t.Fatalf("token bocor di pesan error: %s", err.Error())
	}
}

// TestSanitizeTransportErrorMenggantiToken adalah inti perbaikan: pesan yang
// memuat token diganti penanda sebelum keluar dari paket ini.
func TestSanitizeTransportErrorMenggantiToken(t *testing.T) {
	original := &stubError{msg: `Post "https://api.telegram.org/bot` + fakeToken + `/sendMessage": dial tcp: refused`}

	cleaned := sanitizeTransportError(original, fakeToken)
	if cleaned == nil {
		t.Fatal("hasil sanitasi tidak boleh nil")
	}
	if strings.Contains(cleaned.Error(), fakeToken) {
		t.Fatalf("token masih ada: %s", cleaned.Error())
	}
	if !strings.Contains(cleaned.Error(), "<token-disembunyikan>") {
		t.Fatalf("token seharusnya diganti penanda, dapat: %s", cleaned.Error())
	}
}

// TestSanitizeTransportErrorNil aman terhadap error kosong.
func TestSanitizeTransportErrorNil(t *testing.T) {
	if got := sanitizeTransportError(nil, fakeToken); got != nil {
		t.Fatalf("ingin nil, dapat %v", got)
	}
}

type stubError struct{ msg string }

func (e *stubError) Error() string { return e.msg }
