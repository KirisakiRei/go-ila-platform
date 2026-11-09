package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/service"
	"ila/backend/pkg/apperror"
)

// ---------------------------------------------------------------------------
// Ganda palsu untuk alur konfirmasi entry
// ---------------------------------------------------------------------------

type fakeTokenStore struct {
	token  *model.ApprovalToken
	marked bool
	usedAt time.Time
	// diterbitkan menyimpan token yang dibuat lewat Issue.
	diterbitkan []*model.ApprovalToken
}

func (s *fakeTokenStore) FindByHash(_ context.Context, hash string) (*model.ApprovalToken, error) {
	if s.token == nil || s.token.TokenHash != hash {
		return nil, apperror.ErrNotFound
	}
	return s.token, nil
}

func (s *fakeTokenStore) MarkUsed(_ context.Context, _ string, at time.Time) error {
	if s.marked {
		return apperror.ErrConflict
	}
	s.marked = true
	s.usedAt = at
	return nil
}

// Create menyimpan token yang diterbitkan untuk tombol dashboard.
func (s *fakeTokenStore) Create(_ context.Context, token *model.ApprovalToken) error {
	s.diterbitkan = append(s.diterbitkan, token)
	s.token = token
	return nil
}

type fakeUserSignalStore struct {
	signal *model.UserSignal
}

func (s *fakeUserSignalStore) FindUserSignal(_ context.Context, _, _ string) (*model.UserSignal, error) {
	if s.signal == nil {
		return nil, apperror.ErrNotFound
	}
	return s.signal, nil
}

type fakeSignalStore struct {
	signal *model.Signal
}

func (s *fakeSignalStore) FindSignal(_ context.Context, _ string) (*model.Signal, error) {
	if s.signal == nil {
		return nil, apperror.ErrNotFound
	}
	return s.signal, nil
}

type fakeAuditRecorder struct {
	events []string
}

func (a *fakeAuditRecorder) Record(_ context.Context, entry *model.AuditLog) error {
	a.events = append(a.events, entry.Event)
	return nil
}

type fakeRunner struct {
	approveReason string
	approveErr    error
	approved      bool
	skipped       bool
}

func (r *fakeRunner) Approve(_ context.Context, _, _, _ string) (string, error) {
	r.approved = true
	return r.approveReason, r.approveErr
}

func (r *fakeRunner) Skip(_ context.Context, _, _ string) error {
	r.skipped = true
	return nil
}

// ---------------------------------------------------------------------------
// Fixture
// ---------------------------------------------------------------------------

const rawToken = "token-uji-abc"

func freshSignal() *model.Signal {
	return &model.Signal{
		SignalID:        "sig-1",
		Symbol:          "XAUUSD",
		Direction:       "BUY",
		StrategyVersion: "ila-2.0",
		EntryPrice:      4120.50,
		StopLoss:        4112.30,
		TakeProfit:      4149.20,
		RiskReward:      2.30,
		Status:          model.SignalStatusReadyToEntry,
		ExpiresAt:       time.Now().UTC().Add(15 * time.Minute),
	}
}

func freshToken(userID string) *model.ApprovalToken {
	return &model.ApprovalToken{
		ID:        "tok-1",
		TokenHash: service.HashApprovalToken(rawToken),
		UserID:    userID,
		SignalID:  "sig-1",
		Action:    "ENTRY",
		ExpiresAt: time.Now().UTC().Add(5 * time.Minute),
	}
}

type harness struct {
	tokens  *fakeTokenStore
	users   *fakeUserSignalStore
	signals *fakeSignalStore
	audit   *fakeAuditRecorder
	runner  *fakeRunner
	svc     service.ApproveService
}

func newHarness() *harness {
	h := &harness{
		tokens:  &fakeTokenStore{},
		users:   &fakeUserSignalStore{},
		signals: &fakeSignalStore{},
		audit:   &fakeAuditRecorder{},
		runner:  &fakeRunner{},
	}
	h.svc = service.NewApproveService(h.tokens, h.users, h.signals, h.audit, h.runner)
	return h
}

// ---------------------------------------------------------------------------
// Tes
// ---------------------------------------------------------------------------

func TestConfirmExecutesValidToken(t *testing.T) {
	h := newHarness()
	h.tokens.token = freshToken("u1")
	h.signals.signal = freshSignal()
	h.users.signal = &model.UserSignal{Status: model.SignalStatusReadyToEntry, BrokerAccountID: "acct-1"}

	result, err := h.svc.Confirm(context.Background(), "u1", rawToken)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if !h.runner.approved {
		t.Fatal("runner harus dipanggil")
	}
	if result.Status != model.SignalStatusConsumed {
		t.Fatalf("status = %q, ingin CONSUMED", result.Status)
	}
	if !h.tokens.marked {
		t.Fatal("token harus ditandai terpakai")
	}
	if len(h.audit.events) == 0 {
		t.Fatal("keputusan user harus tercatat di audit")
	}
}

// Token milik orang lain tidak boleh bisa dipakai.
func TestConfirmRejectsForeignToken(t *testing.T) {
	h := newHarness()
	h.tokens.token = freshToken("u2")
	h.signals.signal = freshSignal()

	_, err := h.svc.Confirm(context.Background(), "u1", rawToken)
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("err = %v, ingin ErrForbidden", err)
	}
	if h.runner.approved {
		t.Fatal("runner tidak boleh dipanggil")
	}
}

// Token yang sudah dipakai tidak bisa dipakai ulang.
func TestConfirmRejectsUsedToken(t *testing.T) {
	h := newHarness()
	used := time.Now().UTC()
	token := freshToken("u1")
	token.UsedAt = &used
	h.tokens.token = token
	h.signals.signal = freshSignal()

	_, err := h.svc.Confirm(context.Background(), "u1", rawToken)
	if !errors.Is(err, apperror.ErrConflict) {
		t.Fatalf("err = %v, ingin ErrConflict", err)
	}
}

func TestConfirmRejectsExpiredToken(t *testing.T) {
	h := newHarness()
	token := freshToken("u1")
	token.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	h.tokens.token = token
	h.signals.signal = freshSignal()

	_, err := h.svc.Confirm(context.Background(), "u1", rawToken)
	if !errors.Is(err, apperror.ErrConflict) {
		t.Fatalf("err = %v, ingin ErrConflict", err)
	}
	if h.runner.approved {
		t.Fatal("token kedaluwarsa tidak boleh mengeksekusi")
	}
}

// Sinyal yang sudah batal menonaktifkan tombol beserta alasannya.
func TestPreviewBlocksCancelledSignal(t *testing.T) {
	h := newHarness()
	h.tokens.token = freshToken("u1")
	signal := freshSignal()
	signal.Status = model.SignalStatusCancelled
	h.signals.signal = signal

	preview, err := h.svc.Resolve(context.Background(), "u1", rawToken)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if preview.CanConfirm {
		t.Fatal("sinyal batal tidak boleh bisa dikonfirmasi")
	}
	if preview.BlockedReason == "" {
		t.Fatal("alasan harus ditampilkan, bukan dikosongkan")
	}
}

func TestPreviewBlocksExpiredSignal(t *testing.T) {
	h := newHarness()
	h.tokens.token = freshToken("u1")
	signal := freshSignal()
	signal.ExpiresAt = time.Now().UTC().Add(-time.Second)
	h.signals.signal = signal

	preview, err := h.svc.Resolve(context.Background(), "u1", rawToken)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if preview.CanConfirm || preview.BlockedReason == "" {
		t.Fatal("sinyal kedaluwarsa harus nonaktif dengan alasan")
	}
}

// Penolakan aturan harian ikut tampil sebagai alasan.
func TestPreviewShowsDailyRejectionReason(t *testing.T) {
	h := newHarness()
	h.tokens.token = freshToken("u1")
	h.signals.signal = freshSignal()
	h.users.signal = &model.UserSignal{
		Status:       model.SignalStatusRejected,
		RejectReason: "Target harian tercapai (1 win). Trading dihentikan hari ini.",
	}

	preview, err := h.svc.Resolve(context.Background(), "u1", rawToken)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if preview.CanConfirm {
		t.Fatal("sinyal tertahan tidak boleh dikonfirmasi")
	}
	if preview.BlockedReason != h.users.signal.RejectReason {
		t.Fatalf("alasan = %q", preview.BlockedReason)
	}
}

// Confirm menolak walau tombol sempat aktif saat halaman dirender.
func TestConfirmRechecksBlockedState(t *testing.T) {
	h := newHarness()
	h.tokens.token = freshToken("u1")
	signal := freshSignal()
	signal.Status = model.SignalStatusCancelled
	h.signals.signal = signal
	h.users.signal = &model.UserSignal{Status: model.SignalStatusCancelled, BrokerAccountID: "acct-1"}

	_, err := h.svc.Confirm(context.Background(), "u1", rawToken)
	if err == nil {
		t.Fatal("Confirm harus menolak sinyal yang sudah batal")
	}
	if h.runner.approved {
		t.Fatal("runner tidak boleh dipanggil untuk sinyal batal")
	}
}

// Tanpa akun broker, konfirmasi ditolak dengan alasan yang jelas.
func TestConfirmRequiresBrokerAccount(t *testing.T) {
	h := newHarness()
	h.tokens.token = freshToken("u1")
	h.signals.signal = freshSignal()
	h.users.signal = &model.UserSignal{Status: model.SignalStatusReadyToEntry}

	_, err := h.svc.Confirm(context.Background(), "u1", rawToken)
	if err == nil {
		t.Fatal("tanpa akun broker harus ditolak")
	}
}

// Penolakan dari Runner diteruskan apa adanya ke user.
func TestConfirmPropagatesRunnerRejection(t *testing.T) {
	h := newHarness()
	h.tokens.token = freshToken("u1")
	h.signals.signal = freshSignal()
	h.users.signal = &model.UserSignal{Status: model.SignalStatusReadyToEntry, BrokerAccountID: "acct-1"}
	h.runner.approveReason = "Masih ada posisi terbuka. Tunggu sampai ditutup."

	result, err := h.svc.Confirm(context.Background(), "u1", rawToken)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if result.Status != model.SignalStatusRejected {
		t.Fatalf("status = %q, ingin REJECTED", result.Status)
	}
	if result.Message != h.runner.approveReason {
		t.Fatalf("pesan = %q", result.Message)
	}
}

// Lewati menandai sinyal dan memakai token.
func TestDeclineSkipsAndConsumesToken(t *testing.T) {
	h := newHarness()
	h.tokens.token = freshToken("u1")
	h.signals.signal = freshSignal()

	result, err := h.svc.Decline(context.Background(), "u1", rawToken)
	if err != nil {
		t.Fatalf("Decline: %v", err)
	}
	if !h.runner.skipped {
		t.Fatal("runner harus menerima aksi skip")
	}
	if !h.tokens.marked {
		t.Fatal("token harus ditandai terpakai")
	}
	if result.Status != model.SignalStatusSkipped {
		t.Fatalf("status = %q, ingin SKIPPED", result.Status)
	}
}

func TestResolveRejectsUnknownToken(t *testing.T) {
	h := newHarness()
	h.signals.signal = freshSignal()

	_, err := h.svc.Resolve(context.Background(), "u1", "token-palsu")
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("err = %v, ingin ErrNotFound", err)
	}
}

// TestIssueMenerbitkanTokenUntukTombolDashboard memastikan tombol eksekusi di
// dashboard mendapat token yang sah, bukan hanya tautan Telegram.
func TestIssueMenerbitkanTokenUntukTombolDashboard(t *testing.T) {
	h := newHarness()
	h.signals.signal = freshSignal()
	h.users.signal = &model.UserSignal{
		Status: model.SignalStatusReadyToEntry, BrokerAccountID: "acct-1",
	}

	hasil, err := h.svc.Issue(context.Background(), "u1", "sig-1")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if !hasil.CanConfirm {
		t.Fatalf("seharusnya bisa dikonfirmasi, terblokir: %q", hasil.BlockedReason)
	}
	if hasil.Token == "" {
		t.Fatal("token tidak boleh kosong")
	}
	if len(h.tokens.diterbitkan) != 1 {
		t.Fatalf("token tersimpan = %d, ingin 1", len(h.tokens.diterbitkan))
	}
	// Yang disimpan hanya hash; token mentah tidak boleh ikut tersimpan.
	simpan := h.tokens.diterbitkan[0]
	if simpan.TokenHash == hasil.Token {
		t.Fatal("basis data tidak boleh menyimpan token mentah")
	}
	if simpan.TokenHash != service.HashApprovalToken(hasil.Token) {
		t.Fatal("hash token tidak cocok dengan token yang dikirim")
	}
}

// TestIssuePadaPeluangTetapBisaDikonfirmasi adalah inti fitur: sinyal peluang
// TIDAK dieksekusi otomatis, tetapi user harus tetap bisa memilih
// mengeksekusinya. Bila pesan peluang ikut terbaca sebagai alasan blokir,
// pilihan itu jadi mustahil.
func TestIssuePadaPeluangTetapBisaDikonfirmasi(t *testing.T) {
	h := newHarness()
	h.signals.signal = freshSignal()
	h.users.signal = &model.UserSignal{
		Status:          model.SignalStatusOpportunity,
		RejectReason:    "Setup pasangan W1-H4 tersedia untuk dipantau. Eksekusi otomatis tidak aktif untuk akun ini.",
		BrokerAccountID: "acct-1",
	}

	hasil, err := h.svc.Issue(context.Background(), "u2", "sig-1")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if !hasil.CanConfirm {
		t.Fatalf("peluang harus bisa dieksekusi manual, terblokir: %q", hasil.BlockedReason)
	}
}

// TestIssueDitolakSaatKedaluwarsa memastikan tombol tidak diberi token ketika
// sinyal sudah lewat batas waktu — keputusan A yang disepakati.
func TestIssueDitolakSaatKedaluwarsa(t *testing.T) {
	h := newHarness()
	sig := freshSignal()
	sig.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	h.signals.signal = sig

	hasil, err := h.svc.Issue(context.Background(), "u1", "sig-1")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if hasil.CanConfirm {
		t.Fatal("sinyal kedaluwarsa tidak boleh mendapat token")
	}
	if hasil.BlockedReason == "" {
		t.Fatal("alasan harus diisi agar user tahu kenapa tombol mati")
	}
	if len(h.tokens.diterbitkan) != 0 {
		t.Fatalf("token tidak boleh diterbitkan, dapat %d", len(h.tokens.diterbitkan))
	}
}
