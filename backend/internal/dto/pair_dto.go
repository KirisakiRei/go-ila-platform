package dto

// --- Katalog pair (dibaca user) ---------------------------------------------

// PairItem adalah satu simbol yang dapat dipilih user.
type PairItem struct {
	Symbol      string `json:"symbol"`
	DisplayName string `json:"name"`
	Description string `json:"description,omitempty"`
	Aliases     string `json:"aliases,omitempty"`
	// CorrelatedSymbols adalah instrumen pembanding SMT (bisa lebih dari satu).
	// Nilainya hanya informasi; evaluasi korelasi tetap milik engine.
	CorrelatedSymbols []string `json:"correlated_symbols,omitempty"`
	// Metadata kontrak untuk kalkulator risiko. Nol berarti belum diisi dan
	// klien wajib memakai nilai bawaan kategori.
	ValuePerLot  float64 `json:"value_per_lot,omitempty"`
	ContractSize float64 `json:"contract_size,omitempty"`
	MinLot       float64 `json:"min_lot,omitempty"`
	MaxLot       float64 `json:"max_lot,omitempty"`
	LotStep      float64 `json:"lot_step,omitempty"`
	// FeedSource adalah sumber candle pilihan admin, mis. "FOREXCOM".
	// Dipakai dashboard untuk menampilkan label broker pada chart.
	FeedSource string `json:"feed_source,omitempty"`
}

// PairGroupView adalah sub-kategori beserta pair di dalamnya.
type PairGroupView struct {
	Key   string     `json:"key"`
	Label string     `json:"label"`
	Pairs []PairItem `json:"pairs"`
}

// PairCategoryView adalah kategori beserta grup dan pair-nya.
// GrupNested kosong berarti kategorinya tidak memakai sub-kategori.
type PairCategoryView struct {
	Key    string          `json:"key"`
	Label  string          `json:"label"`
	Groups []PairGroupView `json:"groups"`
}

// PairCatalogResponse adalah katalog aktif untuk checklist di frontend.
type PairCatalogResponse struct {
	Categories []PairCategoryView `json:"categories"`
	// Symbols adalah daftar datar simbol aktif, memudahkan validasi di klien.
	Symbols []string `json:"symbols"`
}

// --- Admin: kelola katalog --------------------------------------------------

// CreatePairCategoryRequest membuat kategori baru.
type CreatePairCategoryRequest struct {
	Key       string `json:"key" binding:"required,min=2,max=30"`
	Label     string `json:"label" binding:"required,min=2,max=60"`
	SortOrder int    `json:"sort_order"`
	IsEnabled *bool  `json:"is_enabled"`
}

// UpdatePairCategoryRequest mengubah kategori.
type UpdatePairCategoryRequest struct {
	Label     *string `json:"label" binding:"omitempty,min=2,max=60"`
	SortOrder *int    `json:"sort_order"`
	IsEnabled *bool   `json:"is_enabled"`
}

// CreatePairGroupRequest membuat sub-kategori.
type CreatePairGroupRequest struct {
	CategoryID string `json:"category_id" binding:"required"`
	Key        string `json:"key" binding:"required,min=2,max=30"`
	Label      string `json:"label" binding:"required,min=2,max=60"`
	SortOrder  int    `json:"sort_order"`
	IsEnabled  *bool  `json:"is_enabled"`
}

// UpdatePairGroupRequest mengubah sub-kategori.
type UpdatePairGroupRequest struct {
	Label     *string `json:"label" binding:"omitempty,min=2,max=60"`
	SortOrder *int    `json:"sort_order"`
	IsEnabled *bool   `json:"is_enabled"`
}

// CreatePairRequest menambahkan simbol ke katalog.
type CreatePairRequest struct {
	Symbol      string `json:"symbol" binding:"required,min=3,max=20"`
	DisplayName string `json:"display_name" binding:"max=60"`
	CategoryID  string `json:"category_id" binding:"required"`
	GroupID     string `json:"group_id"`
	Aliases     string `json:"aliases" binding:"max=120"`
	// CorrelatedSymbols diisi simbol pembanding SMT dipisah koma.
	CorrelatedSymbols string `json:"correlated_symbols" binding:"max=120"`
	SortOrder         int    `json:"sort_order"`
	IsEnabled         *bool  `json:"is_enabled"`
	Description       string `json:"description" binding:"max=255"`
	// Metadata kontrak; nilai nol berarti mengikuti bawaan kategori.
	ValuePerLot  float64 `json:"value_per_lot"`
	ContractSize float64 `json:"contract_size"`
	MinLot       float64 `json:"min_lot"`
	MaxLot       float64 `json:"max_lot"`
	LotStep      float64 `json:"lot_step"`
	// FeedSource adalah sumber candle untuk simbol ini, mis. "ICMARKETS".
	// Kosong berarti mengikuti pemetaan bawaan engine.
	FeedSource string `json:"feed_source" binding:"max=30"`
}

// UpdatePairRequest mengubah simbol.
type UpdatePairRequest struct {
	DisplayName       *string  `json:"display_name" binding:"omitempty,max=60"`
	CategoryID        *string  `json:"category_id"`
	GroupID           *string  `json:"group_id"`
	Aliases           *string  `json:"aliases" binding:"omitempty,max=120"`
	CorrelatedSymbols *string  `json:"correlated_symbols" binding:"omitempty,max=120"`
	SortOrder         *int     `json:"sort_order"`
	IsEnabled         *bool    `json:"is_enabled"`
	Description       *string  `json:"description" binding:"omitempty,max=255"`
	ValuePerLot       *float64 `json:"value_per_lot"`
	ContractSize      *float64 `json:"contract_size"`
	MinLot            *float64 `json:"min_lot"`
	MaxLot            *float64 `json:"max_lot"`
	LotStep           *float64 `json:"lot_step"`
	// FeedSource kosong mengembalikan simbol ke pemetaan bawaan engine.
	FeedSource *string `json:"feed_source" binding:"omitempty,max=30"`
}

// AdminPairView menampilkan pair lengkap untuk panel admin.
type AdminPairView struct {
	ID          string `json:"id"`
	Symbol      string `json:"symbol"`
	DisplayName string `json:"display_name"`
	CategoryID  string `json:"category_id"`
	CategoryKey string `json:"category_key"`
	GroupID     string `json:"group_id,omitempty"`
	GroupKey    string `json:"group_key,omitempty"`
	Aliases     string `json:"aliases,omitempty"`
	// CorrelatedSymbols dikirim sebagai CSV agar mudah disunting sebagai teks.
	CorrelatedSymbols string `json:"correlated_symbols,omitempty"`
	SortOrder         int    `json:"sort_order"`
	IsEnabled         bool   `json:"is_enabled"`
	Description       string `json:"description,omitempty"`
	// Subscribers adalah jumlah langganan aktif yang memakai simbol ini.
	// Admin perlu melihatnya sebelum menonaktifkan atau menghapus pair.
	Subscribers int64 `json:"subscribers"`
	// Metadata kontrak untuk kalkulator risiko dan pratinjau lot.
	ValuePerLot  float64 `json:"value_per_lot"`
	ContractSize float64 `json:"contract_size"`
	MinLot       float64 `json:"min_lot"`
	MaxLot       float64 `json:"max_lot"`
	LotStep      float64 `json:"lot_step"`
	// FeedSource adalah sumber candle pilihan admin, mis. "ICMARKETS".
	FeedSource string `json:"feed_source"`
}

// AdminPairCatalogResponse adalah seluruh katalog untuk admin.
type AdminPairCatalogResponse struct {
	Categories []AdminPairCategoryView `json:"categories"`
}

// AdminPairCategoryView adalah kategori beserta grup dan pair untuk admin.
type AdminPairCategoryView struct {
	ID        string               `json:"id"`
	Key       string               `json:"key"`
	Label     string               `json:"label"`
	SortOrder int                  `json:"sort_order"`
	IsEnabled bool                 `json:"is_enabled"`
	Groups    []AdminPairGroupView `json:"groups"`
	Pairs     []AdminPairView      `json:"pairs"`
}

// AdminPairGroupView adalah sub-kategori untuk admin.
type AdminPairGroupView struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Label     string `json:"label"`
	SortOrder int    `json:"sort_order"`
	IsEnabled bool   `json:"is_enabled"`
}

// --- Checklist pair per akun ------------------------------------------------

// AccountPairSelectionResponse adalah kondisi checklist satu akun.
type AccountPairSelectionResponse struct {
	AccountID   string              `json:"account_id"`
	RiskPercent float64             `json:"risk_percent"`
	Selected    []string            `json:"selected"`
	Catalog     PairCatalogResponse `json:"catalog"`
}

// UpdateAccountPairsRequest mengganti seluruh pilihan pair satu akun.
type UpdateAccountPairsRequest struct {
	Symbols []string `json:"symbols"`
}
