package email

import (
	"strings"
	"testing"

	"ila/backend/internal/model"
)

type mapTokenSource map[string]string

func (m mapTokenSource) TokenFor(id string) (string, bool) {
	v, ok := m[id]
	return v, ok
}

func TestRenderInviteApprovedHTMLAndEscaping(t *testing.T) {
	PublicBaseURL = "https://rekavia.com"
	row := model.EmailOutbox{
		ID:       "outbox-1",
		ToEmail:  "trader@example.com",
		Template: "invite_approved",
		Payload:  `{"name":"<script>alert(1)</script> Budi"}`,
	}

	msg, err := Render(row, mapTokenSource{"outbox-1": "INVITECODE123"})
	if err != nil {
		t.Fatalf("render gagal: %v", err)
	}
	wantLink := "https://rekavia.com/daftar?invite=INVITECODE123"
	if !strings.Contains(msg.Body, wantLink) {
		t.Fatalf("body plain-text tidak memuat link: %s", msg.Body)
	}
	if !strings.Contains(msg.HTML, wantLink) {
		t.Fatalf("HTML tidak memuat link: %s", msg.HTML)
	}
	if strings.Contains(msg.HTML, "<script>") {
		t.Fatalf("nama pemohon tidak ter-escape di HTML")
	}
	if !strings.Contains(msg.HTML, "Buat Akun dengan Undangan") {
		t.Fatalf("tombol CTA tidak muncul di HTML")
	}
}

func TestSMTPSenderMultipartAndEnvelopeFrom(t *testing.T) {
	addr, header := envelopeFrom("Imaginary Liquidity <favcreamyy@gmail.com>")
	if addr != "favcreamyy@gmail.com" {
		t.Fatalf("envelope address salah: got %q", addr)
	}
	if !strings.Contains(header, "Imaginary Liquidity") || !strings.Contains(header, "favcreamyy@gmail.com") {
		t.Fatalf("header From salah: got %q", header)
	}

	sender := NewSMTPSender(Config{
		Host: "smtp.gmail.com",
		Port: 587,
		From: "Imaginary Liquidity <favcreamyy@gmail.com>",
	})
	raw := string(sender.render(Message{
		To:      "user@example.com\r\nBcc: evil@example.com",
		Subject: "Undangan Akses",
		Body:    "Halo versi teks",
		HTML:    "<html><body>Halo versi HTML</body></html>",
	}))
	if !strings.Contains(raw, "Content-Type: multipart/alternative;") {
		t.Fatalf("expected multipart/alternative header, got:\n%s", raw)
	}
	if !strings.Contains(raw, "Content-Type: text/plain; charset=UTF-8") || !strings.Contains(raw, "Content-Type: text/html; charset=UTF-8") {
		t.Fatalf("expected both text/plain and text/html parts")
	}
	if strings.Contains(raw, "\r\nBcc: evil@example.com") {
		t.Fatalf("header injection CRLF harus disanitasi")
	}
}
