package model

// AccountPairTimeframe adalah pengaturan pantau dan auto-eksekusi untuk satu
// pasangan timeframe pada satu akun.
//
// Latar belakang: satu pair dipantau oleh beberapa pasangan timeframe yang
// sudah pakem dari aturan trading ILA 2.0 (mis. W-H4, H4-M15). Platform tidak
// pernah menentukan pasangannya — ia hanya mencatat pilihan user mana yang
// dipantau dan mana yang boleh dieksekusi otomatis.
//
// Semua kolom saklar bersifat sengaja:
//   - IsWatch=false berarti sinyal pasangan ini tidak relevan bagi akun itu dan
//     tidak diproses sama sekali.
//   - IsAuto=false berarti sinyalnya tetap ditampilkan sebagai peluang, tetapi
//     tidak dieksekusi otomatis. User dapat memutuskan lewat tombol eksekusi.
//
// IsAuto hanya sah bila IsWatch ikut menyala: tanpa dipantau, sinyalnya tidak
// pernah datang sehingga auto tidak akan pernah berjalan. Batasan ini ditegakkan
// di service, bukan hanya di UI.
type AccountPairTimeframe struct {
	Base
	BrokerAccountID string `gorm:"type:varchar(36);index:idx_account_pair_timeframe,unique,priority:1;not null" json:"broker_account_id"`
	UserID          string `gorm:"type:varchar(36);index;not null" json:"user_id"`
	// Symbol adalah kunci kanonik yang dipakai engine.
	Symbol string `gorm:"type:varchar(20);index:idx_account_pair_timeframe,unique,priority:2;not null" json:"symbol"`
	// Strategy berisi "ila-2.0" atau "ila-3.0". Pasangan timeframe ILA 3.0
	// berbeda dari ILA 2.0, jadi metode ikut menjadi bagian kunci.
	Strategy string `gorm:"type:varchar(20);index:idx_account_pair_timeframe,unique,priority:3;not null" json:"strategy"`
	// Htf dan Ltf mengikuti penulisan baku dari NormalizeTimeframe.
	Htf string `gorm:"type:varchar(10);index:idx_account_pair_timeframe,unique,priority:4;not null" json:"htf"`
	Ltf string `gorm:"type:varchar(10);index:idx_account_pair_timeframe,unique,priority:5;not null" json:"ltf"`
	// IsWatch menandai pasangan ini dipantau (sinyalnya diminta ke engine dan
	// ditampilkan ke user).
	IsWatch bool `gorm:"not null;default:false" json:"is_watch"`
	// IsAuto menandai pasangan ini boleh dieksekusi otomatis.
	IsAuto bool `gorm:"not null;default:false" json:"is_auto"`
}

// TableName menetapkan nama tabel.
func (AccountPairTimeframe) TableName() string {
	return "account_pair_timeframe"
}
