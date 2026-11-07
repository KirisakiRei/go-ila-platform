package service

import (
	"encoding/json"
	"fmt"

	"ila/backend/internal/broker"
)

// encodeCredentials menyiapkan blob yang akan dienkripsi.
//
// Bentuk JSON-nya sama dengan yang dibaca Runner (broker/Credentials), tetapi
// didefinisikan ulang di sini agar backend-api tidak bergantung pada Runner.
func encodeCredentials(creds broker.Credentials) ([]byte, error) {
	raw, err := json.Marshal(creds)
	if err != nil {
		return nil, fmt.Errorf("menyandikan kredensial: %w", err)
	}
	return raw, nil
}
