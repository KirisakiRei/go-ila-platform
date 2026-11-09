package dto

// RoleResponse adalah satu role beserta jumlah akun pemakainya.
type RoleResponse struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// IsSystem menandai role bawaan: tidak dapat diubah atau dihapus.
	IsSystem bool `json:"is_system"`
	// Editable dipakai antarmuka untuk mengunci tombol ubah.
	Editable    bool     `json:"editable"`
	Permissions []string `json:"permissions"`
	UserCount   int64    `json:"user_count"`
	CreatedAt   string   `json:"created_at"`
}

// RoleDetailResponse menambahkan katalog permission yang bisa dipilih.
type RoleDetailResponse struct {
	RoleResponse
	AvailablePermissions []PermissionResponse `json:"available_permissions"`
}

// PermissionResponse adalah satu kewenangan di katalog.
type PermissionResponse struct {
	Key   string `json:"key"`
	Group string `json:"group"`
	Label string `json:"label"`
	Risk  string `json:"risk"`
	// Grantable=false berarti pelaku tidak boleh memberikan izin ini.
	Grantable bool `json:"grantable"`
}

// PermissionGroup mengelompokkan permission untuk tampilan centang bergrup.
type PermissionGroup struct {
	Group       string               `json:"group"`
	Permissions []PermissionResponse `json:"permissions"`
}

// CreateRoleRequest adalah payload pembuatan role baru.
type CreateRoleRequest struct {
	Code        string   `json:"code" binding:"required,min=2,max=40"`
	Name        string   `json:"name" binding:"required,min=2,max=80"`
	Description string   `json:"description" binding:"omitempty,max=255"`
	Permissions []string `json:"permissions" binding:"required,min=1,max=200"`
}

// UpdateRoleRequest mengganti seluruh permission sebuah role.
type UpdateRoleRequest struct {
	Permissions []string `json:"permissions" binding:"required,min=1,max=200"`
}

// AssignRoleRequest menugaskan role RBAC ke sebuah akun.
// RoleID nil berarti mencabut role admin akun tersebut.
type AssignRoleRequest struct {
	RoleID *string `json:"role_id"`
}
