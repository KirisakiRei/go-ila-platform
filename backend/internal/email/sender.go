// Package email mengirim surel lewat antrean database.
//
// Pengiriman langsung ditolak: bila SMTP sedang mati, pesan harus tetap
// tersimpan dan dicoba ulang, bukan hilang (plans/17 §2.2).
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
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

// Send mengirim satu surel teks biasa.
func (s *SMTPSender) Send(ctx context.Context, msg Message) error {
	if !s.cfg.Enabled() {
		return fmt.Errorf("SMTP belum dikonfigurasi")
	}

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
	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("pengirim ditolak: %w", err)
	}
	if err := client.Rcpt(msg.To); err != nil {
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

// render menyusun surel dengan header minimal dan pengodean aman.
func (s *SMTPSender) render(msg Message) []byte {
	var b strings.Builder
	b.WriteString("From: " + s.cfg.From + "\r\n")
	b.WriteString("To: " + msg.To + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("\r\n")
	b.WriteString(msg.Body)
	b.WriteString("\r\n")
	return []byte(b.String())
}
