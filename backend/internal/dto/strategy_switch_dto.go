package dto

// UpdateStrategySwitchRequest mengubah saklar metode untuk satu akun dan satu
// pair. Kuncinya nama metode ("ila-2.0" / "ila-3.0"), nilainya saklar aktif.
//
// Hanya metode yang dikirim yang berubah; metode lain pada pair tersebut tidak
// disentuh.
type UpdateStrategySwitchRequest struct {
	Strategies map[string]bool `json:"strategies" binding:"required"`
}

// StrategySwitchAccount adalah satu baris matriks akun × metode untuk sebuah pair.
type StrategySwitchAccount struct {
	AccountID  string `json:"account_id"`
	Label      string `json:"label"`
	BrokerType string `json:"broker_type"`
	IsActive   bool   `json:"is_active"`
	IsDryRun   bool   `json:"is_dry_run"`
	// Strategies memetakan nama metode ke status saklarnya.
	Strategies map[string]bool `json:"strategies"`
	// Allowed memetakan nama metode ke apakah tier pengguna berhak memakainya.
	// Metode yang tidak berhak tidak boleh ditampilkan UI.
	Allowed map[string]bool `json:"allowed"`
}

// StrategySwitchMatrix adalah jawaban daftar akun beserta saklar metodenya
// untuk satu pair. Pair pantauan bersifat global, sehingga matriks ini yang
// menentukan metode mana yang dieksekusi tiap akun.
type StrategySwitchMatrix struct {
	Symbol string                  `json:"symbol"`
	Akun   []StrategySwitchAccount `json:"accounts"`
}
