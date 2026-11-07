package utils

import "golang.org/x/crypto/bcrypt"

// HashPassword menghasilkan hash bcrypt dari password plaintext.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ComparePassword membandingkan hash tersimpan dengan password yang dikirim.
// Hasilnya selalu boolean agar pemanggil tidak salah mengira galat sebagai "cocok".
func ComparePassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
