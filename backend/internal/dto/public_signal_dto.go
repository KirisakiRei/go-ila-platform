package dto

import "time"

// PublicSignalPreview adalah ringkasan sinyal untuk halaman publik.
//
// Bentuknya SENGAJA terpisah dari SignalResponse. Endpoint publik hanya boleh
// memuat arah dan waktu; harga masuk, batas kerugian, target, dan ringkasan
// analisis adalah isi berlangganan. Memisahkan tipe membuat field harga tidak
// ikut terbawa hanya karena SignalResponse suatu saat ditambah.
type PublicSignalPreview struct {
	GeneratedAt time.Time                 `json:"generated_at"`
	Market      PublicMarketState         `json:"market"`
	Items       []PublicSignalPreviewItem `json:"items"`
}

// PublicMarketState adalah keadaan pasar saat ringkasan disusun.
type PublicMarketState struct {
	Open  bool   `json:"open"`
	Label string `json:"label"`
}

// PublicSignalPreviewItem adalah satu instrumen beserta arah terakhirnya.
type PublicSignalPreviewItem struct {
	Symbol      string `json:"symbol"`
	DisplayName string `json:"display_name"`
	// Category memakai kunci katalog: forex, indices, commodity, crypto.
	Category  string                `json:"category"`
	HasSignal bool                  `json:"has_signal"`
	Signal    *PublicSignalSnapshot `json:"signal,omitempty"`
}

// PublicSignalSnapshot adalah arah sinyal terakhir tanpa parameter apa pun.
type PublicSignalSnapshot struct {
	// Direction bernilai BULLISH atau BEARISH.
	Direction string `json:"direction"`
	// Htf dan Ltf adalah rentang waktu sumber sinyal. Kosong berarti engine
	// belum mengirimkannya; tampilan wajib memperlakukannya sebagai belum
	// diketahui, bukan menebak.
	Htf string `json:"htf"`
	Ltf string `json:"ltf"`
	// Status hanya ACTIVE atau EXPIRED. Sinyal yang dibatalkan atau ditolak
	// tidak pernah sampai ke sini.
	Status      string    `json:"status"`
	PublishedAt time.Time `json:"published_at"`
}
