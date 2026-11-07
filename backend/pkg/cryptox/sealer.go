package cryptox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Sealer mengenkripsi data tunggal tanpa kunci per-akun, dipakai untuk secret
// yang terkait langsung dengan satu identitas (mis. secret TOTP).
//
// Berbeda dengan Keyring yang menyimpan kunci pembungkus per akun, di sini
// identitas pemilik diikat sebagai associated data sehingga ciphertext satu
// akun tidak bisa dipakai akun lain.
type Sealer struct {
	key     []byte
	version int
}

// NewSealer menyiapkan sealer dari kunci base64 32 byte.
func NewSealer(version int, rawKey string) (*Sealer, error) {
	key, err := decodeKey(rawKey)
	if err != nil {
		return nil, err
	}
	return &Sealer{key: key, version: version}, nil
}

// Version mengembalikan versi kunci yang dipakai.
func (s *Sealer) Version() int {
	return s.version
}

// SealString mengenkripsi teks dan mengembalikan ciphertext base64.
// ownerID diikat ke ciphertext: membukanya butuh ownerID yang sama.
func (s *Sealer) SealString(plaintext, ownerID string) (string, error) {
	aead, err := s.aead()
	if err != nil {
		return "", err
	}

	ciphertext, nonce, err := sealWith(aead, []byte(plaintext), []byte(ownerID))
	if err != nil {
		return "", err
	}

	// Nonce disimpan bersama ciphertext agar pembukaan tidak butuh data lain.
	payload := make([]byte, 0, len(nonce)+len(ciphertext))
	payload = append(payload, nonce...)
	payload = append(payload, ciphertext...)
	return base64.StdEncoding.EncodeToString(payload), nil
}

// OpenString membuka ciphertext base64 milik ownerID.
func (s *Sealer) OpenString(encoded, ownerID string) (string, error) {
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrDecrypt
	}

	aead, err := s.aead()
	if err != nil {
		return "", err
	}
	if len(payload) <= aead.NonceSize() {
		return "", ErrDecrypt
	}

	nonce := payload[:aead.NonceSize()]
	ciphertext := payload[aead.NonceSize():]

	plaintext, err := openWith(aead, ciphertext, nonce, []byte(ownerID))
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (s *Sealer) aead() (cipher.AEAD, error) {
	if len(s.key) != KeySize {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, fmt.Errorf("membuat cipher: %w", err)
	}
	return cipher.NewGCM(block)
}

// NewRandomBytes membuat nilai acak sepanjang n byte.
func NewRandomBytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return nil, fmt.Errorf("membuat nilai acak: %w", err)
	}
	return buf, nil
}
