package redact_test

import (
	"fmt"
	"regexp"
	"testing"

	"ila/backend/pkg/redact"
)

// TestKasusNyataDariKodeProduksi memakai bentuk pesan error yang benar-benar
// muncul dari transport Telegram dan Match-Trader.
func TestKasusNyataDariKodeProduksi(t *testing.T) {
	cases := []struct {
		nama      string
		pesan     string
		terlarang string
	}{
		{
			nama: "telegram dns gagal",
			pesan: `Post "https://api.telegram.org/bot7654321098:AAHq7Xk2LmNpQrStUvWxYz0123456789abc/sendMessage": ` +
				`dial tcp: lookup api.telegram.org: no such host`,
			terlarang: "AAHq7Xk2LmNpQrStUvWxYz0123456789abc",
		},
		{
			nama: "header bearer tercetak",
			pesan: "request gagal, header Authorization: Bearer " +
				"eyJhbGciOiJIUzI1NiJ9.eyJ1IjoxMTQ3fQ.abc123signature",
			terlarang: "eyJhbGciOiJIUzI1NiJ9",
		},
		{
			nama:      "match-trader menolak",
			pesan:     `{"token":"eyJhbGciOiJIUzI1NiJ9.brokerToken.x"}`,
			terlarang: "brokerToken",
		},
	}

	for _, tc := range cases {
		t.Run(tc.nama, func(t *testing.T) {
			hasil := redact.Secrets(tc.pesan)
			if regexp.MustCompile(regexp.QuoteMeta(tc.terlarang)).MatchString(hasil) {
				t.Fatalf("rahasia masih ada:\n  masukan : %s\n  keluaran: %s", tc.pesan, hasil)
			}
			fmt.Printf("  %-22s -> %s\n", tc.nama, hasil)
		})
	}
}
