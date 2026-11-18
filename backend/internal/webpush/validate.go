package webpush

import (
	"crypto/ecdh"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"strings"
)

var (
	ErrInvalidEndpoint = errors.New("endpoint Web Push tidak sah atau berasal dari penyedia yang tidak didukung")
)

// ValidateEndpoint memverifikasi bahwa URL endpoint berasal dari gateway Push
// resmi browser (Google FCM, Apple APNs Web Push, Mozilla Autopush, Microsoft
// WNS, Samsung Push) melalui HTTPS murni untuk menutup celah SSRF.
func ValidateEndpoint(rawURL string) error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" || len(trimmed) > 2048 {
		return ErrInvalidEndpoint
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ErrInvalidEndpoint
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return ErrInvalidEndpoint
	}
	if parsed.User != nil || parsed.RawQuery != "" && strings.Contains(parsed.RawQuery, "@") {
		return ErrInvalidEndpoint
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return ErrInvalidEndpoint
	}

	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "" {
		return ErrInvalidEndpoint
	}
	// Tolak seluruh literal alamat IP (IPv4/IPv6).
	if net.ParseIP(host) != nil {
		return ErrInvalidEndpoint
	}

	if !isAllowedPushHost(host) {
		return ErrInvalidEndpoint
	}
	if parsed.Path == "" || parsed.Path == "/" {
		return ErrInvalidEndpoint
	}
	return nil
}

func isAllowedPushHost(host string) bool {
	exact := []string{
		"fcm.googleapis.com",
		"jmt17.google.com",
		"android.googleapis.com",
		"web.push.apple.com",
		"updates.push.services.mozilla.com",
	}
	for _, allowed := range exact {
		if host == allowed {
			return true
		}
	}

	suffixes := []string{
		".fcm.googleapis.com",
		".push.apple.com",
		".push.services.mozilla.com",
		".notify.windows.com",
		".push.samsungosp.com",
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(host, suffix) && len(host) > len(suffix) {
			return true
		}
	}
	return false
}

// ValidateSubscriptionKeys memeriksa kunci p256dh (65 byte) dan auth (16 byte)
// lalu mengembalikannya dalam format kanonis base64url tanpa padding.
func ValidateSubscriptionKeys(p256dh, auth string) (normP256dh, normAuth string, err error) {
	pubBytes, err := decodeBase64Flexible(p256dh)
	if err != nil || len(pubBytes) != 65 {
		return "", "", ErrInvalidClientKey
	}
	if _, err := ecdh.P256().NewPublicKey(pubBytes); err != nil {
		return "", "", ErrInvalidClientKey
	}

	authBytes, err := decodeBase64Flexible(auth)
	if err != nil || len(authBytes) != 16 {
		return "", "", ErrInvalidClientKey
	}

	return base64.RawURLEncoding.EncodeToString(pubBytes),
		base64.RawURLEncoding.EncodeToString(authBytes),
		nil
}

// EndpointHash menghasilkan sidik jari SHA-256 hex (64 karakter) untuk indeks unik.
func EndpointHash(endpoint string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(endpoint)))
	return hex.EncodeToString(sum[:])
}

// RedactEndpoint menyamarkan path rahasia pada URL endpoint agar aman ditulis ke log.
func RedactEndpoint(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return "[redacted-endpoint]"
	}
	return parsed.Scheme + "://" + parsed.Host + "/[redacted]"
}
