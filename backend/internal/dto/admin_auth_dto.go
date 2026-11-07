package dto

// AdminIdentity adalah identitas minimal akun admin.
type AdminIdentity struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// AdminRoleSummary adalah ringkasan role RBAC sesi admin.
type AdminRoleSummary struct {
	Code string `json:"code"`
	Name string `json:"name"`
	// IsSuperadmin menandai kewenangan yang tidak bisa didelegasikan.
	IsSuperadmin bool `json:"is_superadmin"`
}

// AdminLoginResponse membawa identitas, role, dan kewenangan sesi admin.
// Token tidak ikut dikirim: satu-satunya tempatnya adalah cookie HttpOnly.
type AdminLoginResponse struct {
	Token       string        `json:"token,omitempty"`
	SessionID   string        `json:"session_id,omitempty"`
	ExpiresAt   string        `json:"expires_at"`
	User        AdminIdentity `json:"user"`
	RoleCode    string        `json:"role_code"`
	Permissions []string      `json:"permissions"`
}

// AdminProfileResponse adalah balasan GET /admin/auth/me.
type AdminProfileResponse struct {
	User        AdminIdentity    `json:"user"`
	Role        AdminRoleSummary `json:"role"`
	Permissions []string         `json:"permissions"`
}
