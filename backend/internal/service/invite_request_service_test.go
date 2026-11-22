package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ila/backend/internal/dto"
	"ila/backend/internal/email"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/service"
	"ila/backend/pkg/apperror"
)

func newInviteReqTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.InviteCode{}, &model.InviteRedemption{}, &model.InviteRequest{}, &model.EmailOutbox{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestInviteRequestSubmitAndApprove(t *testing.T) {
	db := newInviteReqTestDB(t)
	ctx := context.Background()

	inviteRepo := repository.NewInviteRepository(db)
	inviteSvc := service.NewInviteService(inviteRepo, nil, nil)
	reqRepo := repository.NewInviteRequestRepository(db)
	emailRepo := repository.NewEmailRepository(db)
	svc := service.NewInviteRequestService(reqRepo, inviteSvc, emailRepo, true, nil, nil)

	// Honeypot terisi: sukses diam-diam tanpa menulis baris.
	if err := svc.Submit(ctx, dto.SubmitInviteRequest{
		Name:    "Bot",
		Email:   "bot@example.com",
		Website: "https://spam.invalid",
	}); err != nil {
		t.Fatalf("honeypot submit error: %v", err)
	}
	listAfterBot, _, err := svc.List(ctx, "", 1, 20)
	if err != nil || len(listAfterBot.Items) != 0 {
		t.Fatalf("honeypot tidak boleh menyimpan baris; got %d, err=%v", len(listAfterBot.Items), err)
	}

	// Submit normal berhasil, submit kedua dengan email sama ditolak conflict.
	if err := svc.Submit(ctx, dto.SubmitInviteRequest{
		Name:    "Budi Trader",
		Email:   "Budi@Example.com",
		Contact: "@buditrader",
		Note:    "Fokus XAUUSD & EURUSD sesi NY",
	}); err != nil {
		t.Fatalf("submit normal gagal: %v", err)
	}
	err = svc.Submit(ctx, dto.SubmitInviteRequest{
		Name:  "Budi Lagi",
		Email: "budi@example.com",
	})
	if !errors.Is(err, apperror.ErrConflict) {
		t.Fatalf("duplikat pending harus ErrConflict, got: %v", err)
	}

	listed, meta, err := svc.List(ctx, model.InviteRequestPending, 1, 20)
	if err != nil {
		t.Fatalf("list gagal: %v", err)
	}
	if meta.TotalItems != 1 || listed.PendingCount != 1 {
		t.Fatalf("unexpected counts: total=%d pending=%d", meta.TotalItems, listed.PendingCount)
	}

	// Approve membuat undangan terkunci email + mengantrekan email outbox.
	actor := service.AdminActor{ID: "admin-1", RequestID: "req-1"}
	approved, err := svc.Approve(ctx, actor, listed.Items[0].ID, dto.ApproveInviteRequestBody{
		Tier:        model.TierPlus,
		ExpiresDays: 7,
	})
	if err != nil {
		t.Fatalf("approve gagal: %v", err)
	}
	if approved.Invite.Code == "" || approved.Invite.Email != "budi@example.com" || approved.Invite.MaxUses != 1 {
		t.Fatalf("invite hasil approve tidak sesuai: %+v", approved.Invite)
	}
	if !approved.EmailQueued {
		t.Fatalf("email_queued seharusnya true saat smtpEnabled=true")
	}

	// Pastikan email outbox dapat dirender menjadi pesan valid dengan tautan kode.
	pendingOutbox, err := emailRepo.Pending(ctx, 10)
	if err != nil || len(pendingOutbox) != 1 {
		t.Fatalf("outbox pending harus 1, got %d (err=%v)", len(pendingOutbox), err)
	}
	email.PublicBaseURL = "https://rekavia.com"
	msg, err := email.Render(pendingOutbox[0], emailRepo)
	if err != nil {
		t.Fatalf("render invite_approved gagal: %v", err)
	}
	expectedLink := "https://rekavia.com/daftar?invite=" + approved.Invite.Code
	if msg.To != "budi@example.com" || !containsString(msg.Body, expectedLink) {
		t.Fatalf("isi surel tidak memuat tautan yang benar: to=%s body=%s", msg.To, msg.Body)
	}

	// Approve kedua pada baris yang sama ditolak conflict.
	if _, err := svc.Approve(ctx, actor, listed.Items[0].ID, dto.ApproveInviteRequestBody{}); !errors.Is(err, apperror.ErrConflict) {
		t.Fatalf("approve ulang harus ErrConflict, got: %v", err)
	}
}

func TestInviteRequestReject(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.InviteRequest{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	ctx := context.Background()
	reqRepo := repository.NewInviteRequestRepository(db)
	svc := service.NewInviteRequestService(reqRepo, nil, nil, false, nil, nil)

	if err := svc.Submit(ctx, dto.SubmitInviteRequest{Name: "Sari", Email: "sari@example.com"}); err != nil {
		t.Fatalf("submit: %v", err)
	}
	listed, _, err := svc.List(ctx, "", 1, 10)
	if err != nil || len(listed.Items) != 1 {
		t.Fatalf("list: %v", err)
	}
	if err := svc.Reject(ctx, service.AdminActor{ID: "admin-1"}, listed.Items[0].ID, "kuota batch penuh"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	row, err := reqRepo.FindByID(ctx, listed.Items[0].ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if row.Status != model.InviteRequestRejected || row.ReviewedAt == nil || row.ReviewedAt.After(time.Now().Add(time.Minute)) {
		t.Fatalf("status reject tidak sesuai: %+v", row)
	}

	// Email yang baru ditolak tidak boleh langsung spam ulang dalam jendela cooldown 24 jam.
	err = svc.Submit(ctx, dto.SubmitInviteRequest{Name: "Sari Lagi", Email: "sari@example.com"})
	if !errors.Is(err, apperror.ErrTooManyRequests) {
		t.Fatalf("submit ulang dalam 24 jam harus ErrTooManyRequests, got: %v", err)
	}
}

func containsString(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
