package service

import (
	"context"
	"strconv"
	"time"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/export"
	"ila/backend/internal/pagination"
	"ila/backend/internal/timezone"
)

// ExportUsers menyusun CSV daftar pelanggan sesuai filter yang sedang aktif.
// Setiap ekspor tercatat di audit beserta jumlah barisnya (plans/24 §6).
func (s *adminUserService) ExportUsers(ctx context.Context, actor AdminActor, filter dto.AdminUserFilter) ([]byte, string, error) {
	_, storeFilter, err := buildUserList(filter)
	if err != nil {
		return nil, "", err
	}

	builder := export.New(pagination.ExportMaxRows)
	if err := builder.Header(
		"id", "nama", "email", "tier", "status", "telegram",
		"jumlah_akun", "terdaftar_wib", "terdaftar_ny", "login_terakhir_wib",
	); err != nil {
		return nil, "", err
	}

	// Diambil bertahap agar memori tetap kecil walau pelanggan bertambah.
	for page := 1; ; page++ {
		batch, err := s.store.List(ctx, storeFilter, pagination.Params{
			Page: page, PerPage: pagination.MaxPerPage, Sort: "created_at", Order: "desc",
		})
		if err != nil {
			return nil, "", err
		}
		for _, row := range batch {
			wibCreated, nyCreated := export.TimeColumns(&row.CreatedAt)
			wibLogin, _ := export.TimeColumns(row.LastLoginAt)
			telegram := "tidak"
			if row.TelegramChatID != nil && *row.TelegramChatID != "" {
				telegram = "ya"
			}
			if err := builder.Row(
				row.ID, row.Name, row.Email, row.Tier, row.Status, telegram,
				strconv.FormatInt(row.AccountCount, 10),
				wibCreated, nyCreated, wibLogin,
			); err != nil {
				return nil, "", err
			}
		}
		if len(batch) < pagination.MaxPerPage {
			break
		}
	}

	if err := s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.UserExport,
		TargetType: "user", TargetID: "semua",
		Reason:    "ekspor daftar pelanggan",
		RequestID: actor.RequestID,
		After:     map[string]any{"rows": builder.Rows()},
	}); err != nil {
		return nil, "", err
	}

	name := "pengguna-" + time.Now().In(timezone.WIB).Format("20060102") + ".csv"
	return builder.Finished(), name, nil
}
