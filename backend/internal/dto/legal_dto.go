package dto

// LegalDocumentResponse adalah satu versi dokumen legal.
type LegalDocumentResponse struct {
	ID      string `json:"id,omitempty"`
	Slug    string `json:"slug"`
	Version int    `json:"version"`
	Title   string `json:"title"`
	BodyMD  string `json:"body_md,omitempty"`
	// RequireReaccept menandai perubahan yang wajib disetujui ulang.
	RequireReaccept bool `json:"require_reaccept"`
	Published       bool `json:"published"`
	// EffectiveAt adalah waktu berlaku; Effective membawa WIB dan NY.
	EffectiveAt string   `json:"effective_at"`
	Effective   TimePair `json:"effective_pair"`
	PublishedAt string   `json:"published_at,omitempty"`
}

// LegalStatusItem adalah keadaan persetujuan akun atas satu dokumen.
type LegalStatusItem struct {
	Slug            string   `json:"slug"`
	Title           string   `json:"title"`
	CurrentVersion  int      `json:"current_version"`
	AcceptedVersion int      `json:"accepted_version"`
	Accepted        bool     `json:"accepted"`
	RequireReaccept bool     `json:"require_reaccept"`
	EffectiveAt     string   `json:"effective_at"`
	Effective       TimePair `json:"effective_pair"`
}

// LegalPendingItem adalah dokumen wajib yang belum disetujui.
type LegalPendingItem struct {
	Slug           string `json:"slug"`
	Title          string `json:"title"`
	CurrentVersion int    `json:"current_version"`
	Required       bool   `json:"required"`
}

// LegalStatusResponse merangkum seluruh keadaan persetujuan sebuah akun.
type LegalStatusResponse struct {
	Documents []LegalStatusItem  `json:"documents"`
	Pending   []LegalPendingItem `json:"pending"`
	// Blocked berarti eksekusi dan pembayaran ditahan sampai user menyetujui.
	Blocked bool `json:"blocked"`
}

// AcceptLegalRequest adalah payload persetujuan satu dokumen.
type AcceptLegalRequest struct {
	// Version wajib versi yang sedang berlaku; 0 berarti versi terbaru.
	Version int `json:"version"`
}

// LegalDocumentRequest adalah payload pembuatan/perubahan draf oleh admin.
type LegalDocumentRequest struct {
	Slug            string `json:"slug" binding:"required,max=40"`
	Title           string `json:"title" binding:"omitempty,max=150"`
	BodyMD          string `json:"body_md" binding:"omitempty,max=200000"`
	RequireReaccept bool   `json:"require_reaccept"`
	EffectiveAt     string `json:"effective_at" binding:"omitempty,max=40"`
}

// PublishLegalRequest membawa alasan penerbitan versi baru.
type PublishLegalRequest struct {
	Reason string `json:"reason" binding:"omitempty,max=255"`
}

// ActivateTwoFactorRequest membawa kode dari aplikasi authenticator.
type ActivateTwoFactorRequest struct {
	Code string `json:"code" binding:"required,len=6,numeric"`
}

// CodeOnlyRequest dipakai penonaktifan dan pembuatan ulang kode pemulihan.
// Nilainya boleh kode 6 digit atau satu kode pemulihan.
type CodeOnlyRequest struct {
	Code string `json:"code" binding:"required,max=40"`
}

// VerifyTwoFactorRequest adalah payload verifikasi kode saat masuk.
// Code boleh kode 6 digit atau satu kode pemulihan.
type VerifyTwoFactorRequest struct {
	ChallengeID string `json:"challenge_id" binding:"required,max=64"`
	Code        string `json:"code" binding:"required,max=40"`
}
