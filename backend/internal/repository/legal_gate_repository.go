package repository

import (
	"context"

	"gorm.io/gorm"
)

// NewLegalGate membuat pemeriksa persetujuan dokumen untuk Runner.
//
// Runner memakainya sebelum eksekusi: bila ada dokumen wajib yang belum
// disetujui pada versi terbitnya, entry baru ditolak. Posisi yang sudah
// berjalan tidak disentuh (plans/09).
func NewLegalGate(db *gorm.DB) func(ctx context.Context, userID string) bool {
	// Versi terbit terakhir per slug dihitung lebih dulu; persetujuan harus
	// pada versi itu, bukan versi lama.
	const query = `WITH latest AS (
			SELECT DISTINCT ON (slug) slug, version
			FROM legal_document
			WHERE published_at IS NOT NULL AND require_reaccept = true
			ORDER BY slug, version DESC
		)
		SELECT COUNT(*) FROM latest l
		WHERE NOT EXISTS (
			SELECT 1 FROM legal_acceptance a
			WHERE a.user_id = ? AND a.document_slug = l.slug AND a.version >= l.version
		)`

	return func(ctx context.Context, userID string) bool {
		var pending int64
		if err := db.WithContext(ctx).Raw(query, userID).Scan(&pending).Error; err != nil {
			// Gagal membaca bukan alasan memblokir semua pengguna. Jalur
			// trading tidak boleh mati hanya karena satu query gagal.
			return false
		}
		return pending > 0
	}
}
