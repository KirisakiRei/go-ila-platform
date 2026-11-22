// Package email mengirim surel lewat antrean database.
//
// Pengiriman langsung ditolak: bila SMTP sedang mati, pesan harus tetap
// tersimpan dan dicoba ulang, bukan hilang (plans/17 §2.2).
package email

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

// Config adalah pengaturan SMTP.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Enabled menandai SMTP terkonfigurasi.
func (c Config) Enabled() bool {
	return strings.TrimSpace(c.Host) != "" && strings.TrimSpace(c.From) != ""
}

// Message adalah satu surel siap kirim.
type Message struct {
	To      string
	Subject string
	Body    string
	HTML    string
}

// Sender mengirim satu surel.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// SMTPSender mengirim lewat SMTP dengan STARTTLS bila server mendukungnya.
type SMTPSender struct {
	cfg     Config
	timeout time.Duration
}

// NewSMTPSender membuat pengirim SMTP. Timeout mencegah worker menggantung.
func NewSMTPSender(cfg Config) *SMTPSender {
	return &SMTPSender{cfg: cfg, timeout: 15 * time.Second}
}

// envelopeFrom mengekstrak alamat surel murni dari format "Nama <email@domain>"
// karena perintah SMTP MAIL FROM hanya menerima alamat tanpa nama tampilan.
func envelopeFrom(raw string) (string, string) {
	trimmed := strings.TrimSpace(raw)
	if parsed, err := mail.ParseAddress(trimmed); err == nil && parsed.Address != "" {
		if parsed.Name != "" {
			return parsed.Address, parsed.String()
		}
		return parsed.Address, parsed.Address
	}
	return trimmed, trimmed
}

func sanitizeHeader(v string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(strings.TrimSpace(v))
}

// Send mengirim satu surel (plain-text atau multipart HTML + plain-text).
func (s *SMTPSender) Send(ctx context.Context, msg Message) error {
	if !s.cfg.Enabled() {
		return fmt.Errorf("SMTP belum dikonfigurasi")
	}
	fromAddr, _ := envelopeFrom(s.cfg.From)
	toAddr := sanitizeHeader(msg.To)

	addr := net.JoinHostPort(s.cfg.Host, fmt.Sprint(s.cfg.Port))
	dialer := &net.Dialer{Timeout: s.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("menghubungi SMTP: %w", err)
	}
	_ = conn.SetDeadline(time.Now().Add(s.timeout))
	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("membuka sesi SMTP: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Port 587 umumnya menawarkan STARTTLS; wajib dipakai bila tersedia.
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("STARTTLS: %w", err)
		}
	}
	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("autentikasi SMTP: %w", err)
		}
	}
	if err := client.Mail(fromAddr); err != nil {
		return fmt.Errorf("pengirim ditolak: %w", err)
	}
	if err := client.Rcpt(toAddr); err != nil {
		return fmt.Errorf("penerima ditolak: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("membuka isi surel: %w", err)
	}
	if _, err := writer.Write(s.render(msg)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("menulis isi surel: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("menutup isi surel: %w", err)
	}
	return client.Quit()
}

func randomBoundary() string {
	var buf [12]byte
	_, _ = rand.Read(buf[:])
	return "ila-" + hex.EncodeToString(buf[:])
}

// render menyusun surel dengan header aman dan format multipart/alternative bila HTML tersedia.
func (s *SMTPSender) render(msg Message) []byte {
	_, fromHeader := envelopeFrom(s.cfg.From)
	toHeader := sanitizeHeader(msg.To)
	subjectHeader := mime.QEncoding.Encode("UTF-8", sanitizeHeader(msg.Subject))

	var b strings.Builder
	b.WriteString("From: " + fromHeader + "\r\n")
	b.WriteString("To: " + toHeader + "\r\n")
	b.WriteString("Subject: " + subjectHeader + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n")

	if strings.TrimSpace(msg.HTML) == "" {
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		b.WriteString("\r\n")
		b.WriteString(msg.Body)
		b.WriteString("\r\n")
		return []byte(b.String())
	}

	boundary := randomBoundary()
	b.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", boundary))
	b.WriteString("\r\n")
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(msg.Body)
	b.WriteString("\r\n")
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(msg.HTML)
	b.WriteString("\r\n")
	b.WriteString("--" + boundary + "--\r\n")
	return []byte(b.String())
}
