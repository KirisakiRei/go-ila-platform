package repository

import (
	"context"
	"time"

	"ila/backend/internal/authstate"
	"ila/backend/internal/model"
)

// Interface di paket ini adalah tipe kembalian konstruktor.
// Service mendefinisikan kontraknya sendiri (lihat internal/service/interfaces.go),
// sehingga layer atas tidak terikat pada paket ini.

// UserRepository adalah kontrak penyimpanan akun.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	// FindByIdentifier mencocokkan email atau username untuk satu kolom masuk.
	FindByIdentifier(ctx context.Context, identifier string) (*model.User, error)
	FindByTelegramChatID(ctx context.Context, chatID string) (*model.User, error)
	UpdateLastLogin(ctx context.Context, id string, at time.Time) error
	Update(ctx context.Context, user *model.User) error
	CountByTier(ctx context.Context, tier string) (int64, error)
	// ListTrials membaca pengguna yang memakai atau pernah memakai uji coba.
	ListTrials(ctx context.Context, status string, limit int) ([]model.User, error)

	// FindSessionUser membaca kolom minimal untuk validasi sesi tiap request.
	FindSessionUser(ctx context.Context, id string) (*authstate.User, error)
	// RevokeSessions menaikkan session_version: seluruh token lama ditolak.
	RevokeSessions(ctx context.Context, id string) error
	// UpdatePassword mengganti hash dan menaikkan session_version sekaligus.
	UpdatePassword(ctx context.Context, id, passwordHash string) error
}

// BrokerAccountRepository adalah kontrak penyimpanan akun broker.
type BrokerAccountRepository interface {
	Create(ctx context.Context, account *model.BrokerAccount) error
	FindByID(ctx context.Context, id string) (*model.BrokerAccount, error)
	ListByUser(ctx context.Context, userID string) ([]model.BrokerAccount, error)
	CountActiveByUser(ctx context.Context, userID string) (int64, error)
	Update(ctx context.Context, account *model.BrokerAccount) error
}

// CredentialRepository menyimpan dan mengambil blob kredensial terenkripsi.
type CredentialRepository interface {
	Upsert(ctx context.Context, credential *model.Credential) error
	FindByAccount(ctx context.Context, accountID string) (*model.Credential, error)
	DeleteByAccount(ctx context.Context, accountID string) error
}

// SubscriptionRepository mengelola simbol dan strategi yang dipantau user.
type SubscriptionRepository interface {
	Create(ctx context.Context, sub *model.Subscription) error
	ListActiveByUser(ctx context.Context, userID string) ([]model.Subscription, error)
	ListByUser(ctx context.Context, userID string) ([]model.Subscription, error)
	FindActive(ctx context.Context, userID, symbol string) (*model.Subscription, error)
	FindByID(ctx context.Context, id, userID string) (*model.Subscription, error)
	Deactivate(ctx context.Context, id, userID string) error
	Delete(ctx context.Context, id, userID string) error
	DeleteBySymbol(ctx context.Context, userID, symbol string) error
	Reactivate(ctx context.Context, id, userID string) error

	// ListActiveByAccount membaca langganan aktif yang berlaku untuk satu akun.
	// Langganan lama tanpa broker_account_id (NULL) ikut dianggap berlaku,
	// sehingga data sebelum katalog pair tetap dieksekusi.
	ListActiveByAccount(ctx context.Context, userID, accountID string) ([]model.Subscription, error)
	// ReplaceAccountPairs mengganti seluruh daftar simbol satu akun dalam satu
	// transaksi: simbol lama dinonaktifkan, simbol baru diaktifkan.
	ReplaceAccountPairs(ctx context.Context, userID, accountID string, symbols []string) error
}

// PairRepository menyimpan katalog kategori, grup, dan pair yang dipilih user.
type PairRepository interface {
	CreateCategory(ctx context.Context, category *model.PairCategory) error
	FindCategoryByID(ctx context.Context, id string) (*model.PairCategory, error)
	FindCategoryByKey(ctx context.Context, key string) (*model.PairCategory, error)
	ListCategories(ctx context.Context, onlyEnabled bool) ([]model.PairCategory, error)
	UpdateCategory(ctx context.Context, category *model.PairCategory) error
	DeleteCategory(ctx context.Context, id string) error

	CreateGroup(ctx context.Context, group *model.PairGroup) error
	FindGroupByID(ctx context.Context, id string) (*model.PairGroup, error)
	ListGroups(ctx context.Context, categoryID string, onlyEnabled bool) ([]model.PairGroup, error)
	UpdateGroup(ctx context.Context, group *model.PairGroup) error
	DeleteGroup(ctx context.Context, id string) error

	CreatePair(ctx context.Context, pair *model.TradingPair) error
	FindPairByID(ctx context.Context, id string) (*model.TradingPair, error)
	FindPairBySymbol(ctx context.Context, symbol string) (*model.TradingPair, error)
	ListPairs(ctx context.Context, filter PairFilter) ([]model.TradingPair, error)
	UpdatePair(ctx context.Context, pair *model.TradingPair) error
	DeletePair(ctx context.Context, id string) error
	CountActiveSubscriptionsBySymbol(ctx context.Context, symbol string) (int64, error)
}

// BrokerAccountStateRepository menyimpan telemetry akun yang ditulis Runner.
type BrokerAccountStateRepository interface {
	ListByUser(ctx context.Context, userID string) ([]model.BrokerAccountState, error)
}

// SignalRepository menyimpan sinyal yang diterima dari engine.
type SignalRepository interface {
	Upsert(ctx context.Context, signal *model.Signal) error
	FindBySignalID(ctx context.Context, signalID string) (*model.Signal, error)
	UpdateStatus(ctx context.Context, signalID, status, reason string) error
	ListForUser(ctx context.Context, userID string, since time.Time, limit int) ([]model.Signal, error)
	// ListRadarForUser membaca baris radar terpisah dari riwayat supaya keduanya
	// tidak saling memakan jatah limit.
	ListRadarForUser(ctx context.Context, userID string, batas int) ([]model.Signal, error)
	// ListKnownTimeframes mengembalikan pasangan timeframe yang pernah benar-benar
	// terlihat pada sinyal, dikelompokkan per metode analisis.
	//
	// Ini satu-satunya sumber jujur untuk menampilkan pilihan pasangan sebelum
	// engine mengirimkan katalog resminya: yang ditampilkan adalah pasangan yang
	// memang terbukti menghasilkan sinyal, bukan tebakan platform.
	ListKnownTimeframes(ctx context.Context) (map[string][]TimeframePairRef, error)
}

// PositionRepository mengelola posisi nyata di broker.
type PositionRepository interface {
	Create(ctx context.Context, pos *model.Position) error
	FindOpenByUser(ctx context.Context, userID string) ([]model.Position, error)
	ListByUser(ctx context.Context, userID string, limit int) ([]model.Position, error)
	FindByTicket(ctx context.Context, ticket string) (*model.Position, error)
	Close(ctx context.Context, id string, exitPrice, pnl, outcomeR float64, status string) error
}

// ApprovalTokenRepository menyimpan token konfirmasi entry berbentuk hash.
type ApprovalTokenRepository interface {
	Create(ctx context.Context, token *model.ApprovalToken) error
	FindByHash(ctx context.Context, hash string) (*model.ApprovalToken, error)
	MarkUsed(ctx context.Context, id string, at time.Time) error
}

// AuditRepository mencatat peristiwa ke audit log append-only.
type AuditRepository interface {
	Record(ctx context.Context, entry *model.AuditLog) error
}
