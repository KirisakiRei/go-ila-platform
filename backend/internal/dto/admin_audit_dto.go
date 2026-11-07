package dto

// AdminAuditFilter adalah masukan daftar audit. Rentang tanggal wajib agar
// halaman ini tidak pernah memindai seluruh tabel (plans/24 §4.3).
type AdminAuditFilter struct {
	Page       int
	PerPage    int
	Cursor     string
	Query      string
	Sort       string
	Order      string
	From       string
	To         string
	ActorID    string
	Action     string
	TargetType string
	TargetID   string
}

// AuditEntry adalah satu baris audit yang dikirim ke panel admin.
type AuditEntry struct {
	ID          string   `json:"id"`
	ActorID     string   `json:"actor_id"`
	ActorType   string   `json:"actor_type"`
	ActorEmail  string   `json:"actor_email,omitempty"`
	Action      string   `json:"action"`
	TargetType  string   `json:"target_type"`
	TargetID    string   `json:"target_id"`
	Reason      string   `json:"reason,omitempty"`
	BeforeState string   `json:"before_state,omitempty"`
	AfterState  string   `json:"after_state,omitempty"`
	RequestID   string   `json:"request_id,omitempty"`
	CreatedAt   TimePair `json:"created_at"`
}

// AuditCatalogEntry adalah pilihan aksi untuk filter daftar audit.
type AuditCatalogEntry struct {
	Action    string `json:"action"`
	Group     string `json:"group"`
	Label     string `json:"label"`
	Sensitive bool   `json:"sensitive"`
}
