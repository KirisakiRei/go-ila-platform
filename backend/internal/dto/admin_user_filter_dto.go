package dto

// AdminUserFilter adalah masukan daftar pelanggan.
type AdminUserFilter struct {
	Page        int
	PerPage     int
	Query       string
	Sort        string
	Order       string
	From        string
	To          string
	Statuses    string
	Tiers       string
	HasAccounts string
	HasTrial    bool
	OnlyAdmins  bool
}
