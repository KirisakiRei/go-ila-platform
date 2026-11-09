package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// approvalTokenSize adalah panjang token konfirmasi dalam byte.
// 32 byte acak, dikodekan base64url — cukup besar untuk tidak bisa ditebak.
const approvalTokenSize = 32

// newApprovalTokenPair membuat token mentah sekaligus hash SHA-256-nya.
//
// Yang disimpan di basis data hanya hash; token mentah tidak pernah tersimpan
// sehingga bocornya isi tabel tidak membuat tautan konfirmasi bisa dipalsukan.
func newApprovalTokenPair() (raw string, hash string, err error) {
	buf := make([]byte, approvalTokenSize)
	if _, readErr := rand.Read(buf); readErr != nil {
		return "", "", fmt.Errorf("membuat token acak: %w", readErr)
	}

	raw = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:]), nil
}
