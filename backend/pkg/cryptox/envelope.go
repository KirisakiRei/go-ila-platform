// Package cryptox menyediakan enkripsi amplop untuk kredensial broker.
//
// Modelnya tiga lapis agar satu kebocoran tidak membuka semua akun:
//
//	master key (env) ─membungkus→ user key (DB) ─mengenkripsi→ credential blob (DB)
//
// Master key tidak pernah disimpan di database, sehingga salinan database saja
// tidak cukup untuk membaca kredensial.
package cryptox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// KeySize adalah panjang kunci AES-256 dalam byte.
const KeySize = 32

var (
	// ErrInvalidKey menandai kunci dengan panjang salah.
	ErrInvalidKey = errors.New("kunci harus 32 byte (AES-256)")
	// ErrDecrypt menandai ciphertext rusak atau kunci tidak cocok.
	ErrDecrypt = errors.New("gagal mendekripsi data")
	// ErrKeyringEmpty menandai keyring tanpa kunci aktif.
	ErrKeyringEmpty = errors.New("keyring tidak memiliki kunci aktif")
)

// Envelope adalah hasil enkripsi amplop: kunci pembungkus dan isi terenkripsi.
type Envelope struct {
	UserKeyWrapped []byte
	UserKeyNonce   []byte
	Ciphertext     []byte
	Nonce          []byte
	KeyVersion     int
}

// Keyring menyimpan kunci master per versi agar rotasi tidak memutus data lama.
type Keyring struct {
	keys   map[int]cipher.AEAD
	active int
}

// NewKeyring menyiapkan kunci aktif dari string base64 (32 byte).
// Kunci lama ikut didaftarkan agar blob lama tetap bisa dibaca.
func NewKeyring(activeVersion int, activeKey string, legacy map[int]string) (*Keyring, error) {
	key, err := decodeKey(activeKey)
	if err != nil {
		return nil, fmt.Errorf("kunci aktif: %w", err)
	}

	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}

	ring := &Keyring{keys: map[int]cipher.AEAD{activeVersion: aead}, active: activeVersion}

	for version, raw := range legacy {
		if version == activeVersion {
			continue
		}
		legacyKey, decodeErr := decodeKey(raw)
		if decodeErr != nil {
			return nil, fmt.Errorf("kunci versi %d: %w", version, decodeErr)
		}
		legacyAEAD, aeadErr := newAEAD(legacyKey)
		if aeadErr != nil {
			return nil, aeadErr
		}
		ring.keys[version] = legacyAEAD
	}

	return ring, nil
}

// ActiveVersion mengembalikan versi kunci yang dipakai untuk enkripsi baru.
func (r *Keyring) ActiveVersion() int {
	return r.active
}

// Seal mengenkripsi kredensial dengan kunci user baru.
func (r *Keyring) Seal(plaintext, associatedData []byte) (Envelope, error) {
	aead, ok := r.keys[r.active]
	if !ok {
		return Envelope{}, ErrKeyringEmpty
	}

	userKey := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, userKey); err != nil {
		return Envelope{}, fmt.Errorf("membuat kunci user: %w", err)
	}

	userAEAD, err := newAEAD(userKey)
	if err != nil {
		return Envelope{}, err
	}

	ciphertext, nonce, err := sealWith(userAEAD, plaintext, associatedData)
	if err != nil {
		return Envelope{}, err
	}

	wrapped, wrappedNonce, err := sealWith(aead, userKey, associatedData)
	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		UserKeyWrapped: wrapped,
		UserKeyNonce:   wrappedNonce,
		Ciphertext:     ciphertext,
		Nonce:          nonce,
		KeyVersion:     r.active,
	}, nil
}

// Open membuka envelope dengan versi kunci yang tercatat di dalamnya.
func (r *Keyring) Open(env Envelope, associatedData []byte) ([]byte, error) {
	aead, ok := r.keys[env.KeyVersion]
	if !ok {
		return nil, fmt.Errorf("versi kunci %d tidak dikenal", env.KeyVersion)
	}

	userKey, err := openWith(aead, env.UserKeyWrapped, env.UserKeyNonce, associatedData)
	if err != nil {
		return nil, err
	}
	if len(userKey) != KeySize {
		return nil, ErrDecrypt
	}

	userAEAD, err := newAEAD(userKey)
	if err != nil {
		return nil, err
	}

	return openWith(userAEAD, env.Ciphertext, env.Nonce, associatedData)
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("membuat cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("membuat GCM: %w", err)
	}
	return aead, nil
}

func sealWith(aead cipher.AEAD, plaintext, associatedData []byte) ([]byte, []byte, error) {
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("membuat nonce: %w", err)
	}
	return aead.Seal(nil, nonce, plaintext, associatedData), nonce, nil
}

func openWith(aead cipher.AEAD, ciphertext, nonce, associatedData []byte) ([]byte, error) {
	if len(nonce) != aead.NonceSize() {
		return nil, ErrDecrypt
	}
	plaintext, err := aead.Open(nil, nonce, ciphertext, associatedData)
	if err != nil {
		// Detail kegagalan tidak diteruskan: itu sinyal untuk penyerang.
		return nil, ErrDecrypt
	}
	return plaintext, nil
}

func decodeKey(raw string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("kunci bukan base64: %w", err)
	}
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}
	return key, nil
}
