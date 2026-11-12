package service

import (
	"strings"

	"ila/backend/internal/model"
)

// BugItem adalah satu baris riwayat laporan milik pengguna.
type BugItem struct {
	ID        string `json:"id"`
	Category  string `json:"category"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	AdminNote string `json:"admin_note"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// BugAttachmentItem adalah satu lampiran untuk ditampilkan.
type BugAttachmentItem struct {
	ID        string `json:"id"`
	FileName  string `json:"file_name"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
}

// BugDetail adalah laporan lengkap untuk admin dan pemiliknya.
type BugDetail struct {
	BugItem
	Body        string              `json:"body"`
	PageURL     string              `json:"page_url"`
	UserAgent   string              `json:"user_agent"`
	UserID      string              `json:"user_id"`
	Attachments []BugAttachmentItem `json:"attachments"`
}

func toBugItem(row model.BugReport) BugItem {
	return BugItem{
		ID:        row.ID,
		Category:  row.Category,
		Title:     row.Title,
		Status:    row.Status,
		AdminNote: row.AdminNote,
		CreatedAt: row.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: row.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toBugDetail(row model.BugReport, atts []model.BugReportAttachment) *BugDetail {
	out := &BugDetail{
		BugItem:     toBugItem(row),
		Body:        row.Body,
		PageURL:     row.PageURL,
		UserAgent:   row.UserAgent,
		UserID:      row.UserID,
		Attachments: make([]BugAttachmentItem, 0, len(atts)),
	}
	for _, att := range atts {
		out.Attachments = append(out.Attachments, BugAttachmentItem{
			ID: att.ID, FileName: att.FileName, MimeType: att.MimeType, SizeBytes: att.SizeBytes,
		})
	}
	return out
}

// KategoriLaporan mengembalikan daftar kategori untuk formulir.
// Dipakai frontend lewat endpoint, agar menambah kategori tidak menuntut rilis UI.
func KategoriLaporan() []string {
	out := make([]string, 0, len(model.BugKategori))
	for _, k := range model.BugKategori {
		out = append(out, strings.TrimSpace(k))
	}
	return out
}
