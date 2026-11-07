package runner

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/broker"
)

// ErrMalformedCredential menandai blob kredensial yang tidak bisa dibaca.
var ErrMalformedCredential = errors.New("blob kredensial tidak sah")

// decodeCredentials membaca JSON kredensial hasil dekripsi.
// Dipisah agar bisa diuji tanpa database.
func decodeCredentials(plaintext []byte) (*broker.Credentials, error) {
	var creds broker.Credentials
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedCredential, err)
	}
	if creds.Platform == "" {
		return nil, fmt.Errorf("%w: platform kosong", ErrMalformedCredential)
	}
	return &creds, nil
}

// EncodeCredentials menyiapkan blob yang akan dienkripsi.
// Dipakai backend-api saat user menyimpan kredensial.
func EncodeCredentials(creds broker.Credentials) ([]byte, error) {
	raw, err := json.Marshal(creds)
	if err != nil {
		return nil, fmt.Errorf("menyandikan kredensial: %w", err)
	}
	return raw, nil
}
