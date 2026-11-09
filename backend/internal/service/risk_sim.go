package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	"ila/backend/internal/dto"
	"ila/backend/pkg/apperror"
)

// Simulasi risiko: satu tempat yang mengubah ukuran posisi menjadi nominal
// uang (dan sebaliknya) untuk ditampilkan sebelum order dijalankan.
//
// Rumus dasarnya sama dengan kalkulator lot di runner:
//
//	stop_distance  = |entry - stop_loss|
//	risiko_nominal = stop_distance x nilai_per_lot x lots
//	persen_efektif = risiko_nominal / equity x 100
//
// Nilai per lot diambil berlapis (A lalu C):
//  1. metadata kontrak pada katalog pair (diisi admin),
//  2. bawaan kategori bila metadata kosong,
//  3. spesifikasi broker bila akun nyata dan tokennya masih sah.
const (
	// lotTabelSimulasi adalah ukuran lot yang ditampilkan sebagai padanan.
	// Dipilih supaya pengguna melihat rentang kecil sampai besar dalam satu layar.
	lotTabelSimulasi = 7
)

// LotSpecResolved adalah metadata kontrak yang akhirnya dipakai perhitungan.
type LotSpecResolved struct {
	ValuePerLot  float64
	ContractSize float64
	MinLot       float64
	MaxLot       float64
	LotStep      float64
	// Sumber menjelaskan asal angka: "broker", "katalog", atau "kategori".
	Sumber string
}

// HasilSimulasi adalah satu baris perhitungan risiko.
type HasilSimulasi struct {
	Lots         float64 `json:"lots"`
	RiskAmount   float64 `json:"risk_amount"`
	RiskPercent  float64 `json:"risk_percent"`
	StopDistance float64 `json:"stop_distance"`
	Warning      bool    `json:"warning"`
}

// BarisTabelSimulasi adalah satu baris pratinjau pada tabel persen.
type BarisTabelSimulasi struct {
	RiskPercent float64 `json:"risk_percent"`
	Lots        float64 `json:"lots"`
	RiskAmount  float64 `json:"risk_amount"`
	Warning     bool    `json:"warning"`
}

// BarisTabelLot adalah satu baris pratinjau pada tabel lot.
type BarisTabelLot struct {
	Lots        float64 `json:"lots"`
	RiskAmount  float64 `json:"risk_amount"`
	RiskPercent float64 `json:"risk_percent"`
	Warning     bool    `json:"warning"`
}

// RiskSimResponse adalah jawaban lengkap simulasi risiko.
type RiskSimResponse struct {
	SumberNilai  string  `json:"sumber_nilai"`
	ValuePerLot  float64 `json:"value_per_lot"`
	ContractSize float64 `json:"contract_size"`
	MinLot       float64 `json:"min_lot"`
	MaxLot       float64 `json:"max_lot"`
	LotStep      float64 `json:"lot_step"`
	Equity       float64 `json:"equity"`
	StopDistance float64 `json:"stop_distance"`
	Mode         string  `json:"mode"`

	Hasil      HasilSimulasi        `json:"hasil"`
	Tabel      []BarisTabelSimulasi `json:"tabel"`
	TabelLot   []BarisTabelLot      `json:"tabel_lot"`
	Peringatan []string             `json:"peringatan"`
}

// RiskSimulator menghitung simulasi risiko untuk satu akun & pair.
type RiskSimulator interface {
	SimulateRisk(ctx context.Context, userID string, req dto.RiskSimRequest) (*RiskSimResponse, error)
}

// BrokerSpecProvider menyediakan spesifikasi simbol langsung dari broker.
//
// Sesi broker hidup di proses Runner, bukan backend-api, sehingga penyedia ini
// diisi oleh jembatan ke Runner. Bila nil atau gagal, simulasi memakai nilai
// katalog dan menandai sumbernya secara jujur.
type BrokerSpecProvider interface {
	SymbolSpec(ctx context.Context, userID, accountID, symbol string) (SpecBroker, bool)
}

// SpecBroker adalah irisan spesifikasi broker yang relevan untuk perhitungan.
type SpecBroker struct {
	ValuePerLot  float64
	ContractSize float64
	MinLot       float64
	MaxLot       float64
	LotStep      float64
}

type riskSimulator struct {
	accounts AccountService
	catalog  PairCatalogService
	broker   BrokerSpecProvider
}

// NewRiskSimulator merangkai kalkulator risiko.
func NewRiskSimulator(accounts AccountService, catalog PairCatalogService, broker BrokerSpecProvider) RiskSimulator {
	return &riskSimulator{accounts: accounts, catalog: catalog, broker: broker}
}

// batasPeringatan adalah ambang mulai munculnya peringatan di UI.
// Bukan penolakan: nilai tetap dihitung dan dikembalikan.
const batasPeringatan = 1.00

// SimulateRisk menghitung simulasi satu skenario beserta tabel pendamping.
func (s *riskSimulator) SimulateRisk(ctx context.Context, userID string, req dto.RiskSimRequest) (*RiskSimResponse, error) {
	symbol := strings.ToUpper(strings.TrimSpace(req.Symbol))
	if symbol == "" {
		return nil, fmt.Errorf("%w: simbol wajib diisi", apperror.ErrBadRequest)
	}
	if req.EntryPrice <= 0 || req.StopLoss <= 0 {
		return nil, fmt.Errorf("%w: harga entry dan stop loss harus lebih dari nol", apperror.ErrBadRequest)
	}
	if math.Abs(req.EntryPrice-req.StopLoss) < 1e-9 {
		return nil, fmt.Errorf("%w: stop loss tidak boleh sama dengan harga entry", apperror.ErrBadRequest)
	}

	spec, err := s.resolveSpec(ctx, userID, req, symbol)
	if err != nil {
		return nil, err
	}

	equity := req.Equity
	if equity <= 0 {
		equity = s.equityDariAkun(ctx, userID, req.AccountID)
	}
	if equity <= 0 {
		return nil, fmt.Errorf("%w: ekuitas wajib diisi untuk simulasi", apperror.ErrBadRequest)
	}

	stopDistance := math.Abs(req.EntryPrice - req.StopLoss)

	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = "percent"
	}

	resp := &RiskSimResponse{
		SumberNilai:  spec.Sumber,
		ValuePerLot:  spec.ValuePerLot,
		ContractSize: spec.ContractSize,
		MinLot:       spec.MinLot,
		MaxLot:       spec.MaxLot,
		LotStep:      spec.LotStep,
		Equity:       equity,
		StopDistance: stopDistance,
		Mode:         mode,
	}

	// Hasil sesuai pilihan pengguna.
	switch mode {
	case "lot":
		lot := req.LotSize
		if lot <= 0 {
			lot = spec.MinLot
		}
		resp.Hasil = hasilDariLot(lot, stopDistance, spec.ValuePerLot, equity, spec)
	default:
		resp.Hasil = hasilDariPersen(req.RiskPercent, stopDistance, spec.ValuePerLot, equity, spec)
	}

	// Tabel 1%–5%: supaya pengguna melihat konsekuensi tiap tingkat.
	for persen := 1.0; persen <= 5.0001; persen += 1.0 {
		row := hasilDariPersen(persen, stopDistance, spec.ValuePerLot, equity, spec)
		resp.Tabel = append(resp.Tabel, BarisTabelSimulasi{
			RiskPercent: persen,
			Lots:        row.Lots,
			RiskAmount:  row.RiskAmount,
			Warning:     persen > batasPeringatan,
		})
	}

	// Padanan lot: berapa risiko bila memakai ukuran lot tertentu.
	for _, lot := range lotsSimulasi(spec) {
		row := hasilDariLot(lot, stopDistance, spec.ValuePerLot, equity, spec)
		resp.TabelLot = append(resp.TabelLot, BarisTabelLot{
			Lots:        lot,
			RiskAmount:  row.RiskAmount,
			RiskPercent: row.RiskPercent,
			Warning:     row.RiskPercent > batasPeringatan,
		})
	}

	resp.Peringatan = peringatanSimulasi(mode, resp.Hasil, spec)
	return resp, nil
}

// resolveSpec menentukan nilai per lot yang dipakai: katalog dulu, lalu
// ditimpa spesifikasi broker bila tersedia.
func (s *riskSimulator) resolveSpec(ctx context.Context, userID string, req dto.RiskSimRequest, symbol string) (LotSpecResolved, error) {
	spec := LotSpecResolved{
		MinLot:  MinLotSize,
		MaxLot:  MaxLotSize,
		LotStep: 0.01,
		Sumber:  "kategori",
	}

	// 1) Katalog pair (A).
	if s.catalog != nil {
		if catalog, err := s.catalog.Catalog(ctx); err == nil {
			for _, category := range catalog.Categories {
				for _, group := range category.Groups {
					for _, item := range group.Pairs {
						if !strings.EqualFold(item.Symbol, symbol) {
							continue
						}
						if item.ValuePerLot > 0 {
							spec.ValuePerLot = item.ValuePerLot
							spec.Sumber = "katalog"
						}
						if item.ContractSize > 0 {
							spec.ContractSize = item.ContractSize
						}
						if item.MinLot > 0 {
							spec.MinLot = item.MinLot
						}
						if item.MaxLot > 0 {
							spec.MaxLot = item.MaxLot
						}
						if item.LotStep > 0 {
							spec.LotStep = item.LotStep
						}
					}
				}
			}
		}
	}

	// 2) Bawaan kategori bila katalog belum mengisi.
	if spec.ValuePerLot <= 0 {
		fallback := defaultValuePerLot(symbol)
		spec.ValuePerLot = fallback
		spec.Sumber = "kategori"
	}

	// 3) Spesifikasi broker bila tersedia. Kegagalan broker tidak menggagalkan
	// simulasi: nilai katalog tetap dipakai.
	if req.AccountID != "" && s.broker != nil {
		if brokerSpec, ok := s.broker.SymbolSpec(ctx, userID, req.AccountID, symbol); ok {
			if brokerSpec.ValuePerLot > 0 {
				spec.ValuePerLot = brokerSpec.ValuePerLot
				spec.Sumber = "broker"
			}
			if brokerSpec.ContractSize > 0 {
				spec.ContractSize = brokerSpec.ContractSize
			}
			if brokerSpec.MinLot > 0 {
				spec.MinLot = brokerSpec.MinLot
			}
			if brokerSpec.MaxLot > 0 {
				spec.MaxLot = brokerSpec.MaxLot
			}
			if brokerSpec.LotStep > 0 {
				spec.LotStep = brokerSpec.LotStep
			}
		}
	}

	return spec, nil
}

// equityDariAkun memakai ekuitas terakhir yang dilaporkan Runner.
func (s *riskSimulator) equityDariAkun(ctx context.Context, userID, accountID string) float64 {
	if accountID == "" || s.accounts == nil {
		return 0
	}
	accounts, err := s.accounts.List(ctx, userID)
	if err != nil {
		return 0
	}
	for _, account := range accounts {
		if account.ID == accountID && account.Equity > 0 {
			return account.Equity
		}
	}
	return 0
}

// hasilDariPersen menghitung lot dari persentase risiko.
func hasilDariPersen(percent, stopDistance, valuePerLot, equity float64, spec LotSpecResolved) HasilSimulasi {
	if percent <= 0 {
		percent = MinRiskPercent
	}
	riskAmount := equity * percent / 100
	lots := 0.0
	if stopDistance*valuePerLot > 0 {
		lots = riskAmount / (stopDistance * valuePerLot)
	}
	return normalkanHasil(lots, percent, stopDistance, valuePerLot, equity, spec)
}

// hasilDariLot menghitung risiko nominal dari ukuran lot tetap.
func hasilDariLot(lot, stopDistance, valuePerLot, equity float64, spec LotSpecResolved) HasilSimulasi {
	riskAmount := stopDistance * valuePerLot * lot
	percent := 0.0
	if equity > 0 {
		percent = riskAmount / equity * 100
	}
	return normalkanHasil(lot, percent, stopDistance, valuePerLot, equity, spec)
}

// normalkanHasil membulatkan lot ke kelipatan lot step lalu menghitung ulang
// nominal risiko dari lot yang benar-benar dipakai.
func normalkanHasil(lots, percent, stopDistance, valuePerLot, equity float64, spec LotSpecResolved) HasilSimulasi {
	step := spec.LotStep
	if step <= 0 {
		step = 0.01
	}
	// Mode persen: lot dibatasi pagar platform. Mode lot: lot sudah pilihan user.
	clamped := lots
	if clamped < spec.MinLot {
		clamped = spec.MinLot
	}
	if clamped > spec.MaxLot {
		clamped = spec.MaxLot
	}
	clamped = math.Round(clamped/step) * step
	clamped = math.Round(clamped*100) / 100

	riskAmount := stopDistance * valuePerLot * clamped
	effective := percent
	if equity > 0 {
		effective = riskAmount / equity * 100
	}

	return HasilSimulasi{
		Lots:         clamped,
		RiskAmount:   math.Round(riskAmount*100) / 100,
		RiskPercent:  math.Round(effective*100) / 100,
		StopDistance: math.Round(stopDistance*1e6) / 1e6,
		Warning:      effective > batasPeringatan,
	}
}

// lotsSimulasi menyusun ukuran lot contoh untuk tabel padanan.
func lotsSimulasi(spec LotSpecResolved) []float64 {
	acuan := []float64{0.01, 0.05, 0.10, 0.50, 1.00, 5.00, 10.00}
	out := make([]float64, 0, lotTabelSimulasi)
	for _, lot := range acuan {
		if lot < spec.MinLot {
			continue
		}
		if lot > spec.MaxLot {
			continue
		}
		out = append(out, lot)
	}
	if len(out) == 0 {
		out = append(out, spec.MinLot)
	}
	return out
}

// peringatanSimulasi menyusun catatan untuk ditampilkan di UI.
func peringatanSimulasi(mode string, hasil HasilSimulasi, spec LotSpecResolved) []string {
	out := make([]string, 0, 3)

	if spec.Sumber == "kategori" {
		out = append(out, "Nilai per lot memakai bawaan kategori karena metadata pair belum diisi admin. Angka bersifat perkiraan.")
	}
	if hasil.RiskPercent > batasPeringatan {
		out = append(out, fmt.Sprintf("Risiko %.2f%% per transaksi tergolong agresif. Sepuluh kerugian beruntun akan menggerus sekitar %.0f%% ekuitas.", hasil.RiskPercent, math.Min(hasil.RiskPercent*10, 100)))
	}
	if mode == "lot" && hasil.Lots >= spec.MaxLot {
		out = append(out, fmt.Sprintf("Ukuran lot menyentuh batas maksimum %.2f lot.", spec.MaxLot))
	}
	return out
}

// defaultValuePerLot adalah cadangan terakhir bila metadata pair dan kategori
// sama-sama tidak tersedia di katalog.
func defaultValuePerLot(symbol string) float64 {
	switch {
	case symbol == "XAUUSD", symbol == "XAUEUR":
		return 100
	case symbol == "XAGUSD":
		return 5_000
	case strings.HasSuffix(symbol, "JPY"):
		return 1_000
	case symbol == "USTEC", symbol == "US30", symbol == "US500", symbol == "GER40":
		return 1
	case symbol == "USOIL":
		return 1_000
	case strings.HasPrefix(symbol, "BTC"), strings.HasPrefix(symbol, "ETH"):
		return 1
	default:
		return 100_000
	}
}
