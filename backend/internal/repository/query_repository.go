package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
)

// Pembacaan daftar untuk dashboard. Semua kueri dibatasi userID agar tidak ada
// kebocoran antar akun (plans/12).

// UserSignalLister membaca sinyal pada simbol yang dipantau user.
type UserSignalLister interface {
	ListForUser(ctx context.Context, userID string, since time.Time, limit int) ([]model.Signal, error)
	// ListRadarForUser membaca baris radar (identitas `status-*`) terpisah dari
	// riwayat. Dipisah karena radar punya siklus hidup berbeda: jumlahnya tetap
	// sebanyak slot, dan `published_at`-nya tidak pernah berubah.
	ListRadarForUser(ctx context.Context, userID string, batas int) ([]model.Signal, error)
}

// UserPositionLister membaca posisi user.
type UserPositionLister interface {
	ListByUser(ctx context.Context, userID string, limit int) ([]model.Position, error)
}

// UserAccountStateLister membaca telemetry akun broker user.
type UserAccountStateLister interface {
	ListByUser(ctx context.Context, userID string) ([]model.BrokerAccountState, error)
}

// TimeframePairRef adalah pasangan timeframe yang pernah terlihat pada sinyal.
type TimeframePairRef struct {
	Htf string
	Ltf string
	// Jumlah sinyal yang membawa pasangan ini, dipakai mengurutkan pilihan.
	Jumlah int
}

// ListKnownTimeframes mengembalikan pasangan timeframe yang pernah muncul pada
// sinyal, dikelompokkan per metode analisis.
//
// Sinyal tanpa htf/ltf (dari engine yang belum mengirimkannya) dilewati, sehingga
// hasil kosong berarti "belum ada data", bukan "tidak ada pasangan".
func (r *signalRepository) ListKnownTimeframes(ctx context.Context) (map[string][]TimeframePairRef, error) {
	type row struct {
		Strategy string `gorm:"column:strategy_version"`
		Htf      string
		Ltf      string
		Jumlah   int
	}

	var rows []row
	err := r.db.WithContext(ctx).
		Model(&model.Signal{}).
		Select("strategy_version, htf, ltf, COUNT(*) AS jumlah").
		Where("htf <> '' AND ltf <> ''").
		Group("strategy_version, htf, ltf").
		Order("strategy_version ASC, htf ASC, ltf ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca pasangan timeframe yang dikenal: %w", err)
	}

	out := make(map[string][]TimeframePairRef, len(rows))
	for _, item := range rows {
		out[item.Strategy] = append(out[item.Strategy], TimeframePairRef{
			Htf: item.Htf, Ltf: item.Ltf, Jumlah: item.Jumlah,
		})
	}
	return out, nil
}

// ListForUser mengembalikan sinyal riwayat pada simbol yang dipantau user.
//
// Baris radar (identitas `status-*`) SENGAJA tidak diambil di sini. Baris itu
// memakai ID tetap per slot sehingga jumlahnya bertambah seiring jumlah slot,
// dan dulu ikut memakan jatah `limit`. Ketika jumlah slot melebihi limit,
// seluruh jatah habis oleh radar dan riwayat tidak pernah terkirim sama sekali.
// Radar kini punya jalannya sendiri lewat ListRadarForUser.
func (r *signalRepository) ListForUser(ctx context.Context, userID string, since time.Time, limit int) ([]model.Signal, error) {
	var signals []model.Signal

	err := r.db.WithContext(ctx).
		Model(&model.Signal{}).
		// EXISTS dipakai, bukan JOIN ke subscription.
		//
		// JOIN membuat satu sinyal bisa muncul berkali-kali bila ada lebih dari
		// satu baris langganan untuk simbol yang sama, sehingga dulu harus
		// ditambal SELECT DISTINCT. Tambalan itu justru merusak: PostgreSQL
		// menolak DISTINCT yang ORDER BY-nya memuat ekspresi di luar select
		// list (SQLSTATE 42P10), sehingga /api/v1/signals gagal 500 dan seluruh
		// dashboard tampak kosong. EXISTS tidak pernah menggandakan baris, jadi
		// DISTINCT tidak lagi dibutuhkan.
		Where("EXISTS (SELECT 1 FROM subscription s WHERE s.symbol = signal.symbol AND s.user_id = ? AND s.is_active = ?)", userID, true).
		Where("signal.published_at >= ?", since).
		Where("signal.signal_id NOT LIKE ?", "status-%").
		Order("signal.published_at DESC").
		Limit(limit).
		Find(&signals).Error
	if err != nil {
		return nil, fmt.Errorf("membaca sinyal user: %w", err)
	}
	return signals, nil
}

// ListRadarForUser mengembalikan baris radar terkini untuk simbol yang dipantau user.
//
// Baris radar memakai identitas tetap per slot dan `published_at`-nya beku sejak
// slot lahir, sedangkan `updated_at` berubah setiap kali engine mengirim kemajuan
// baru. Pengurutan karena itu memakai `updated_at`: dengan `published_at`, slot
// paling basi justru terurut paling atas dan radar terbaca tidak bergerak.
//
// `batas` membatasi jumlah baris agar respons tetap wajar bila jumlah slot tumbuh.
func (r *signalRepository) ListRadarForUser(ctx context.Context, userID string, batas int) ([]model.Signal, error) {
	var signals []model.Signal

	err := r.db.WithContext(ctx).
		Model(&model.Signal{}).
		Where("EXISTS (SELECT 1 FROM subscription s WHERE s.symbol = signal.symbol AND s.user_id = ? AND s.is_active = ?)", userID, true).
		Where("signal.signal_id LIKE ?", "status-%").
		Order("COALESCE(signal.updated_at, signal.published_at) DESC").
		Limit(batas).
		Find(&signals).Error
	if err != nil {
		return nil, fmt.Errorf("membaca radar user: %w", err)
	}
	return signals, nil
}

// ListByUser mengembalikan posisi user, terbaru lebih dulu.
func (r *positionRepository) ListByUser(ctx context.Context, userID string, limit int) ([]model.Position, error) {
	var positions []model.Position

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("opened_at DESC").
		Limit(limit).
		Find(&positions).Error
	if err != nil {
		return nil, fmt.Errorf("membaca posisi user: %w", err)
	}
	return positions, nil
}

type brokerAccountStateRepository struct {
	db *gorm.DB
}

// NewBrokerAccountStateRepository membuat pembaca telemetry akun.
func NewBrokerAccountStateRepository(db *gorm.DB) BrokerAccountStateRepository {
	return &brokerAccountStateRepository{db: db}
}

// ListByUser mengembalikan telemetry terakhir seluruh akun broker user.
func (r *brokerAccountStateRepository) ListByUser(ctx context.Context, userID string) ([]model.BrokerAccountState, error) {
	var states []model.BrokerAccountState

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&states).Error
	if err != nil {
		return nil, fmt.Errorf("membaca status akun: %w", err)
	}
	return states, nil
}
