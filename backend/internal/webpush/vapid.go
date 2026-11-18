// Package webpush mengimplementasikan pengiriman Web Push terenkripsi sesuai
// standar RFC 8030 (Generic Event Delivery), RFC 8291 (Message Encryption /
// aes128gcm), dan RFC 8292 (VAPID Authentication) menggunakan pustaka standar Go.
package webpush

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// DefaultSubject dipakai bila VAPID_SUBJECT tidak disetel di environment.
	// Apple Web Push (web.push.apple.com) mewajibkan klaim `sub` berawalan
	// mailto: atau https:.
	DefaultSubject = "mailto:notifikasi@rekavia.com"
)

var (
	ErrInvalidVAPIDKey = errors.New("kunci VAPID tidak sah")
)

// GenerateVAPIDKeys membangkitkan pasangan kunci ECDSA P-256 dalam format
// base64url tanpa padding (public key 65 byte uncompressed, private key 32 byte scalar).
func GenerateVAPIDKeys() (publicKey, privateKey string, err error) {
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("membangkitkan kunci P-256: %w", err)
	}
	pubBytes := priv.PublicKey().Bytes() // 65 byte: 0x04 || X || Y
	privBytes := priv.Bytes()            // 32 byte scalar

	return base64.RawURLEncoding.EncodeToString(pubBytes),
		base64.RawURLEncoding.EncodeToString(privBytes),
		nil
}

// ParsePrivateKey mengurai private key VAPID base64url (32 byte) menjadi
// *ecdsa.PrivateKey sekaligus menurunkan public key uncompressed base64url-nya.
func ParsePrivateKey(privateKeyB64 string) (*ecdsa.PrivateKey, string, error) {
	rawPriv, err := decodeBase64Flexible(privateKeyB64)
	if err != nil || len(rawPriv) != 32 {
		return nil, "", ErrInvalidVAPIDKey
	}

	ecdhPriv, err := ecdh.P256().NewPrivateKey(rawPriv)
	if err != nil {
		return nil, "", ErrInvalidVAPIDKey
	}

	pubBytes := ecdhPriv.PublicKey().Bytes()
	if len(pubBytes) != 65 || pubBytes[0] != 0x04 {
		return nil, "", ErrInvalidVAPIDKey
	}

	x := new(big.Int).SetBytes(pubBytes[1:33])
	y := new(big.Int).SetBytes(pubBytes[33:65])
	d := new(big.Int).SetBytes(rawPriv)

	ecdsaKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     x,
			Y:     y,
		},
		D: d,
	}

	pubB64 := base64.RawURLEncoding.EncodeToString(pubBytes)
	return ecdsaKey, pubB64, nil
}

// NormalizeSubject memastikan klaim `sub` VAPID memenuhi syarat RFC 8292.
func NormalizeSubject(subject string) string {
	s := strings.TrimSpace(subject)
	if strings.HasPrefix(s, "mailto:") || strings.HasPrefix(s, "https://") {
		return s
	}
	return DefaultSubject
}

// BuildVAPIDHeader menyusun nilai header Authorization sesuai RFC 8292 §2.2:
// `vapid t=<JWT ES256>, k=<publicKeyBase64URL>`.
func BuildVAPIDHeader(endpointURL, subject, publicKeyB64, privateKeyB64 string, now time.Time) (string, error) {
	parsed, err := url.Parse(endpointURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", ErrInvalidEndpoint
	}

	privKey, derivedPubB64, err := ParsePrivateKey(privateKeyB64)
	if err != nil {
		return "", err
	}

	pubB64 := strings.TrimSpace(publicKeyB64)
	if pubB64 == "" {
		pubB64 = derivedPubB64
	} else {
		rawPub, decErr := decodeBase64Flexible(pubB64)
		if decErr != nil || len(rawPub) != 65 || rawPub[0] != 0x04 {
			return "", ErrInvalidVAPIDKey
		}
		pubB64 = base64.RawURLEncoding.EncodeToString(rawPub)
		if pubB64 != derivedPubB64 {
			return "", ErrInvalidVAPIDKey
		}
	}

	aud := parsed.Scheme + "://" + parsed.Host
	claims := jwt.MapClaims{
		"aud": aud,
		"exp": now.UTC().Add(12 * time.Hour).Unix(),
		"sub": NormalizeSubject(subject),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signed, err := token.SignedString(privKey)
	if err != nil {
		return "", fmt.Errorf("menandatangani token VAPID: %w", err)
	}

	return fmt.Sprintf("vapid t=%s, k=%s", signed, pubB64), nil
}

func decodeBase64Flexible(s string) ([]byte, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil, errors.New("base64 kosong")
	}
	// Ubah karakter standar menjadi URL-safe lalu buang padding '=' agar satu
	// decoder RawURLEncoding melayani seluruh variasi browser.
	normalized := strings.NewReplacer("+", "-", "/", "_").Replace(trimmed)
	normalized = strings.TrimRight(normalized, "=")
	return base64.RawURLEncoding.DecodeString(normalized)
}
