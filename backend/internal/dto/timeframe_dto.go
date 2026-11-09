package dto

// TimeframePair adalah satu pasangan timeframe yang tersedia untuk sebuah
// metode analisis.
//
// Pasangannya berasal dari aturan trading ILA (pakem engine), bukan ditentukan
// platform. Platform hanya mencatat mana yang dipantau dan mana yang boleh
// dieksekusi otomatis.
type TimeframePair struct {
	Htf string `json:"htf"`
	Ltf string `json:"ltf"`
	// Label siap tampil, mis. "H4 · M15".
	Label string `json:"label"`
}

// TimeframeSwitchItem adalah saklar satu akun untuk satu pasangan timeframe.
type TimeframeSwitchItem struct {
	Htf   string `json:"htf"`
	Ltf   string `json:"ltf"`
	Watch bool   `json:"watch"`
	Auto  bool   `json:"auto"`
}

// TimeframeSwitchAccount adalah seluruh saklar timeframe satu akun untuk satu
// metode pada satu pair.
type TimeframeSwitchAccount struct {
	AccountID string `json:"account_id"`
	Label     string `json:"label"`
	// Pairs berisi saklar per pasangan timeframe.
	Pairs []TimeframeSwitchItem `json:"pairs"`
}

// StrategyTimeframeGroup mengelompokkan saklar per metode analisis.
type StrategyTimeframeGroup struct {
	Strategy string                   `json:"strategy"`
	Akun     []TimeframeSwitchAccount `json:"accounts"`
}

// TimeframeSwitchMatrix adalah jawaban seluruh saklar timeframe untuk satu pair.
type TimeframeSwitchMatrix struct {
	Symbol string `json:"symbol"`
	// Tersedia memetakan metode ke daftar pasangan timeframe yang dikenal.
	// Kosong berarti engine belum pernah mengirim pasangan untuk metode itu.
	Tersedia map[string][]TimeframePair `json:"tersedia"`
	// AutoAllowed menandai apakah tier pengguna boleh mengeksekusi otomatis.
	// Bila false, UI menyembunyikan saklar auto sepenuhnya.
	AutoAllowed bool                     `json:"auto_allowed"`
	Groups      []StrategyTimeframeGroup `json:"groups"`
}

// UpdateTimeframeSwitchRequest mengubah saklar satu akun untuk satu metode.
type UpdateTimeframeSwitchRequest struct {
	Strategy string                `json:"strategy" binding:"required"`
	Pairs    []TimeframeSwitchItem `json:"pairs" binding:"required"`
}
