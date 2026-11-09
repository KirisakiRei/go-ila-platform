package dto

// RiskSimRequest adalah permintaan simulasi risiko untuk satu skenario.
//
// Equity opsional: bila kosong, server memakai ekuitas terakhir yang dilaporkan
// Runner untuk akun tersebut. Bila keduanya kosong, permintaan ditolak.
type RiskSimRequest struct {
	AccountID   string  `json:"account_id"`
	Symbol      string  `json:"symbol" binding:"required,min=3,max=20"`
	EntryPrice  float64 `json:"entry_price" binding:"required"`
	StopLoss    float64 `json:"stop_loss" binding:"required"`
	Equity      float64 `json:"equity"`
	Mode        string  `json:"mode" binding:"omitempty,oneof=percent lot"`
	RiskPercent float64 `json:"risk_percent"`
	LotSize     float64 `json:"lot_size"`
}
