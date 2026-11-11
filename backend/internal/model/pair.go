package model

// PairCategory adalah kelas aset yang dikelola admin: Forex, Indices,
// Commodity, Crypto. User tidak pernah mengetik nama kategori; mereka hanya
// memilih dari daftar yang disediakan (plans pair dictionary).
type PairCategory struct {
	Base
	// Key adalah kunci stabil yang dipakai frontend dan API, mis. "forex".
	Key   string `gorm:"type:varchar(30);uniqueIndex;not null" json:"key"`
	Label string `gorm:"type:varchar(60);not null" json:"label"`
	// SortOrder menentukan urutan tampil di frontend; kecil lebih dulu.
	SortOrder int `gorm:"not null;default:0" json:"sort_order"`
	// IsEnabled=false menyembunyikan seluruh kategori beserta isinya dari user,
	// tanpa menghapus data. Admin dapat mengaktifkannya kembali kapan saja.
	IsEnabled bool `gorm:"not null;default:true" json:"is_enabled"`
}

// TableName menetapkan nama tabel.
func (PairCategory) TableName() string {
	return "pair_category"
}

// PairGroup adalah sub-kategori di dalam kategori, mis. "USD Quote", "JPY
// Quote", atau "Mayor"/"Minor". Tujuannya semata memudahkan navigating di
// frontend; secara logika pair tetap milik kategorinya.
type PairGroup struct {
	Base
	CategoryID string `gorm:"type:varchar(36);index;not null" json:"category_id"`
	Key        string `gorm:"type:varchar(30);not null" json:"key"`
	Label      string `gorm:"type:varchar(60);not null" json:"label"`
	SortOrder  int    `gorm:"not null;default:0" json:"sort_order"`
	IsEnabled  bool   `gorm:"not null;default:true" json:"is_enabled"`
}

// TableName menetapkan nama tabel.
func (PairGroup) TableName() string {
	return "pair_group"
}

// TradingPair adalah simbol yang boleh dipilih user.
//
// Symbol adalah kunci kanonik yang dipakai engine. Aliases memetakan nama lain
// yang mungkin dikirim engine atau broker (mis. "GOLD" untuk "XAUUSD") ke
// simbol kanonik ini, sehingga sinyal lama tetap dikenali.
type TradingPair struct {
	Base
	Symbol      string `gorm:"type:varchar(20);uniqueIndex;not null" json:"symbol"`
	DisplayName string `gorm:"type:varchar(60)" json:"display_name"`
	CategoryID  string `gorm:"type:varchar(36);index;not null" json:"category_id"`
	// GroupID boleh kosong: pair tetap sah walau belum dimasukkan ke sub-kategori.
	GroupID *string `gorm:"type:varchar(36);index" json:"group_id"`
	// Aliases berisi alias dipisah koma, mis. "GOLD,XAU". Pencocokan
	// case-insensitive dan membuang spasi.
	Aliases string `gorm:"type:varchar(120)" json:"aliases"`
	// CorrelatedSymbols berisi satu atau lebih simbol pembanding SMT dipisah
	// koma, mis. "XAGUSD,XAUEUR". Nilainya hanya data: cara memakainya
	// (mis. SMT divergence) sepenuhnya milik engine.
	CorrelatedSymbols string `gorm:"type:varchar(120)" json:"correlated_symbols"`
	// FeedSource adalah sumber candle yang dipakai engine untuk simbol ini,
	// mis. "FOREXCOM" atau "ICMARKETS". Hanya admin yang menetapkannya; user
	// tidak pernah memilih broker data. Nilai kosong berarti pemetaan bawaan
	// engine (FOREXCOM untuk forex/logam/indeks, BINANCE untuk kripto).
	FeedSource  string `gorm:"type:varchar(30);not null;default:''" json:"feed_source"`
	SortOrder   int    `gorm:"not null;default:0" json:"sort_order"`
	IsEnabled   bool   `gorm:"not null;default:true" json:"is_enabled"`
	Description string `gorm:"type:varchar(255)" json:"description"`

	// Metadata kontrak dipakai kalkulator risiko untuk mengubah lot menjadi
	// nominal uang. Nilai nol berarti belum diisi: kalkulator memakai nilai
	// bawaan kategori, lalu ditimpa spesifikasi broker saat eksekusi.
	//
	// ValuePerLot adalah nilai uang untuk pergerakan harga 1.0 pada 1 lot.
	ValuePerLot  float64 `gorm:"not null;default:0" json:"value_per_lot"`
	ContractSize float64 `gorm:"not null;default:0" json:"contract_size"`
	MinLot       float64 `gorm:"not null;default:0.01" json:"min_lot"`
	MaxLot       float64 `gorm:"not null;default:10" json:"max_lot"`
	LotStep      float64 `gorm:"not null;default:0.01" json:"lot_step"`
}

// TableName menetapkan nama tabel.
func (TradingPair) TableName() string {
	return "trading_pair"
}
