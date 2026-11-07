package redact

import (
	"errors"
	"strings"
	"testing"
)

func TestSecretsMenggantiTokenBotTelegram(t *testing.T) {
	input := `Post "https://api.telegram.org/bot123456789:AAFakeTokenValueYangPanjangSekali/sendMessage": refused`

	got := Secrets(input)
	if strings.Contains(got, "AAFakeTokenValueYangPanjangSekali") {
		t.Fatalf("token bot masih ada: %s", got)
	}
	if !strings.Contains(got, penanda) {
		t.Fatalf("seharusnya ada penanda: %s", got)
	}
}

func TestSecretsMenggantiJWT(t *testing.T) {
	token := "eyJhbGciOiJIUzI1NiJ9.eyJ0b2tlblV1aWQiOiJhYmMifQ.signatureBagianKetiga"
	got := Secrets("gagal login dengan token " + token)

	if strings.Contains(got, token) {
		t.Fatalf("jwt masih ada: %s", got)
	}
}

func TestSecretsMenggantiHeaderBearer(t *testing.T) {
	got := Secrets("Authorization: Bearer abcdefghijklmnop")

	if strings.Contains(got, "abcdefghijklmnop") {
		t.Fatalf("nilai bearer masih ada: %s", got)
	}
}

func TestSecretsMenggantiParameterRahasia(t *testing.T) {
	cases := []string{
		`{"token":"rahasia123"}`,
		`?access_token=rahasia123`,
		`password=rahasia123`,
		`"api_key": "rahasia123"`,
	}

	for _, input := range cases {
		got := Secrets(input)
		if strings.Contains(got, "rahasia123") {
			t.Fatalf("nilai rahasia masih ada pada %q menjadi %q", input, got)
		}
	}
}

func TestSecretsTidakMerusakTeksBiasa(t *testing.T) {
	input := "sinyal XAUUSD ditolak karena batas harian tercapai"
	if got := Secrets(input); got != input {
		t.Fatalf("teks biasa berubah: %q", got)
	}
}

func TestSecretsAmanUntukStringKosong(t *testing.T) {
	if got := Secrets(""); got != "" {
		t.Fatalf("ingin kosong, dapat %q", got)
	}
}

func TestErrorAmanUntukNil(t *testing.T) {
	if got := Error(nil); got != "" {
		t.Fatalf("ingin kosong, dapat %q", got)
	}
}

func TestErrorMenyaringIsi(t *testing.T) {
	err := errors.New(`dial https://api.telegram.org/bot123456789:AAFakeTokenValueYangPanjangSekali/x`)

	got := Error(err)
	if strings.Contains(got, "AAFakeTokenValueYangPanjangSekali") {
		t.Fatalf("token masih ada: %s", got)
	}
}
