// Package broker mendefinisikan kontrak eksekusi ke platform broker.
// Runner hanya bergantung pada interface ini, bukan pada SDK tertentu,
// sehingga cTrader dan Match-Trader bisa dipasang tanpa mengubah inti.
package broker

import (
	"context"
	"errors"
	"fmt"
)

// Error yang harus dibedakan pemanggil.
var (
	// ErrNotConnected: sesi ke broker belum siap.
	ErrNotConnected = errors.New("broker tidak terhubung")
	// ErrInsufficientMargin: broker menolak karena margin tidak cukup.
	ErrInsufficientMargin = errors.New("margin tidak mencukupi")
	// ErrInvalidCredentials: kredensial ditolak broker.
	ErrInvalidCredentials = errors.New("kredensial broker tidak sah")
	// ErrUnsupported: simbol atau operasi tidak didukung adapter.
	ErrUnsupported = errors.New("operasi tidak didukung broker ini")
	// ErrPositionNotFound: posisi sudah tidak ada di broker.
	ErrPositionNotFound = errors.New("posisi tidak ditemukan di broker")
)

// SymbolSpec adalah kontrak simbol yang dibutuhkan perhitungan lot.
type SymbolSpec struct {
	Symbol       string
	Digits       int
	PipSize      float64
	ValuePerLot  float64 // nilai uang per 1.0 pergerakan harga untuk 1 lot
	MinLot       float64
	MaxLot       float64
	LotStep      float64
	ContractSize float64
}

// AccountSnapshot adalah kondisi akun saat diminta.
type AccountSnapshot struct {
	AccountID  string
	Currency   string
	Balance    float64
	Equity     float64
	FreeMargin float64
	Leverage   int
}

// Quote adalah harga dua arah dari broker beserta waktu kutipannya.
//
// Bid dan Ask dipisahkan karena spread akun adalah informasi yang dibutuhkan
// gerbang eksekusi: akun dengan spread lebar wajar bergerak lebih jauh dari
// level entry sebelum order terkirim. Tanpa bid/ask, toleransi hanya bisa
// menebak dari karakter instrumen.
type Quote struct {
	// Bid adalah harga jual terbaik yang tersedia.
	Bid float64
	// Ask adalah harga beli terbaik yang tersedia.
	Ask float64
	// At adalah waktu kutipan dalam unix detik menurut broker.
	// Nol berarti broker tidak melaporkan waktu.
	At int64
}

// Mid mengembalikan harga tengah bid/ask.
func (q Quote) Mid() float64 { return (q.Bid + q.Ask) / 2 }

// Spread mengembalikan selisih bid/ask, dibulatkan agar tidak membawa derau
// float64 ke perhitungan batas. Adapter yang tidak punya bid/ask mengembalikan
// Quote dengan Bid==Ask, sehingga spread-nya nol.
func (q Quote) Spread() float64 {
	d := q.Ask - q.Bid
	if d < 0 {
		return 0
	}
	return d
}

// QuoteSource adalah kemampuan opsional adapter membaca harga dua arah.
//
// Sengaja dipisahkan dari Client agar adapter yang belum mendukung bid/ask
// tidak wajib berubah. Pemanggil memeriksa dukungan lewat type assertion dan
// jatuh ke CurrentPrice bila tidak tersedia.
type QuoteSource interface {
	Quote(ctx context.Context, symbol string) (Quote, error)
}

// OpenRequest adalah permintaan membuka posisi dengan SL/TP sejak awal.
// SL dan TP wajib ikut dikirim agar proteksi terpasang di sisi broker.
type OpenRequest struct {
	Symbol     string
	Side       string // "BUY" | "SELL"
	Lots       float64
	StopLoss   float64
	TakeProfit float64
	Comment    string
	ClientID   string // idempotency key, diteruskan ke broker bila didukung
}

// OpenResult adalah bukti order diterima broker.
type OpenResult struct {
	Ticket        string
	ExecutedPrice float64
	Lots          float64
	StopLoss      float64
	TakeProfit    float64
}

// PositionState adalah posisi yang masih hidup di broker.
type PositionState struct {
	Ticket     string
	Symbol     string
	Side       string
	Lots       float64
	OpenPrice  float64
	StopLoss   float64
	TakeProfit float64
	Profit     float64
	// Comment adalah komentar order di broker. Runner menaruh id sinyal di sini
	// untuk mengenali order yang sudah terkirim sebelum koneksi terputus.
	Comment string
}

// CloseReason menjelaskan bagaimana posisi berakhir.
type CloseReason string

const (
	CloseReasonTP     CloseReason = "TP"
	CloseReasonSL     CloseReason = "SL"
	CloseReasonManual CloseReason = "MANUAL"
	CloseReasonOther  CloseReason = "OTHER"
)

// ClosedPosition adalah posisi yang sudah tidak ada lagi di broker.
type ClosedPosition struct {
	Ticket    string
	ExitPrice float64
	Profit    float64
	Reason    CloseReason
	ClosedAt  int64 // unix detik menurut broker
}

// Credentials adalah isi blob kredensial setelah didekripsi.
// Nilainya hanya hidup di memori Runner dan tidak pernah dicatat ke log.
type Credentials struct {
	Platform  string            `json:"platform"`
	AccountID string            `json:"account_id"`
	Login     string            `json:"login,omitempty"`
	Password  string            `json:"password,omitempty"`
	Server    string            `json:"server,omitempty"`
	Token     string            `json:"token,omitempty"`
	Extra     map[string]string `json:"extra,omitempty"`
}

// String menutupi nilai rahasia bila tidak sengaja tercetak.
func (c Credentials) String() string {
	return fmt.Sprintf("Credentials{platform:%s account:%s}", c.Platform, c.AccountID)
}

// GoString berlaku untuk format %#v agar rahasia tetap tersembunyi.
func (c Credentials) GoString() string {
	return c.String()
}

// Client adalah satu akun broker milik user.
// Implementasi wajib aman dipakai dari beberapa goroutine.
type Client interface {
	Platform() string
	AccountID() string

	// Connect membuka sesi dengan kredensial yang sudah didekripsi.
	Connect(ctx context.Context, creds Credentials) error
	// Connected menandai sesi masih hidup.
	Connected() bool
	// Close memutus sesi dan membuang kredensial dari memori.
	Close() error

	Account(ctx context.Context) (AccountSnapshot, error)
	Symbol(ctx context.Context, symbol string) (SymbolSpec, error)

	// CurrentPrice mengembalikan harga acuan eksekusi (bid/ask tengah).
	// Dipakai slippage guard; bukan feed untuk menutup posisi.
	CurrentPrice(ctx context.Context, symbol string) (float64, error)

	// OpenPosition memasang order beserta SL/TP di server broker.
	OpenPosition(ctx context.Context, req OpenRequest) (OpenResult, error)

	ListPositions(ctx context.Context) ([]PositionState, error)
	ClosedSince(ctx context.Context, unixSeconds int64) ([]ClosedPosition, error)
}

// Factory membuat klien untuk satu akun broker.
type Factory func(accountID, brokerType string) (Client, error)

// Registry menyimpan adapter per jenis broker.
type Registry struct {
	factories map[string]Factory
}

// NewRegistry membuat registry kosong.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]Factory)}
}

// Register mendaftarkan satu adapter.
func (r *Registry) Register(brokerType string, factory Factory) {
	r.factories[brokerType] = factory
}

// New membuat klien sesuai jenis broker akun.
func (r *Registry) New(brokerType, accountID string) (Client, error) {
	factory, ok := r.factories[brokerType]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupported, brokerType)
	}
	return factory(accountID, brokerType)
}

// Supported menandai jenis broker yang punya adapter terpasang.
func (r *Registry) Supported(brokerType string) bool {
	_, ok := r.factories[brokerType]
	return ok
}
