package utils

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Parameter Argon2id sesuai plans/12 §4.1.
const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MB dalam KiB
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

// ErrHashFormat menandai hash tersimpan yang tidak bisa dibaca.
var ErrHashFormat = errors.New("format hash tidak dikenal")

// HashPassword menghasilkan hash Argon2id berformat standar PHC.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("membuat salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// ComparePassword membandingkan hash tersimpan dengan password yang dikirim.
// Hash bcrypt lama tetap diterima agar akun yang sudah ada tidak terkunci.
func ComparePassword(hash, password string) bool {
	if strings.HasPrefix(hash, "$argon2id$") {
		return compareArgon2id(hash, password)
	}
	if strings.HasPrefix(hash, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	}
	return false
}

// NeedsRehash menandai hash lama yang sebaiknya diperbarui saat user login.
func NeedsRehash(hash string) bool {
	return !strings.HasPrefix(hash, "$argon2id$")
}

func compareArgon2id(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return false
	}

	var memory uint32
	var timeCost uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeCost, &threads); err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	got := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}
