package dto

// RegisterRequest adalah payload pendaftaran akun.
//
// Username dan Phone tidak wajib agar klien lama tetap berjalan; bila
// dikirim, keduanya divalidasi keunikan dan formatnya (plans/09 §3).
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=3,max=100"`
	Username string `json:"username" binding:"omitempty,min=3,max=32,alphanum"`
	Email    string `json:"email" binding:"required,email,max=150"`
	Phone    string `json:"phone" binding:"omitempty,max=24"`
	Password string `json:"password" binding:"required,min=10,max=72"`
	// AcceptTerms menandai centang Syarat & Ketentuan + Kebijakan Privasi.
	// Registrasi tanpa centang ditolak (plans/09 §3).
	AcceptTerms bool `json:"accept_terms"`
	// InviteCode diisi dari tautan undangan. Wajib bila platform sedang
	// invite only; diabaikan pada pendaftaran mandiri biasa.
	InviteCode string `json:"invite_code,omitempty"`
}

// LoginRequest menerima email ATAU username pada kolom yang sama.
//
// Satu kolom dipakai supaya pengguna tidak perlu memilih jenis identitas,
// dan pesan galatnya tetap seragam sehingga tidak membocorkan akun mana
// yang terdaftar.
type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required,min=3,max=150"`
	Password   string `json:"password" binding:"required"`
}

// ChangePasswordRequest meminta verifikasi password lama sebelum berganti.
// Ganti password mencabut seluruh sesi lain (plans/23 §6).
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=10,max=72"`
}

// UserResponse adalah representasi akun yang aman dikirim ke klien.
type UserResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
	Tier     string `json:"tier"`
	// IdleWarning memutuskan apakah modal "masih di sini?" ditampilkan.
	// Batas diam di server tidak terpengaruh nilai ini (plans/23 §5).
	IdleWarning bool   `json:"idle_warning"`
	CreatedAt   string `json:"created_at"`
}

// LoginResponse membawa data akun; token sesi dikirim lewat cookie HttpOnly
// dan sengaja tidak pernah masuk badan balasan.
type LoginResponse struct {
	// Step bernilai "complete" (sesi terbit) atau "two_factor_required".
	Step        string       `json:"step,omitempty"`
	ChallengeID string       `json:"challenge_id,omitempty"`
	ExpiresIn   int          `json:"expires_in_seconds,omitempty"`
	Token       string       `json:"token,omitempty"`
	ExpiresAt   string       `json:"expires_at"`
	User        UserResponse `json:"user"`
	// SessionID menunjuk baris sesi yang baru dibuat.
	SessionID string `json:"session_id,omitempty"`
}

// SessionResponse adalah satu perangkat yang sedang masuk.
type SessionResponse struct {
	ID          string   `json:"id"`
	DeviceLabel string   `json:"device_label"`
	UserAgent   string   `json:"user_agent"`
	LastSeenAt  string   `json:"last_seen_at"`
	LastSeen    TimePair `json:"last_seen_pair"`
	CreatedAt   string   `json:"created_at"`
	Created     TimePair `json:"created_pair"`
	ExpiresAt   string   `json:"expires_at"`
	Expires     TimePair `json:"expires_pair"`
	// Current menandai sesi yang sedang dipakai perangkat ini.
	Current bool `json:"current"`
}
