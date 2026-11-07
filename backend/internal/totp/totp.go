// Package totp mengimplementasikan TOTP (RFC 6238) di atas HOTP (RFC 4226)
// tanpa dependensi luar. Aplikasi authenticator mana pun bisa dipakai karena
// algoritmanya standar: HMAC-SHA1, 6 digit, periode 30 detik.
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Parameter tetap sesuai plans/05 §3.2.
const (
	Period = 30 * time.Second
	Digits = 6
	// Skew adalah toleransi ±1 periode: jam perangkat sering bergeser.
	// Lebih longgar dari ini memperbesar jendela serangan.
	Skew = 1
	// SecretBytes adalah panjang secret sebelum di-encode: 160 bit.
	SecretBytes = 20
	Issuer      = "Imaginary Liquidity"
)

// encodeBase32 dipakai tanpa padding agar cocok dengan yang diharapkan
// aplikasi authenticator.
var encodeBase32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewSecret membuat secret acak berformat base32.
func NewSecret() (string, error) {
	buf := make([]byte, SecretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("membuat secret 2FA: %w", err)
	}
	return encodeBase32.EncodeToString(buf), nil
}

// ProvisioningURL menyusun tautan otpauth:// yang dipindai aplikasi
// authenticator. Tidak ada server pihak ketiga yang dihubungi.
func ProvisioningURL(secret, account, issuer string) string {
	if issuer == "" {
		issuer = Issuer
	}
	label := url.PathEscape(issuer + ":" + account)

	params := url.Values{}
	params.Set("secret", secret)
	params.Set("issuer", issuer)
	params.Set("algorithm", "SHA1")
	params.Set("digits", fmt.Sprint(Digits))
	params.Set("period", fmt.Sprint(int(Period.Seconds())))

	return "otpauth://totp/" + label + "?" + params.Encode()
}

// Validate memeriksa kode 6 digit pada waktu tertentu, dengan toleransi ±1
// periode. Perbandingan memakai constant-time.
func Validate(secret, code string, at time.Time) (bool, error) {
	code = strings.TrimSpace(code)
	if len(code) != Digits {
		return false, nil
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false, nil
		}
	}

	counter := Step(at)
	for offset := -Skew; offset <= Skew; offset++ {
		expected, err := hotp(secret, uint64(counter+int64(offset)))
		if err != nil {
			return false, err
		}
		if subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return true, nil
		}
	}
	return false, nil
}

// Generate menghitung kode untuk waktu tertentu. Dipakai pengujian dan
// diagnostik; tidak pernah dipanggil pada jalur login.
func Generate(secret string, at time.Time) (string, error) {
	return hotp(secret, uint64(Step(at)))
}

// Step mengembalikan nomor periode pada saat tertentu. Dipakai menolak kode
// yang dipakai ulang pada jendela yang sama.
func Step(at time.Time) int64 {
	return at.UTC().Unix() / int64(Period.Seconds())
}

// hotp menghitung kode HOTP untuk satu counter.
func hotp(secret string, counter uint64) (string, error) {
	key, err := encodeBase32.DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", fmt.Errorf("secret 2FA tidak sah: %w", err)
	}

	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)

	// Dynamic truncation (RFC 4226 §5.3).
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])

	mod := uint32(1)
	for i := 0; i < Digits; i++ {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", Digits, value%mod), nil
}
