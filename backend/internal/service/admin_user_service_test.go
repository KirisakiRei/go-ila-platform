package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/pagination"
	"ila/backend/internal/repository"
)

// fakeUserStore adalah penyimpanan pelanggan di memori untuk pengujian.
type fakeUserStore struct {
	rows       map[string]*repository.UserRow
	statuses   map[string]string
	anonymized map[string]bool
	adminCnt   int64
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{
		rows:       map[string]*repository.UserRow{},
		statuses:   map[string]string{},
		anonymized: map[string]bool{},
		adminCnt:   2,
	}
}

func (f *fakeUserStore) put(id, email, status, tier, role string) {
	f.rows[id] = &repository.UserRow{
		User: model.User{Base: model.Base{ID: id}, Name: "Nama " + id, Email: email, Status: status, Tier: tier, Role: role},
	}
	f.statuses[id] = status
}

func (f *fakeUserStore) List(ctx context.Context, filter repository.UserFilter, p pagination.Params) ([]repository.UserRow, error) {
	out := make([]repository.UserRow, 0, len(f.rows))
	for _, row := range f.rows {
		out = append(out, *row)
	}
	return out, nil
}

func (f *fakeUserStore) Count(ctx context.Context, filter repository.UserFilter) (int64, error) {
	return int64(len(f.rows)), nil
}

func (f *fakeUserStore) FindRow(ctx context.Context, id string) (*repository.UserRow, error) {
	row, ok := f.rows[id]
	if !ok {
		return nil, errors.New("tidak ditemukan")
	}
	copy := *row
	copy.Status = f.statuses[id]
	return &copy, nil
}

func (f *fakeUserStore) UpdateStatus(ctx context.Context, id, status string, fields map[string]any) error {
	if _, ok := f.rows[id]; !ok {
		return errors.New("tidak ditemukan")
	}
	f.statuses[id] = status
	return nil
}

func (f *fakeUserStore) UpdateTier(ctx context.Context, id, tier string, startedAt, expiresAt *time.Time) error {
	if _, ok := f.rows[id]; !ok {
		return errors.New("tidak ditemukan")
	}
	f.rows[id].Tier = tier
	return nil
}

func (f *fakeUserStore) Anonymize(ctx context.Context, id string) error {
	f.anonymized[id] = true
	f.statuses[id] = model.UserStatusDeleted
	return nil
}

func (f *fakeUserStore) CountActiveAdmins(ctx context.Context) (int64, error) {
	return f.adminCnt, nil
}

func newTestUserService(store *fakeUserStore, rec *fakeAdminAudit) AdminUserService {
	extras := AdminUserExtras{}
	return NewAdminUserService(store, extras, rec, nil)
}

// fakeAdminAudit mencatat baris audit di memori.
type fakeAdminAudit struct {
	entries []AuditEntry
	failOn  audit.Action
}

func (f *fakeAdminAudit) Record(ctx context.Context, entry AuditEntry) error {
	if entry.Action == f.failOn {
		return errors.New("audit gagal")
	}
	// Alasan untuk aksi destruktif tetap divalidasi meski fake.
	if err := ValidateReason(entry.Action, entry.Reason); err != nil {
		return err
	}
	f.entries = append(f.entries, entry)
	return nil
}

func (f *fakeAdminAudit) has(action audit.Action) bool {
	for _, entry := range f.entries {
		if entry.Action == action {
			return true
		}
	}
	return false
}

func TestSuspendRequiresLongReason(t *testing.T) {
	store := newFakeUserStore()
	store.put("u1", "a@contoh.id", model.UserStatusActive, model.TierFree, "user")
	rec := &fakeAdminAudit{}
	svc := newTestUserService(store, rec)

	err := svc.Suspend(context.Background(), AdminActor{ID: "adm"}, "u1",
		dto.UserActionRequest{Reason: "spam"})
	if err == nil {
		t.Fatal("alasan pendek harus ditolak")
	}
	if store.statuses["u1"] != model.UserStatusActive {
		t.Fatal("status tidak boleh berubah saat alasan ditolak")
	}
}

func TestSuspendRejectsSelf(t *testing.T) {
	store := newFakeUserStore()
	store.put("adm", "adm@contoh.id", model.UserStatusActive, model.TierFree, "admin")
	svc := newTestUserService(store, &fakeAdminAudit{})

	err := svc.Suspend(context.Background(), AdminActor{ID: "adm"}, "adm",
		dto.UserActionRequest{Reason: "alasan yang cukup panjang"})
	if err == nil {
		t.Fatal("suspend akun sendiri harus ditolak")
	}
}

func TestBanRequiresEmailTyped(t *testing.T) {
	store := newFakeUserStore()
	store.put("u1", "a@contoh.id", model.UserStatusActive, model.TierFree, "user")
	rec := &fakeAdminAudit{}
	svc := newTestUserService(store, rec)

	err := svc.Ban(context.Background(), AdminActor{ID: "adm"}, "u1", dto.UserActionRequest{
		Reason: "membagikan sinyal ke pihak lain",
	})
	if err == nil {
		t.Fatal("ban tanpa ketik ulang email harus ditolak")
	}

	err = svc.Ban(context.Background(), AdminActor{ID: "adm"}, "u1", dto.UserActionRequest{
		Reason:       "membagikan sinyal ke pihak lain",
		ConfirmEmail: "salah@contoh.id",
	})
	if err == nil {
		t.Fatal("email yang tidak cocok harus ditolak")
	}

	err = svc.Ban(context.Background(), AdminActor{ID: "adm"}, "u1", dto.UserActionRequest{
		Reason:       "membagikan sinyal ke pihak lain",
		ConfirmEmail: "a@contoh.id",
	})
	if err != nil {
		t.Fatalf("ban sah harus berhasil: %v", err)
	}
	if store.statuses["u1"] != model.UserStatusBanned {
		t.Fatal("status harus menjadi banned")
	}
	if !rec.has(audit.UserBan) {
		t.Fatal("ban harus tercatat di audit")
	}
}

func TestDeleteRequiresAcknowledgeAndEmail(t *testing.T) {
	store := newFakeUserStore()
	store.put("u1", "a@contoh.id", model.UserStatusActive, model.TierFree, "user")
	svc := newTestUserService(store, &fakeAdminAudit{})

	err := svc.Delete(context.Background(), AdminActor{ID: "adm"}, "u1", dto.UserActionRequest{
		Reason:       "permintaan pengguna untuk menutup akun",
		ConfirmEmail: "a@contoh.id",
	})
	if err == nil {
		t.Fatal("hapus tanpa centang harus ditolak")
	}

	err = svc.Delete(context.Background(), AdminActor{ID: "adm"}, "u1", dto.UserActionRequest{
		Reason:      "permintaan pengguna untuk menutup akun",
		Acknowledge: true,
	})
	if err == nil {
		t.Fatal("hapus tanpa email harus ditolak")
	}
}

func TestDeleteAnonymizesUser(t *testing.T) {
	store := newFakeUserStore()
	store.put("u1", "a@contoh.id", model.UserStatusActive, model.TierFree, "user")
	rec := &fakeAdminAudit{}
	svc := newTestUserService(store, rec)

	err := svc.Delete(context.Background(), AdminActor{ID: "adm"}, "u1", dto.UserActionRequest{
		Reason:       "permintaan pengguna untuk menutup akun",
		ConfirmEmail: "a@contoh.id",
		Acknowledge:  true,
	})
	if err != nil {
		t.Fatalf("hapus sah harus berhasil: %v", err)
	}
	if !store.anonymized["u1"] {
		t.Fatal("baris user harus dianonimkan, bukan dihapus")
	}
	if !rec.has(audit.UserDelete) {
		t.Fatal("hapus harus tercatat di audit")
	}
}

func TestChangeTierRejectsSelf(t *testing.T) {
	store := newFakeUserStore()
	store.put("adm", "adm@contoh.id", model.UserStatusActive, model.TierFree, "admin")
	svc := newTestUserService(store, &fakeAdminAudit{})

	err := svc.ChangeTier(context.Background(), AdminActor{ID: "adm"}, "adm",
		dto.UserTierRequest{Tier: model.TierPlus, Reason: "mau coba sendiri"})
	if err == nil {
		t.Fatal("menaikkan tier sendiri harus ditolak")
	}
}

func TestAuditFailureBlocksAction(t *testing.T) {
	store := newFakeUserStore()
	store.put("u1", "a@contoh.id", model.UserStatusActive, model.TierFree, "user")
	rec := &fakeAdminAudit{failOn: audit.UserSuspend}
	svc := newTestUserService(store, rec)

	err := svc.Suspend(context.Background(), AdminActor{ID: "adm"}, "u1",
		dto.UserActionRequest{Reason: "gagal bayar 14 hari"})
	if err == nil {
		t.Fatal("kegagalan audit harus menggagalkan aksi")
	}
}

func TestUnknownReasonCategoryRejected(t *testing.T) {
	store := newFakeUserStore()
	store.put("u1", "a@contoh.id", model.UserStatusActive, model.TierFree, "user")
	svc := newTestUserService(store, &fakeAdminAudit{})

	err := svc.Suspend(context.Background(), AdminActor{ID: "adm"}, "u1",
		dto.UserActionRequest{Reason: "alasan yang cukup panjang", Category: "ngawur"})
	if err == nil {
		t.Fatal("kategori alasan tak dikenal harus ditolak")
	}
}
