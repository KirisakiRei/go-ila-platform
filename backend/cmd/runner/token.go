package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// approvalTokenSize adalah panjang token konfirmasi dalam byte.
// 32 byte acak, dikodekan base64url (plans/13 §4.2).
const approvalTokenSize = 32

// newApprovalToken membuat token mentah sekaligus hash SHA-256-nya.
// Yang disimpan di database hanya hash; token mentah tidak pernah tersimpan.
func newApprovalToken() (raw string, hash string, err error) {
	buf := make([]byte, approvalTokenSize)
	if _, readErr := rand.Read(buf); readErr != nil {
		return "", "", fmt.Errorf("membuat token acak: %w", readErr)
	}

	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, hashToken(raw), nil
}

// hashToken menghitung hash yang disimpan untuk verifikasi token.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
