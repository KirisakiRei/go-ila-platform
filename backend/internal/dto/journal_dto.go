package dto

import "time"

// JournalStatsResponse adalah ringkasan angka jurnal eksekusi.
//
// Dihitung dari posisi yang sudah ditutup untuk winrate dan total R; posisi
// yang masih berjalan dipisahkan agar angka tidak berubah setiap halaman
// dimuat dan tetap bisa dipakai menilai strategi.
type JournalStatsResponse struct {
	TotalPositions  int     `json:"total_positions"`
	OpenPositions   int     `json:"open_positions"`
	ClosedTP        int     `json:"closed_tp"`
	ClosedSL        int     `json:"closed_sl"`
	ClosedManual    int     `json:"closed_manual"`
	Untracked       int     `json:"untracked"`
	Wins            int     `json:"wins"`
	Losses          int     `json:"losses"`
	WinRate         float64 `json:"win_rate"`
	TotalR          float64 `json:"total_r"`
	AverageR        float64 `json:"average_r"`
	TotalProfit     float64 `json:"total_profit"`
	DryRunPositions int     `json:"dry_run_positions"`
}

// JournalSkippedResponse adalah satu sinyal yang tidak berakhir sebagai posisi.
//
// Rincian sinyal ikut dikirim agar pengguna bisa menilai mengapa peluang itu
// terlewat (harga sudah lewat, masa berlaku habis, atau dilewati sendiri),
// bukan hanya melihat hitungannya.
type JournalSkippedResponse struct {
	SignalID        string  `json:"signal_id"`
	Symbol          string  `json:"symbol"`
	StrategyVersion string  `json:"strategy_version"`
	Direction       string  `json:"direction"`
	Htf             string  `json:"htf"`
	Ltf             string  `json:"ltf"`
	EntryPrice      float64 `json:"entry_price"`
	StopLoss        float64 `json:"stop_loss"`
	TakeProfit      float64 `json:"take_profit"`
	RiskReward      float64 `json:"risk_reward"`
	// Status adalah status user, bukan status sinyal: satu sinyal bisa
	// ditolak untuk satu akun dan dieksekusi di akun lain.
	Status       string    `json:"status"`
	Reason       string    `json:"reason,omitempty"`
	OccurredAt   time.Time `json:"occurred_at"`
	OccurredPair TimePair  `json:"occurred_at_pair"`
}

// JournalResponse adalah isi satu permintaan jurnal.
type JournalResponse struct {
	Stats     JournalStatsResponse     `json:"stats"`
	Positions []PositionResponse       `json:"positions"`
	Skipped   []JournalSkippedResponse `json:"skipped"`
	// SkippedMeta menjelaskan halaman daftar peluang terlewat. Tanpa ini
	// antarmuka hanya tahu baris yang dikirim, tidak tahu masih ada lagi.
	SkippedMeta JournalMetaResponse `json:"skipped_meta"`
	// Filters mengembalikan pilihan yang benar-benar dipakai server, sehingga
	// antarmuka bisa menampilkan rentang yang berlaku.
	Filters JournalFiltersResponse `json:"filters"`
}

// JournalMetaResponse adalah ringkasan halaman daftar peluang terlewat.
type JournalMetaResponse struct {
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasMore    bool `json:"has_more"`
}

// JournalFiltersResponse adalah filter yang dipakai server.
type JournalFiltersResponse struct {
	From            string `json:"from,omitempty"`
	To              string `json:"to,omitempty"`
	StrategyVersion string `json:"strategy_version,omitempty"`
	Symbol          string `json:"symbol,omitempty"`
}
