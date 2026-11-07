package matchtrader

// Bentuk payload REST Match-Trader Trader API (v1).
// Nama field mengikuti response asli broker apa adanya.

// accountSummary adalah ringkasan saldo akun.
type accountSummary struct {
	Balance   float64 `json:"balance"`
	Credit    float64 `json:"credit"`
	Equity    float64 `json:"equity"`
	FreeFunds float64 `json:"freeFunds"`
	Margin    float64 `json:"margin"`
	MarginLvl float64 `json:"marginLevel"`
	Profit    float64 `json:"profit"`
}

// symbolInfo adalah kontrak satu simbol. Multiplier menentukan nilai
// per lot untuk CFD; contractSize dipakai untuk forex.
type symbolInfo struct {
	Symbol          string  `json:"symbol"`
	Alias           string  `json:"alias"`
	Type            string  `json:"type"`
	BaseCurrency    string  `json:"baseCurrency"`
	QuoteCurrency   string  `json:"quoteCurrency"`
	ContractSize    float64 `json:"contractSize"`
	Multiplier      float64 `json:"multiplier"`
	PricePrecision  int     `json:"pricePrecision"`
	SizeOfOnePoint  float64 `json:"sizeOfOnePoint"`
	VolumeMin       float64 `json:"volumeMin"`
	VolumeMax       float64 `json:"volumeMax"`
	VolumeStep      float64 `json:"volumeStep"`
	VolumePrecision int     `json:"volumePrecision"`
	SessionOpen     bool    `json:"sessionOpen"`
}

type symbolsResponse struct {
	Symbols []symbolInfo `json:"symbols"`
}

// quote adalah harga bid/ask terakhir.
type quote struct {
	Symbol    string  `json:"symbol"`
	Alias     string  `json:"alias"`
	Bid       float64 `json:"bid"`
	Ask       float64 `json:"ask"`
	Timestamp int64   `json:"timestamp"`
}

type quotesResponse struct {
	Quotes []quote `json:"quotes"`
}

// position adalah posisi hidup.
type position struct {
	ID           string   `json:"id"`
	Symbol       string   `json:"symbol"`
	Alias        string   `json:"alias"`
	Side         string   `json:"side"`
	Volume       float64  `json:"volume"`
	OpenPrice    float64  `json:"openPrice"`
	CurrentPrice float64  `json:"currentPrice"`
	SLPrice      *float64 `json:"slPrice"`
	TPPrice      *float64 `json:"tpPrice"`
	Profit       float64  `json:"profit"`
	NetProfit    float64  `json:"netProfit"`
	Margin       float64  `json:"margin"`
	OpenTime     int64    `json:"openTime"`
	Comment      string   `json:"comment"`
}

type positionsResponse struct {
	Positions []position `json:"positions"`
}

// closedPosition adalah posisi yang sudah ditutup broker.
type closedPosition struct {
	ID          string  `json:"id"`
	Symbol      string  `json:"symbol"`
	Alias       string  `json:"alias"`
	Side        string  `json:"side"`
	Volume      float64 `json:"volume"`
	OpenPrice   float64 `json:"openPrice"`
	ClosePrice  float64 `json:"closePrice"`
	Profit      float64 `json:"profit"`
	NetProfit   float64 `json:"netProfit"`
	CloseReason string  `json:"closeReason"`
	OpenTime    int64   `json:"openTime"`
	CloseTime   int64   `json:"closeTime"`
	Comment     string  `json:"comment"`
}

type closedPositionsResponse struct {
	ClosedPositions []closedPosition `json:"closedPositions"`
}

// openPositionRequest adalah body order market beserta SL/TP server-side.
type openPositionRequest struct {
	Symbol  string   `json:"symbol"`
	Side    string   `json:"side"`
	Volume  float64  `json:"volume"`
	SLPrice *float64 `json:"slPrice,omitempty"`
	TPPrice *float64 `json:"tpPrice,omitempty"`
	Comment string   `json:"comment,omitempty"`
}

// apiError adalah amplop error standar broker.
type apiError struct {
	Type   string `json:"type"`
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}
