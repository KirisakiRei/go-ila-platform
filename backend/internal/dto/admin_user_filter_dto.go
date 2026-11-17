package dto

// AdminUserFilter adalah masukan daftar pelanggan.
type AdminUserFilter struct {
	Page        int    `form:"page"`
	PerPage     int    `form:"per_page"`
	Query       string `form:"q"`
	Sort        string `form:"sort"`
	Order       string `form:"order"`
	From        string `form:"from"`
	To          string `form:"to"`
	Statuses    string `form:"status"`
	Tiers       string `form:"tier"`
	HasAccounts string `form:"accounts"`
	HasTrial    bool   `form:"-"`
	OnlyAdmins  bool   `form:"-"`
	// RoleID menyaring pelanggan menurut role RBAC tertentu.
	RoleID string `form:"role_id"`
}
