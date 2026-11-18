package webpush

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func decryptRFC8291ForTest(t *testing.T, body []byte, uaPriv *ecdh.PrivateKey, authSecret []byte) []byte {
	t.Helper()
	if len(body) < 86 {
		t.Fatalf("body terlalu pendek: %d", len(body))
	}
	salt := body[:16]
	rs := binary.BigEndian.Uint32(body[16:20])
	if rs != 4096 {
		t.Fatalf("record size tak terduga: %d", rs)
	}
	idLen := int(body[20])
	if idLen != 65 {
		t.Fatalf("idlen tak terduga: %d", idLen)
	}
	asPubBytes := body[21 : 21+idLen]
	ciphertext := body[21+idLen:]

	asPubKey, err := ecdh.P256().NewPublicKey(asPubBytes)
	if err != nil {
		t.Fatalf("public key server tidak sah: %v", err)
	}

	shared, err := uaPriv.ECDH(asPubKey)
	if err != nil {
		t.Fatalf("ECDH gagal: %v", err)
	}

	uaPubBytes := uaPriv.PublicKey().Bytes()
	keyInfo := append([]byte("WebPush: info\x00"), append(uaPubBytes, asPubBytes...)...)
	ikm, err := deriveHKDF(shared, authSecret, keyInfo, 32)
	if err != nil {
		t.Fatalf("derive IKM gagal: %v", err)
	}
	cek, err := deriveHKDF(ikm, salt, []byte("Content-Encoding: aes128gcm\x00"), 16)
	if err != nil {
		t.Fatalf("derive CEK gagal: %v", err)
	}
	nonce, err := deriveHKDF(ikm, salt, []byte("Content-Encoding: nonce\x00"), 12)
	if err != nil {
		t.Fatalf("derive nonce gagal: %v", err)
	}

	block, err := aes.NewCipher(cek)
	if err != nil {
		t.Fatalf("AES gagal: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("GCM gagal: %v", err)
	}

	record, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		t.Fatalf("dekripsi GCM gagal: %v", err)
	}
	if len(record) == 0 || record[len(record)-1] != 0x02 {
		t.Fatalf("delimiter rekaman akhir 0x02 tidak ditemukan")
	}
	return record[:len(record)-1]
}

func TestVAPIDAndEncryptRoundTrip(t *testing.T) {
	vapidPub, vapidPriv, err := GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("GenerateVAPIDKeys: %v", err)
	}

	uaPriv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ua key: %v", err)
	}
	authSecret := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, authSecret); err != nil {
		t.Fatalf("generate auth secret: %v", err)
	}

	uaPubB64 := base64.RawURLEncoding.EncodeToString(uaPriv.PublicKey().Bytes())
	authB64 := base64.RawURLEncoding.EncodeToString(authSecret)

	pesanAsli := []byte(`{"title":"XAUUSD Siap Entry","body":"BUY @ 2650.00","url":"/app/sinyal"}`)

	var receivedBody []byte
	var receivedAuth, receivedEncoding, receivedUrgency string

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedEncoding = r.Header.Get("Content-Encoding")
		receivedUrgency = r.Header.Get("Urgency")
		var readErr error
		receivedBody, readErr = io.ReadAll(r.Body)
		if readErr != nil {
			t.Errorf("baca body: %v", readErr)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	client := NewClient(0).WithHTTPClient(srv.Client())
	code, err := client.Send(context.Background(), Subscription{
		Endpoint: srv.URL + "/fcm/send/token-rahasia",
		P256dh:   uaPubB64,
		Auth:     authB64,
	}, pesanAsli, SendOptions{
		VAPIDPublicKey:    vapidPub,
		VAPIDPrivateKey:   vapidPriv,
		VAPIDSubject:      "mailto:admin@rekavia.com",
		TTLSeconds:        600,
		Urgency:           "high",
		SkipHostAllowlist: true,
	})
	if err != nil {
		t.Fatalf("Send gagal: %v", err)
	}
	if code != http.StatusCreated {
		t.Fatalf("status HTTP = %d, ingin 201", code)
	}
	if receivedEncoding != "aes128gcm" {
		t.Fatalf("Content-Encoding = %q, ingin aes128gcm", receivedEncoding)
	}
	if receivedUrgency != "high" {
		t.Fatalf("Urgency = %q, ingin high", receivedUrgency)
	}
	if !strings.HasPrefix(receivedAuth, "vapid t=") || !strings.Contains(receivedAuth, ", k="+vapidPub) {
		t.Fatalf("Authorization VAPID tidak sesuai: %q", receivedAuth)
	}

	hasilDekripsi := decryptRFC8291ForTest(t, receivedBody, uaPriv, authSecret)
	if !bytes.Equal(hasilDekripsi, pesanAsli) {
		t.Fatalf("hasil dekripsi = %q, ingin %q", string(hasilDekripsi), string(pesanAsli))
	}
}

func TestSendReturnsExpiredOn410(t *testing.T) {
	vapidPub, vapidPriv, _ := GenerateVAPIDKeys()
	uaPriv, _ := ecdh.P256().GenerateKey(rand.Reader)
	authSecret := make([]byte, 16)

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusGone)
	}))
	defer srv.Close()

	client := NewClient(0).WithHTTPClient(srv.Client())
	_, err := client.Send(context.Background(), Subscription{
		Endpoint: srv.URL + "/push/gone",
		P256dh:   base64.RawURLEncoding.EncodeToString(uaPriv.PublicKey().Bytes()),
		Auth:     base64.RawURLEncoding.EncodeToString(authSecret),
	}, []byte(`{}`), SendOptions{
		VAPIDPublicKey:    vapidPub,
		VAPIDPrivateKey:   vapidPriv,
		SkipHostAllowlist: true,
	})
	if !errors.Is(err, ErrSubscriptionExpired) {
		t.Fatalf("err = %v, ingin ErrSubscriptionExpired", err)
	}
}

func TestValidateEndpointAntiSSRF(t *testing.T) {
	sah := []string{
		"https://fcm.googleapis.com/fcm/send/abc123",
		"https://web.push.apple.com/QD12345",
		"https://updates.push.services.mozilla.com/wpush/v2/abc",
		"https://wns2-sg2p.notify.windows.com/w/?token=xyz",
	}
	for _, u := range sah {
		if err := ValidateEndpoint(u); err != nil {
			t.Errorf("endpoint sah ditolak (%s): %v", u, err)
		}
	}

	berbahaya := []string{
		"http://fcm.googleapis.com/fcm/send/abc",
		"https://127.0.0.1/fcm/send/abc",
		"https://169.254.169.254/latest/meta-data/",
		"https://runner:8081/internal/approve",
		"https://evil.example.com/fcm.googleapis.com",
		"https://fcm.googleapis.com:8443/fcm/send/abc",
		"https://fcm.googleapis.com/",
	}
	for _, u := range berbahaya {
		if err := ValidateEndpoint(u); err == nil {
			t.Errorf("endpoint berbahaya lolos (%s)", u)
		}
	}
}
