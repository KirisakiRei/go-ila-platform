package ila

// Versi aturan ILA 2.0 yang tersimpan di pustaka ini.
//
// Mengapa versi ini penting: hasil backtest hanya bisa dibandingkan bila aturan
// yang dipakai sama. Tanpa penanda versi, perubahan kecil pada aturan entry
// akan membuat angka lama dan angka baru tercampur, dan tidak ada yang bisa
// membedakan mana yang memakai aturan mana. Setiap run backtest menyimpan
// versi ini bersama commit kode, sehingga hasil dari masa lalu tetap bisa
// ditelusuri.
//
// Aturan menaikkan versi:
//   - PATCH (1.0.x): perbaikan yang tidak mengubah keputusan trading.
//   - MINOR (1.x.0): penambahan aturan yang bisa mengubah jumlah entry.
//   - MAJOR (x.0.0): perubahan cara membaca struktur pasar.
//
// Versi ini HARUS naik bersamaan dengan perubahan aturan apa pun, sekecil
// apa pun, sebab angka backtest bergantung padanya.
const (
	// StrategyName adalah nama metode yang aturannya hidup di pustaka ini.
	StrategyName = "ila-2.0"

	// Version adalah versi aturan, dalam bentuk yang aman ditampilkan ke UI
	// maupun disimpan sebagai kunci pembanding hasil.
	Version = "ila-2.0-v2.0.0"
	// VersionNumber adalah bagian angka saja, untuk perbandingan terurut
	// ketika UI perlu mengurutkan hasil menurut versi.
	VersionNumber = "2.0.0"
)
