package webpush

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

const (
	// MaxPayloadBytes adalah batas aman muatan plaintext sebelum enkripsi agar
	// hasil akhir selalu di bawah batas 4096 byte standar RFC 8291.
	MaxPayloadBytes = 3800
	recordSize      = uint32(4096)
)

var (
	ErrPayloadTooLarge = errors.New("muatan notifikasi melebihi batas ukuran Web Push")
	ErrInvalidClientKey = errors.New("kunci enkripsi perangkat (p256dh/auth) tidak sah")
)

// EncryptPayload mengenkripsi plaintext sesuai RFC 8291 (Content-Encoding: aes128gcm).
func EncryptPayload(plaintext []byte, clientP256dhB64, clientAuthB64 string) ([]byte, error) {
	if len(plaintext) > MaxPayloadBytes {
		return nil, ErrPayloadTooLarge
	}

	uaPubBytes, err := decodeBase64Flexible(clientP256dhB64)
	if err != nil || len(uaPubBytes) != 65 {
		return nil, ErrInvalidClientKey
	}
	uaPubKey, err := ecdh.P256().NewPublicKey(uaPubBytes)
	if err != nil {
		return nil, ErrInvalidClientKey
	}

	authSecret, err := decodeBase64Flexible(clientAuthB64)
	if err != nil || len(authSecret) != 16 {
		return nil, ErrInvalidClientKey
	}

	asPrivKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("membangkitkan kunci ephemeral P-256: %w", err)
	}
	asPubBytes := asPrivKey.PublicKey().Bytes()

	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("membangkitkan salt enkripsi: %w", err)
	}

	return encryptWithParams(plaintext, uaPubKey, authSecret, asPrivKey, asPubBytes, salt)
}

func encryptWithParams(
	plaintext []byte,
	uaPubKey *ecdh.PublicKey,
	authSecret []byte,
	asPrivKey *ecdh.PrivateKey,
	asPubBytes []byte,
	salt []byte,
) ([]byte, error) {
	sharedSecret, err := asPrivKey.ECDH(uaPubKey)
	if err != nil {
		return nil, fmt.Errorf("menghitung shared secret ECDH: %w", err)
	}

	uaPubBytes := uaPubKey.Bytes()

	// RFC 8291 §3.3: IKM = HKDF-SHA256(secret=ecdh_secret, salt=auth_secret, info="WebPush: info\0" || ua_public || as_public, L=32)
	keyInfo := make([]byte, 0, 14+len(uaPubBytes)+len(asPubBytes))
	keyInfo = append(keyInfo, []byte("WebPush: info\x00")...)
	keyInfo = append(keyInfo, uaPubBytes...)
	keyInfo = append(keyInfo, asPubBytes...)

	ikm, err := deriveHKDF(sharedSecret, authSecret, keyInfo, 32)
	if err != nil {
		return nil, err
	}

	// RFC 8188 §2.2 & §2.3: CEK (16B) & Nonce (12B)
	cek, err := deriveHKDF(ikm, salt, []byte("Content-Encoding: aes128gcm\x00"), 16)
	if err != nil {
		return nil, err
	}
	nonce, err := deriveHKDF(ikm, salt, []byte("Content-Encoding: nonce\x00"), 12)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, fmt.Errorf("membuat cipher AES: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("membuat GCM: %w", err)
	}

	// RFC 8188 §2: satu rekaman terakhir diakhiri byte pembatas 0x02.
	record := make([]byte, len(plaintext)+1)
	copy(record, plaintext)
	record[len(plaintext)] = 0x02

	ciphertext := gcm.Seal(nil, nonce, record, nil)

	// Header aes128gcm (86 byte): salt (16) || rs (4) || idlen (1) || keyid (65)
	headerLen := 16 + 4 + 1 + len(asPubBytes)
	out := make([]byte, 0, headerLen+len(ciphertext))
	out = append(out, salt...)
	var rsBuf [4]byte
	binary.BigEndian.PutUint32(rsBuf[:], recordSize)
	out = append(out, rsBuf[:]...)
	out = append(out, byte(len(asPubBytes)))
	out = append(out, asPubBytes...)
	out = append(out, ciphertext...)

	return out, nil
}

func deriveHKDF(secret, salt, info []byte, length int) ([]byte, error) {
	reader := hkdf.New(sha256.New, secret, salt, info)
	buf := make([]byte, length)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return nil, fmt.Errorf("menurunkan kunci HKDF: %w", err)
	}
	return buf, nil
}
