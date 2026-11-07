package runner

import (
	"context"
	"fmt"
	"time"

	"ila/backend/internal/broker"
	"ila/backend/internal/model"
	"ila/backend/pkg/cryptox"
)

// IdempotencyKey membentuk kunci unik eksekusi.
// Satu sinyal hanya boleh melahirkan satu order per akun broker.
func IdempotencyKey(signalID, accountID string) string {
	return fmt.Sprintf("exec:%s:%s", signalID, accountID)
}

// CredentialDecoder membuka blob kredensial menjadi isi yang bisa dipakai
// untuk login ke broker. Plaintext hanya hidup di memori proses ini.
func (s *Service) credentialDecoder(ctx context.Context, accountID string) (*broker.Credentials, error) {
	if s.keyring == nil {
		return nil, fmt.Errorf("keyring kredensial belum disiapkan")
	}

	blob, err := s.store.FindCredential(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// Associated data mengikat ciphertext ke akun pemiliknya, sehingga blob
	// tidak bisa dipindahkan antar akun.
	plaintext, err := s.keyring.Open(cryptox.Envelope{
		UserKeyWrapped: blob.UserKeyWrapped,
		UserKeyNonce:   blob.UserKeyNonce,
		Ciphertext:     blob.Ciphertext,
		Nonce:          blob.Nonce,
		KeyVersion:     blob.KeyVersion,
	}, []byte(blob.BrokerAccountID))
	if err != nil {
		return nil, err
	}

	return decodeCredentials(plaintext)
}

// client mengembalikan klien broker yang sudah tersambung untuk satu akun.
//
// Klien di-cache per akun agar sesi broker tetap hidup di antara sinyal;
// membuka login baru untuk setiap sinyal akan lambat dan mudah kena limit.
func (s *Service) client(ctx context.Context, account *model.BrokerAccount) (broker.Client, error) {
	s.mu.Lock()
	if existing, ok := s.clients[account.ID]; ok && existing.Connected() {
		s.mu.Unlock()
		return existing, nil
	}
	s.mu.Unlock()

	creds, err := s.credentialDecoder(ctx, account.ID)
	if err != nil {
		return nil, err
	}

	client, err := s.brokers.New(account.BrokerType, account.ID)
	if err != nil {
		return nil, err
	}

	connectCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	if err := client.Connect(connectCtx, *creds); err != nil {
		return nil, err
	}

	s.mu.Lock()
	if previous, ok := s.clients[account.ID]; ok {
		_ = previous.Close()
	}
	s.clients[account.ID] = client
	s.mu.Unlock()

	return client, nil
}

// CloseClients memutus semua sesi broker. Dipanggil saat Runner berhenti
// agar kredensial tidak tertinggal di memori.
func (s *Service) CloseClients() {
	s.mu.Lock()
	clients := s.clients
	s.clients = make(map[string]broker.Client)
	s.mu.Unlock()

	for id, client := range clients {
		if err := client.Close(); err != nil {
			s.logger.Warn("gagal menutup sesi broker", "account_id", id, "error", err.Error())
		}
	}
}

// WithKeyring memasang pembuka kredensial.
func (s *Service) WithKeyring(keyring *cryptox.Keyring) *Service {
	s.keyring = keyring
	return s
}

// DropClient memutus sesi satu akun dan membuang tokennya dari memori.
// Dipanggil saat user mencabut kredensial (plans/12 §3.4): sesi lama tidak
// boleh tetap hidup setelah rahasianya dihapus dari database.
func (s *Service) DropClient(accountID string) {
	s.mu.Lock()
	client, ok := s.clients[accountID]
	delete(s.clients, accountID)
	s.mu.Unlock()

	if !ok {
		return
	}
	if err := client.Close(); err != nil {
		s.logger.Warn("gagal menutup sesi akun", "account_id", accountID, "error", err.Error())
	}
}
