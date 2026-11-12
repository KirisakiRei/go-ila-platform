package runner

import "strings"

// DefaultValuePerLot mengembalikan nilai uang untuk pergerakan harga 1.0 pada
// 1 lot, dipakai menghitung hasil posisi dalam satuan R.
//
// Mengapa perlu: jarak harga bukan satuan uang. Selisih 0,00164 pada EURUSD
// adalah 164 USD per lot, sedangkan selisih 0,00164 pada XAUUSD hanya sekitar
// 0,16 USD per lot. Membagi profit (USD) dengan lot x jarak harga menghasilkan
// angka tanpa satuan yang bisa meledak; satu posisi EURUSD pernah tercatat
// 192.073,17R padahal hasilnya +1,92R.
//
// Nilai di bawah adalah nilai kontrak standar; bila pair punya metadata di
// katalog admin, nilai itulah yang seharusnya dipakai (lihat runner di
// router.go). Peta ini menjadi cadangan agar R tetap terhitung wajar.
func DefaultValuePerLot(symbol string) float64 {
	simbol := strings.ToUpper(strings.TrimSpace(symbol))

	// Alias yang dikirim engine/broker.
	switch simbol {
	case "GOLD":
		simbol = "XAUUSD"
	case "SILVER":
		simbol = "XAGUSD"
	case "NAS100", "US100", "NDX":
		simbol = "USTEC"
	case "SPX500", "SP500":
		simbol = "US500"
	case "DOW30", "DJ30":
		simbol = "US30"
	case "DAX40", "DE40":
		simbol = "GER40"
	case "WTI", "CRUDEOIL":
		simbol = "USOIL"
	}

	switch simbol {
	case "XAUUSD", "XAUEUR":
		// Emas: 1 lot = 100 ons, pergerakan 1.0 = 100 USD.
		return 100
	case "XAGUSD", "XAGEUR":
		// Perak: 1 lot = 5.000 ons.
		return 5_000
	case "USTEC", "NAS100", "US100", "US30", "US500", "GER40", "UK100", "JP225", "HK50":
		// Indeks: pergerakan 1,0 poin senilai 1 USD per lot.
		return 1
	case "USOIL", "UKOIL":
		return 1_000
	case "NATGAS":
		return 10_000
	}

	// Kripto dikutip dalam USD per koin: 1 lot = 1 koin.
	if strings.HasPrefix(simbol, "BTC") ||
		strings.HasPrefix(simbol, "ETH") ||
		strings.HasPrefix(simbol, "SOL") ||
		strings.HasPrefix(simbol, "XRP") ||
		strings.HasPrefix(simbol, "LTC") {
		return 1
	}

	// Pair yen: 1 lot = 100.000 unit mata uang dasar, dikutip per yen.
	if strings.HasSuffix(simbol, "JPY") {
		return 1_000
	}

	// Forex lainnya: 1 lot = 100.000 unit.
	return 100_000
}

// valuePerLotRunner memilih sumber nilai per lot: fungsi yang dipasang
// pemanggil lebih dulu, lalu nilai bawaan.
func (s *Service) valuePerLotRunner(symbol string) float64 {
	if s.opts.ValuePerLot != nil {
		if nilai := s.opts.ValuePerLot(symbol); nilai > 0 {
			return nilai
		}
	}
	return DefaultValuePerLot(symbol)
}
