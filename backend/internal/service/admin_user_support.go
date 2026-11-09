package service

import (
	"context"
	"strings"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/pagination"
	"ila/backend/internal/repository"
)

// userSortColumns adalah daftar putih kolom urut untuk daftar pelanggan.
var userSortColumns = map[string]string{
	"created_at":    "created_at",
	"last_login_at": "last_login_at",
	"name":          "name",
	"email":         "email",
	"tier":          "tier",
	"status":        "status",
}

// buildUserList memvalidasi filter lalu menyusun parameter repository.
func buildUserList(filter dto.AdminUserFilter) (pagination.Params, repository.UserFilter, error) {
	var zero pagination.Params
	var zeroFilter repository.UserFilter

	params, err := pagination.Build(pagination.Request{
		Page:        filter.Page,
		PerPage:     filter.PerPage,
		Query:       filter.Query,
		Sort:        filter.Sort,
		Order:       filter.Order,
		From:        filter.From,
		To:          filter.To,
		AllowedSort: userSortColumns,
		DefaultSort: "created_at",
	})
	if err != nil {
		return zero, zeroFilter, err
	}

	statuses, err := pagination.ParseStatusFilter(filter.Statuses, []string{
		model.UserStatusActive, model.UserStatusSuspended,
		model.UserStatusBanned, model.UserStatusDeleted,
	})
	if err != nil {
		return zero, zeroFilter, err
	}
	tiers, err := pagination.ParseStatusFilter(filter.Tiers, []string{
		model.TierFree, model.TierBasic, model.TierPlus,
	})
	if err != nil {
		return zero, zeroFilter, err
	}

	var hasAccounts *bool
	switch strings.TrimSpace(filter.HasAccounts) {
	case "":
	case "yes":
		value := true
		hasAccounts = &value
	case "no":
		value := false
		hasAccounts = &value
	default:
		return zero, zeroFilter, validation("filter akun broker harus yes atau no")
	}

	storeFilter := repository.UserFilter{
		Statuses:    statuses,
		Tiers:       tiers,
		HasAccounts: hasAccounts,
		HasTrial:    filter.HasTrial,
		OnlyAdmin:   filter.OnlyAdmins,
		From:        params.From,
		To:          params.To,
	}
	return params, storeFilter, nil
}

// AdminAccountStore adalah akses akun broker dari sisi admin, tanpa kredensial.
type AdminAccountStore interface {
	ListForAdmin(ctx context.Context, userID string) ([]repository.AdminAccountRow, error)
	FindForAdmin(ctx context.Context, accountID string) (*repository.AdminAccountRow, error)
	// BalancesForUser membaca telemetri saldo Runner per akun.
	BalancesForUser(ctx context.Context, userID string) (map[string]repository.AccountBalanceRow, error)
	SetFrozen(ctx context.Context, accountID string, frozen bool, at *time.Time, reason string) error
	// SetLabel hanya mengubah label tampilan akun.
	SetLabel(ctx context.Context, accountID, label string) error
	// DeleteAccount menghapus akun beserta kredensialnya.
	DeleteAccount(ctx context.Context, accountID string) error
	DeleteAllForUser(ctx context.Context, userID string) error
}

// AdminPositionStore membaca riwayat posisi pelanggan.
type AdminPositionStore interface {
	ListForAdmin(ctx context.Context, userID string, limit int) ([]model.Position, error)
}

// AdminLegalStore membaca bukti persetujuan dokumen.
type AdminLegalStore interface {
	ListAcceptance(ctx context.Context, userID string) ([]model.AdminLegalAcceptance, error)
}

// AdminNoteStore mengelola catatan internal.
type AdminNoteStore interface {
	List(ctx context.Context, userID string) ([]model.AdminNoteRow, error)
	Add(ctx context.Context, userID, authorID, body string) error
}

// toUserListItem memetakan baris repository ke DTO daftar.
func toUserListItem(row repository.UserRow) dto.UserListItem {
	item := dto.UserListItem{
		ID:             row.ID,
		Name:           row.Name,
		Email:          row.Email,
		Tier:           row.Tier,
		Status:         row.Status,
		Role:           row.Role,
		RoleCode:       row.RoleCode,
		TelegramLinked: row.TelegramChatID != nil && *row.TelegramChatID != "",
		AccountCount:   row.AccountCount,
		CreatedAt:      dto.NewTimePair(row.CreatedAt),
		LastLoginAt:    dto.NewTimePairPtr(row.LastLoginAt),
		TierExpiresAt:  dto.NewTimePairPtr(row.TierExpiresAt),
	}
	return item
}

// toAdminAccountItem memetakan akun broker tanpa menyentuh kredensial.
func toAdminAccountItem(row repository.AdminAccountRow) dto.AdminAccountItem {
	return dto.AdminAccountItem{
		ID:                row.ID,
		Label:             row.Label,
		BrokerType:        row.BrokerType,
		ExternalAccountID: row.ExternalAccountID,
		Mode:              row.Mode,
		RiskMode:          row.RiskMode,
		RiskPercent:       row.RiskPercent,
		IsActive:          row.IsActive,
		IsFrozen:          row.IsFrozen,
		IsDryRun:          row.IsDryRun,
		FrozenAt:          dto.NewTimePairPtr(row.FrozenAt),
		FrozenReason:      row.FrozenReason,
		LastVerifiedAt:    dto.NewTimePairPtr(row.LastVerifiedAt),
		LastError:         row.LastError,
	}
}

// toAdminPositionItem memetakan posisi ke bentuk dua zona waktu.
func toAdminPositionItem(row model.Position) dto.AdminPositionItem {
	return dto.AdminPositionItem{
		ID:              row.ID,
		Symbol:          row.Symbol,
		Direction:       row.Direction,
		StrategyVersion: row.StrategyVersion,
		Status:          row.Status,
		OpenPrice:       row.OpenPrice,
		ExitPrice:       row.ExitPrice,
		ProfitLoss:      row.ProfitLoss,
		OutcomeR:        row.OutcomeR,
		IsDryRun:        row.IsDryRun,
		OpenedAt:        dto.NewTimePair(row.OpenedAt),
		ClosedAt:        dto.NewTimePairPtr(row.ClosedAt),
	}
}
