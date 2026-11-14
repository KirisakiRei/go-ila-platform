package dto

// BacktestCoverageResponse adalah cakupan data historis satu simbol.
//
// Bentuknya sengaja datar dan berisi angka waktu Unix dalam detik: itu yang
// dipakai pemilih tanggal di UI, dan mengirimnya sebagai teks akan memaksa UI
// menebak zona waktu.
type BacktestCoverageResponse struct {
	Symbol string `json:"symbol"`
	Feed   string `json:"feed"`
	// Mulai dan Selesai adalah irisan seluruh pasangan timeframe, dalam Unix
	// detik UTC. Rentang yang boleh dipilih tidak pernah melebihi ini.
	Mulai    int64                          `json:"mulai"`
	Selesai  int64                          `json:"selesai"`
	Rentang  []BacktestTimeframeCoverageDTO `json:"rentang"`
	Pasangan []BacktestPairOptionDTO        `json:"pasangan"`
	// DiperiksaPada dalam Unix detik UTC.
	DiperiksaPada int64 `json:"diperiksa_pada"`
}

// BacktestTimeframeCoverageDTO adalah cakupan satu timeframe.
type BacktestTimeframeCoverageDTO struct {
	Timeframe string `json:"timeframe"`
	Mulai     int64  `json:"mulai"`
	Selesai   int64  `json:"selesai"`
	Jumlah    int    `json:"jumlah"`
}

// BacktestPairOptionDTO adalah satu pasangan timeframe beserta cakupannya.
type BacktestPairOptionDTO struct {
	HTF      string `json:"htf"`
	LTF      string `json:"ltf"`
	Label    string `json:"label"`
	Mulai    int64  `json:"mulai"`
	Selesai  int64  `json:"selesai"`
	Tersedia bool   `json:"tersedia"`
}

// BacktestQuotaResponse adalah sisa jatah backtest hari ini.
type BacktestQuotaResponse struct {
	Tier          string `json:"tier"`
	BatasHarian   int    `json:"batas_harian"`
	Terpakai      int    `json:"terpakai"`
	Sisa          int    `json:"sisa"`
	MaksHari      int    `json:"maks_hari"`
	RentangMaksRb int    `json:"rentang_maks_bulan"`
	BisaJalankan  bool   `json:"bisa_jalankan"`
	Alasan        string `json:"alasan,omitempty"`
}

// BacktestCreateRequest adalah permintaan menjalankan satu backtest.
type BacktestCreateRequest struct {
	Symbol string `json:"symbol" binding:"required"`
	HTF    string `json:"htf" binding:"required"`
	LTF    string `json:"ltf" binding:"required"`
	// Dari dan Sampai adalah Unix detik UTC. Keduanya inklusif.
	Dari   int64   `json:"dari" binding:"required"`
	Sampai int64   `json:"sampai" binding:"required"`
	Equity float64 `json:"equity"`
	Risk   float64 `json:"risk"`
}

// BacktestRunResponse adalah ringkasan satu run.
type BacktestRunResponse struct {
	ID     string  `json:"id"`
	Symbol string  `json:"symbol"`
	HTF    string  `json:"htf"`
	LTF    string  `json:"ltf"`
	Status string  `json:"status"`
	Error  string  `json:"error,omitempty"`
	Dari   int64   `json:"dari"`
	Sampai int64   `json:"sampai"`
	Equity float64 `json:"equity"`
	Risk   float64 `json:"risk"`

	// Pembanding jam: kapan run mulai dan selesai dikerjakan.
	MulaiPada   int64 `json:"mulai_pada,omitempty"`
	SelesaiPada int64 `json:"selesai_pada,omitempty"`
	DibuatPada  int64 `json:"dibuat_pada"`

	Statistik BacktestStatsDTO `json:"statistik"`

	// Strategi dan versinya ditampilkan agar hasil lama tidak tertukar dengan
	// hasil setelah aturan berubah.
	Strategi   string `json:"strategi"`
	Versi      string `json:"versi"`
	KodeCommit string `json:"kode_commit,omitempty"`
}

// BacktestStatsDTO adalah angka ringkas satu run.
type BacktestStatsDTO struct {
	TotalTrade    int     `json:"total_trade"`
	Menang        int     `json:"menang"`
	Kalah         int     `json:"kalah"`
	Kedaluwarsa   int     `json:"kedaluwarsa"`
	CandleAmbigu  int     `json:"candle_ambigu"`
	SinyalDitolak int     `json:"sinyal_ditolak"`
	WinRate       float64 `json:"win_rate"`
	TotalR        float64 `json:"total_r"`
	MaksDrawdown  float64 `json:"maks_drawdown"`
	ProfitFactor  float64 `json:"profit_factor"`
	EkuitasAkhir  float64 `json:"ekuitas_akhir"`
	EkuitasMulai  float64 `json:"ekuitas_mulai"`
}

// BacktestEquityPointDTO adalah satu titik kurva ekuitas.
type BacktestEquityPointDTO struct {
	Waktu    int64   `json:"waktu"`
	Ekuitas  float64 `json:"ekuitas"`
	Drawdown float64 `json:"drawdown"`
}

// BacktestRunDetailResponse adalah satu run beserta kurva ekuitasnya.
type BacktestRunDetailResponse struct {
	BacktestRunResponse
	KurvaEkuitas []BacktestEquityPointDTO `json:"kurva_ekuitas"`
}

// BacktestTradeDTO adalah satu entry di dalam run.
//
// Potongan candle sengaja tidak ikut di sini: daftar bisa berisi puluhan entry
// dan mengirim semua candle sekaligus membuat balasan membengkak. Chart dimuat
// saat pengguna memilih satu entry.
type BacktestTradeDTO struct {
	Seq  int    `json:"seq"`
	Arah string `json:"arah"`
	// Waktu masuk dan keluar dalam Unix detik UTC.
	WaktuMasuk     int64   `json:"waktu_masuk"`
	WaktuKeluar    int64   `json:"waktu_keluar"`
	Entry          float64 `json:"entry"`
	StopLoss       float64 `json:"stop_loss"`
	TakeProfit     float64 `json:"take_profit"`
	RRRencana      float64 `json:"rr_rencana"`
	Hasil          string  `json:"hasil"`
	R              float64 `json:"r"`
	DurasiDetik    int     `json:"durasi_detik"`
	EkuitasAkhir   float64 `json:"ekuitas_akhir"`
	Ambigu         bool    `json:"ambigu"`
	Ringkasan      string  `json:"ringkasan"`
	LangkahBerikut string  `json:"langkah_berikut"`
	// ChartTersedia bernilai salah setelah candle dibersihkan penyapu.
	ChartTersedia bool `json:"chart_tersedia"`
}

// BacktestCandleDTO adalah satu lilin untuk chart detail.
type BacktestCandleDTO struct {
	Time   int64   `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

// BacktestZoneDTO adalah satu zona FVG pada chart.
type BacktestZoneDTO struct {
	Top    float64 `json:"top"`
	Bottom float64 `json:"bottom"`
	// Dari dan Sampai dalam Unix detik UTC.
	Dari   int64  `json:"dari"`
	Sampai int64  `json:"sampai"`
	Label  string `json:"label"`
}

// BacktestMarkerDTO adalah satu penanda harga pada chart (sweep atau CISD).
type BacktestMarkerDTO struct {
	Waktu int64   `json:"waktu"`
	Harga float64 `json:"harga"`
	Label string  `json:"label"`
}

// BacktestTradeDetailResponse adalah satu entry beserta candle dan levelnya.
type BacktestTradeDetailResponse struct {
	BacktestTradeDTO
	HTF       string              `json:"htf"`
	LTF       string              `json:"ltf"`
	CandleHTF []BacktestCandleDTO `json:"candle_htf"`
	CandleLTF []BacktestCandleDTO `json:"candle_ltf"`
	ZonaFVG   *BacktestZoneDTO    `json:"zona_fvg,omitempty"`
	Sweep     *BacktestMarkerDTO  `json:"sweep,omitempty"`
	CISD      *BacktestMarkerDTO  `json:"cisd,omitempty"`
	// DibersihkanPada diisi ketika candle sudah dihapus penyapu.
	DibersihkanPada int64 `json:"dibersihkan_pada,omitempty"`
}
