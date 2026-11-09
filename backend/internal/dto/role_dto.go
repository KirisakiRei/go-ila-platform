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

// CreateAdminRequest membuat akun pengelola dari panel RBAC.
//
// Yang wajib hanya username, kata sandi, dan role. Akun pengelola tidak
// melewati alur pelanggan (verifikasi email, trial, onboarding), karena akun
// itu tidak memakai fitur trading. Email tetap wajib diisi di basis data
// (kolomnya NOT NULL dan unik), jadi bila tidak diberikan sistem menyusun
// alamat internal dari username agar tidak ada verifikasi yang ditunggu.
type CreateAdminRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	// Name opsional; kosong berarti memakai username.
	Name string `json:"name" binding:"omitempty,max=100"`
	// Email opsional. Diisi hanya bila akun perlu surel nyata.
	Email    string `json:"email" binding:"omitempty,email,max=150"`
	Password string `json:"password" binding:"required,min=10,max=72"`
	// RoleID wajib: akun pengelola tanpa role tidak punya kewenangan apa pun
	// dan tidak bisa masuk lewat pintu admin.
	RoleID string `json:"role_id" binding:"required,uuid"`
}
