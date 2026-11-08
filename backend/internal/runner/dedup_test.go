package runner

import (
	"testing"

	"ila/backend/internal/model"
)

// Dua entri akun yang menunjuk akun broker sama hanya boleh menghasilkan
// satu order. Tanpa penyaringan ini, satu sinyal dieksekusi dua kali.
func TestAkunKembarHanyaSekali(t *testing.T) {
	accounts := []model.BrokerAccount{
		{Base: model.Base{ID: "a1"}, BrokerType: "matchtrader", ExternalAccountID: "1089", IsActive: true},
		{Base: model.Base{ID: "a2"}, BrokerType: "matchtrader", ExternalAccountID: "1089", IsActive: true},
		{Base: model.Base{ID: "a3"}, BrokerType: "matchtrader", ExternalAccountID: "2099", IsActive: true},
	}

	seen := map[string]struct{}{}
	var kept []string
	for _, account := range accounts {
		key := accountIdentity(account)
		if key != "" {
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
		}
		kept = append(kept, account.ID)
	}

	if len(kept) != 2 || kept[0] != "a1" || kept[1] != "a3" {
		t.Fatalf("akun yang dipakai: %v; ingin [a1 a3]", kept)
	}
}

// Akun tanpa penanda broker tidak digabung: setiap entri dianggap berbeda.
func TestAkunTanpaIdentitasTidakDigabung(t *testing.T) {
	if accountIdentity(model.BrokerAccount{Base: model.Base{ID: "x"}}) != "" {
		t.Fatal("akun tanpa identitas broker harus mengembalikan string kosong")
	}
}
