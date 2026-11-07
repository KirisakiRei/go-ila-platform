package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"ila/backend/internal/audit"
	"ila/backend/internal/authz"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/pagination"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

// AdminUserService adalah manajemen pelanggan dari panel admin.
//
// Batas pandangnya tetap: tidak ada metode yang mengembalikan kredensial, dan
// saldo hanya keluar lewat jalur yang tercatat audit (plans/07 §2).
type AdminUserService interface {
	List(ctx context.Context, actor AdminActor, filter dto.AdminUserFilter) ([]dto.UserListItem, pagination.Meta, error)
	Detail(ctx context.Context, actor AdminActor, userID string) (*dto.UserDetail, error)
	Accounts(ctx context.Context, actor AdminActor, userID string) ([]dto.AdminAccountItem, error)
	Positions(ctx context.Context, actor AdminActor, userID string) ([]dto.AdminPositionItem, error)
	Legal(ctx context.Context, actor AdminActor, userID string) ([]dto.AdminLegalAcceptance, error)
	Notes(ctx context.Context, actor AdminActor, userID string) ([]dto.AdminNoteItem, error)
	AddNote(ctx context.Context, actor AdminActor, userID string, req dto.UserNoteRequest) error
	Suspend(ctx context.Context, actor AdminActor, userID string, req dto.UserActionRequest) error
	Unsuspend(ctx context.Context, actor AdminActor, userID string, req dto.UserActionRequest) error
	Ban(ctx context.Context, actor AdminActor, userID string, req dto.UserActionRequest) error
	Unban(ctx context.Context, actor AdminActor, userID string, req dto.UserActionRequest) error
	ChangeTier(ctx context.Context, actor AdminActor, userID string, req dto.UserTierRequest) error
	Delete(ctx context.Context, actor AdminActor, userID string, req dto.UserActionRequest) error
	FreezeAccount(ctx context.Context, actor AdminActor, accountID string, req dto.AccountFreezeRequest) error
	UnfreezeAccount(ctx context.Context, actor AdminActor, accountID string, req dto.AccountFreezeRequest) error
	ExportUsers(ctx context.Context, actor AdminActor, filter dto.AdminUserFilter) ([]byte, string, error)
}

// AdminActor adalah pelaku aksi admin. RequestID dipakai menelusuri satu
// permintaan lintas baris audit.
type AdminActor struct {
	ID        string
	RequestID string
}

// Store kebutuhan manajemen pelanggan.
type adminUserStore interface {
	List(ctx context.Context, filter repository.UserFilter, p pagination.Params) ([]repository.UserRow, error)
	Count(ctx context.Context, filter repository.UserFilter) (int64, error)
	FindRow(ctx context.Context, id string) (*repository.UserRow, error)
	UpdateStatus(ctx context.Context, id, status string, fields map[string]any) error
	UpdateTier(ctx context.Context, id, tier string, startedAt, expiresAt *time.Time) error
	Anonymize(ctx context.Context, id string) error
	CountActiveAdmins(ctx context.Context) (int64, error)
}

type adminUserService struct {
	store    adminUserStore
	extras   AdminUserExtras
	audit    AdminAudit
	sessions SessionRevoker
	now      func() time.Time
}

// AdminUserExtras adalah kebutuhan pendukung: akun, posisi, legal, catatan.
type AdminUserExtras struct {
	Accounts  AdminAccountStore
	Positions AdminPositionStore
	Legal     AdminLegalStore
	Notes     AdminNoteStore
}

// NewAdminUserService merangkai manajemen pelanggan.
func NewAdminUserService(store adminUserStore, extras AdminUserExtras, recorder AdminAudit, sessions SessionRevoker) AdminUserService {
	return &adminUserService{
		store:    store,
		extras:   extras,
		audit:    recorder,
		sessions: sessions,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// DestructiveReason memastikan alasan cukup panjang dan kategorinya dikenal.
func destructiveReason(action audit.Action, req dto.UserActionRequest) (string, string, error) {
	if err := ValidateReason(action, req.Reason); err != nil {
		return "", "", err
	}
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = model.ReasonOther
	}
	if !model.KnownReasonCategory(category) {
		return "", "", validation("kategori alasan tidak dikenal")
	}
	return strings.TrimSpace(req.Reason), category, nil
}

func (s *adminUserService) List(ctx context.Context, actor AdminActor, filter dto.AdminUserFilter) ([]dto.UserListItem, pagination.Meta, error) {
	params, storeFilter, err := buildUserList(filter)
	if err != nil {
		return nil, pagination.Meta{}, err
	}

	rows, err := s.store.List(ctx, storeFilter, params)
	if err != nil {
		return nil, pagination.Meta{}, err
	}
	total, err := s.store.Count(ctx, storeFilter)
	if err != nil {
		return nil, pagination.Meta{}, err
	}

	items := make([]dto.UserListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toUserListItem(row))
	}
	return items, pagination.NewMeta(params, total), nil
}

func (s *adminUserService) Detail(ctx context.Context, actor AdminActor, userID string) (*dto.UserDetail, error) {
	row, err := s.store.FindRow(ctx, userID)
	if err != nil {
		return nil, err
	}

	detail := dto.UserDetail{
		UserListItem:  toUserListItem(*row),
		TrialUsed:     row.TrialUsed,
		TrialExpires:  dto.NewTimePairPtr(row.TrialExpiresAt),
		GraceUntil:    dto.NewTimePairPtr(row.GraceUntil),
		SuspendedAt:   dto.NewTimePairPtr(row.SuspendedAt),
		BannedAt:      dto.NewTimePairPtr(row.BannedAt),
		BanReason:     row.BanReason,
		BanCategory:   row.BanCategory,
		InternalNote:  row.InternalNote,
		IdleWarning:   row.IdleWarning,
		NotifyEnabled: row.NotifyEnabled,
	}
	return &detail, nil
}

func (s *adminUserService) Accounts(ctx context.Context, actor AdminActor, userID string) ([]dto.AdminAccountItem, error) {
	if s.extras.Accounts == nil {
		return []dto.AdminAccountItem{}, nil
	}
	rows, err := s.extras.Accounts.ListForAdmin(ctx, userID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.AdminAccountItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toAdminAccountItem(row))
	}
	return items, nil
}

func (s *adminUserService) Positions(ctx context.Context, actor AdminActor, userID string) ([]dto.AdminPositionItem, error) {
	if _, err := s.store.FindRow(ctx, userID); err != nil {
		return nil, err
	}
	rows, err := s.extras.Positions.ListForAdmin(ctx, userID, 200)
	if err != nil {
		return nil, err
	}

	// Membuka riwayat posisi adalah baca sensitif: wajib tercatat agar
	// pertanyaan "apakah staf melihat data saya?" bisa dijawab (plans/07 §9).
	if err := s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.UserViewSensitive,
		TargetType: "user", TargetID: userID, RequestID: actor.RequestID,
		Reason: "membuka riwayat posisi pelanggan",
	}); err != nil {
		return nil, err
	}

	items := make([]dto.AdminPositionItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toAdminPositionItem(row))
	}
	return items, nil
}

func (s *adminUserService) Legal(ctx context.Context, actor AdminActor, userID string) ([]dto.AdminLegalAcceptance, error) {
	if s.extras.Legal == nil {
		return []dto.AdminLegalAcceptance{}, nil
	}
	rows, err := s.extras.Legal.ListAcceptance(ctx, userID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.AdminLegalAcceptance, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.AdminLegalAcceptance{
			Slug: row.Slug, Title: row.Title, Version: row.Version,
			AcceptedAt: dto.NewTimePair(row.AcceptedAt), IPHash: row.IPHash,
		})
	}
	return items, nil
}

func (s *adminUserService) Notes(ctx context.Context, actor AdminActor, userID string) ([]dto.AdminNoteItem, error) {
	if s.extras.Notes == nil {
		return []dto.AdminNoteItem{}, nil
	}
	rows, err := s.extras.Notes.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.AdminNoteItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.AdminNoteItem{
			ID: row.ID, AuthorID: row.AuthorID, AuthorName: row.AuthorName,
			Body: row.Body, CreatedAt: dto.NewTimePair(row.CreatedAt),
		})
	}
	return items, nil
}

func (s *adminUserService) AddNote(ctx context.Context, actor AdminActor, userID string, req dto.UserNoteRequest) error {
	body := strings.TrimSpace(req.Body)
	if len([]rune(body)) < 3 {
		return validation("catatan terlalu pendek")
	}
	if _, err := s.store.FindRow(ctx, userID); err != nil {
		return err
	}
	if err := s.extras.Notes.Add(ctx, userID, actor.ID, body); err != nil {
		return err
	}
	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.UserNoteAdd,
		TargetType: "user", TargetID: userID, RequestID: actor.RequestID,
		After: map[string]any{"note": body},
	})
}

func (s *adminUserService) Suspend(ctx context.Context, actor AdminActor, userID string, req dto.UserActionRequest) error {
	reason, category, err := destructiveReason(audit.UserSuspend, req)
	if err != nil {
		return err
	}
	row, err := s.store.FindRow(ctx, userID)
	if err != nil {
		return err
	}
	if guardSelf(actor.ID, userID, "menangguhkan") {
		return validation("tidak dapat menangguhkan akun sendiri")
	}
	if guardAdminTarget(*row) {
		return validation("akun admin tidak dapat ditangguhkan")
	}

	now := s.now()
	fields := map[string]any{
		"suspended_at":  nowOf(now),
		"internal_note": appendNote(row.InternalNote, "Suspend: "+reason),
	}
	if err := s.store.UpdateStatus(ctx, userID, model.UserStatusSuspended, fields); err != nil {
		return err
	}
	s.revoke(ctx, userID)

	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.UserSuspend,
		TargetType: "user", TargetID: userID, Reason: reason, RequestID: actor.RequestID,
		Before: map[string]any{"status": row.Status},
		After:  map[string]any{"status": model.UserStatusSuspended, "category": category},
	})
}

func (s *adminUserService) Unsuspend(ctx context.Context, actor AdminActor, userID string, req dto.UserActionRequest) error {
	row, err := s.store.FindRow(ctx, userID)
	if err != nil {
		return err
	}
	if row.Status != model.UserStatusSuspended {
		return validation("akun ini tidak sedang ditangguhkan")
	}

	if err := s.store.UpdateStatus(ctx, userID, model.UserStatusActive, map[string]any{"suspended_at": nil}); err != nil {
		return err
	}

	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.UserUnsuspend,
		TargetType: "user", TargetID: userID, RequestID: actor.RequestID,
		Before: map[string]any{"status": row.Status},
		After:  map[string]any{"status": model.UserStatusActive},
	})
}

func (s *adminUserService) Ban(ctx context.Context, actor AdminActor, userID string, req dto.UserActionRequest) error {
	reason, category, err := destructiveReason(audit.UserBan, req)
	if err != nil {
		return err
	}
	row, err := s.store.FindRow(ctx, userID)
	if err != nil {
		return err
	}
	if guardSelf(actor.ID, userID, "memblokir") {
		return validation("tidak dapat memblokir akun sendiri")
	}
	if err := confirmEmail(row.Email, req.ConfirmEmail); err != nil {
		return err
	}

	now := s.now()
	fields := map[string]any{
		"banned_at":    now,
		"ban_reason":   reason,
		"ban_category": category,
		"suspended_at": nil,
	}
	if err := s.store.UpdateStatus(ctx, userID, model.UserStatusBanned, fields); err != nil {
		return err
	}
	// Ban memutus seluruh sesi: token lama langsung gugur.
	s.revoke(ctx, userID)

	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.UserBan,
		TargetType: "user", TargetID: userID, Reason: reason, RequestID: actor.RequestID,
		Before: map[string]any{"status": row.Status},
		After:  map[string]any{"status": model.UserStatusBanned, "category": category},
	})
}

func (s *adminUserService) Unban(ctx context.Context, actor AdminActor, userID string, req dto.UserActionRequest) error {
	row, err := s.store.FindRow(ctx, userID)
	if err != nil {
		return err
	}
	if row.Status != model.UserStatusBanned {
		return validation("akun ini tidak sedang diblokir")
	}

	updates := map[string]any{"banned_at": nil, "ban_reason": "", "ban_category": ""}
	if err := s.store.UpdateStatus(ctx, userID, model.UserStatusActive, updates); err != nil {
		return err
	}

	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.UserUnban,
		TargetType: "user", TargetID: userID, RequestID: actor.RequestID,
		Before: map[string]any{"status": row.Status, "ban_reason": row.BanReason},
		After:  map[string]any{"status": model.UserStatusActive},
	})
}

func (s *adminUserService) ChangeTier(ctx context.Context, actor AdminActor, userID string, req dto.UserTierRequest) error {
	reason, _, err := destructiveReason(audit.UserTierChange, dto.UserActionRequest{Reason: req.Reason, Category: model.ReasonOther})
	if err != nil {
		return err
	}
	row, err := s.store.FindRow(ctx, userID)
	if err != nil {
		return err
	}
	// Menaikkan tier sendiri berarti memberi diri akses berbayar tanpa bayar.
	if actor.ID == userID {
		return validation("tidak dapat mengubah tier akun sendiri")
	}

	now := s.now()
	var startedAt, expiresAt *time.Time
	if req.Tier != model.TierFree {
		started := now
		startedAt = &started
		if req.Days > 0 {
			end := now.AddDate(0, 0, req.Days)
			expiresAt = &end
		}
	}
	if err := s.store.UpdateTier(ctx, userID, req.Tier, startedAt, expiresAt); err != nil {
		return err
	}

	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.UserTierChange,
		TargetType: "user", TargetID: userID, Reason: reason, RequestID: actor.RequestID,
		Before: map[string]any{"tier": row.Tier},
		After:  map[string]any{"tier": req.Tier, "days": req.Days},
	})
}

func (s *adminUserService) Delete(ctx context.Context, actor AdminActor, userID string, req dto.UserActionRequest) error {
	reason, category, err := destructiveReason(audit.UserDelete, req)
	if err != nil {
		return err
	}
	if !req.Acknowledge {
		return validation("centang pernyataan bahwa penghapusan tidak bisa dibatalkan")
	}
	row, err := s.store.FindRow(ctx, userID)
	if err != nil {
		return err
	}
	if guardSelf(actor.ID, userID, "menghapus") {
		return validation("tidak dapat menghapus akun sendiri")
	}
	if err := confirmEmail(row.Email, req.ConfirmEmail); err != nil {
		return err
	}
	// Akun admin terakhir tidak boleh dihapus: platform akan terkunci.
	if row.Role == "admin" {
		active, err := s.store.CountActiveAdmins(ctx)
		if err != nil {
			return err
		}
		if active <= 1 {
			return validation("tidak dapat menghapus admin aktif terakhir")
		}
	}

	// Akun broker beserta kredensial dihapus lebih dulu, lalu baris user
	// dianonimkan. Posisi dan invoice tetap ada untuk kewajiban pajak.
	if s.extras.Accounts != nil {
		if err := s.extras.Accounts.DeleteAllForUser(ctx, userID); err != nil {
			return err
		}
	}
	s.revoke(ctx, userID)
	if err := s.store.Anonymize(ctx, userID); err != nil {
		return err
	}

	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.UserDelete,
		TargetType: "user", TargetID: userID, Reason: reason, RequestID: actor.RequestID,
		Before: map[string]any{"email": row.Email, "tier": row.Tier, "status": row.Status},
		After:  map[string]any{"status": model.UserStatusDeleted, "category": category},
	})
}

func (s *adminUserService) FreezeAccount(ctx context.Context, actor AdminActor, accountID string, req dto.AccountFreezeRequest) error {
	reason, _, err := destructiveReason(audit.AccountFreeze, dto.UserActionRequest{Reason: req.Reason, Category: model.ReasonOther})
	if err != nil {
		return err
	}
	account, err := s.extras.Accounts.FindForAdmin(ctx, accountID)
	if err != nil {
		return err
	}
	now := s.now()
	if err := s.extras.Accounts.SetFrozen(ctx, accountID, true, &now, reason); err != nil {
		return err
	}
	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.AccountFreeze,
		TargetType: "account", TargetID: accountID, Reason: reason, RequestID: actor.RequestID,
		Before: map[string]any{"is_frozen": account.IsFrozen, "user_id": account.UserID},
		After:  map[string]any{"is_frozen": true},
	})
}

func (s *adminUserService) UnfreezeAccount(ctx context.Context, actor AdminActor, accountID string, req dto.AccountFreezeRequest) error {
	account, err := s.extras.Accounts.FindForAdmin(ctx, accountID)
	if err != nil {
		return err
	}
	if !account.IsFrozen {
		return validation("akun ini tidak sedang dibekukan")
	}
	if err := s.extras.Accounts.SetFrozen(ctx, accountID, false, nil, ""); err != nil {
		return err
	}
	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.AccountUnfreeze,
		TargetType: "account", TargetID: accountID, RequestID: actor.RequestID,
		Before: map[string]any{"is_frozen": true, "reason": account.FrozenReason},
		After:  map[string]any{"is_frozen": false},
	})
}

func (s *adminUserService) revoke(ctx context.Context, userID string) {
	if s.sessions != nil {
		_ = s.sessions.RevokeSessions(ctx, userID)
	}
}

// guardSelf menolak aksi terhadap akun sendiri.
func guardSelf(actorID, targetID, action string) bool {
	return actorID != "" && actorID == targetID
}

// guardAdminTarget menolak suspend pada akun admin: penurunan hak akses admin
// dilakukan lewat role, bukan lewat penangguhan.
func guardAdminTarget(row repository.UserRow) bool {
	return row.Role == "admin"
}

// confirmEmail memastikan admin mengetik ulang email target dengan benar.
func confirmEmail(actual, typed string) error {
	if strings.TrimSpace(typed) == "" {
		return validation("ketik ulang email akun untuk melanjutkan")
	}
	if !strings.EqualFold(strings.TrimSpace(actual), strings.TrimSpace(typed)) {
		return validation("email yang diketik tidak cocok")
	}
	return nil
}

func appendNote(existing, line string) string {
	existing = strings.TrimSpace(existing)
	if existing == "" {
		return line
	}
	return existing + "\n" + line
}

func nowOf(t time.Time) *time.Time {
	return &t
}

var _ = errors.Join
var _ = apperror.ErrForbidden
var _ = authz.UserView
