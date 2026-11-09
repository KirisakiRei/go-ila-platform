package model

// AccountPairStrategy adalah saklar metode analisis per akun dan per pair.
//
// Satu pair dipantau oleh kedua metode, tetapi tiap akun menentukan metode mana
// yang benar-benar dieksekusi. Pemisahan ini menggantikan kolom
// `subscription.strategy` yang lama, yang mengikat metode ke daftar pantau
// global sehingga user tidak bisa memakai metode berbeda di akun berbeda.
type AccountPairStrategy struct {
	Base
	BrokerAccountID string `gorm:"type:varchar(36);index:idx_account_pair_strategy,unique,priority:1;not null" json:"broker_account_id"`
	UserID          string `gorm:"type:varchar(36);index;not null" json:"user_id"`
	// Symbol dipakai, bukan PairID, agar saklar tetap sah walau katalog
	// diubah admin. Simbolnya adalah kunci kanonik yang dipakai engine.
	Symbol string `gorm:"type:varchar(20);index:idx_account_pair_strategy,unique,priority:2;not null" json:"symbol"`
	// Strategy berisi "ila-2.0" atau "ila-3.0".
	Strategy string `gorm:"type:varchar(20);index:idx_account_pair_strategy,unique,priority:3;not null" json:"strategy"`
	// IsEnabled=false mematikan eksekusi metode ini untuk akun ini saja.
	IsEnabled bool `gorm:"not null" json:"is_enabled"`
}

// TableName menetapkan nama tabel.
func (AccountPairStrategy) TableName() string {
	return "account_pair_strategy"
}
