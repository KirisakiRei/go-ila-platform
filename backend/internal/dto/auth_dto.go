package dto

// RegisterRequest adalah payload pendaftaran akun.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=3,max=100"`
	Email    string `json:"email" binding:"required,email,max=150"`
	Password string `json:"password" binding:"required,min=10,max=72"`
}

// LoginRequest adalah payload masuk akun.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UserResponse adalah representasi akun yang aman dikirim ke klien.
type UserResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Tier      string `json:"tier"`
	CreatedAt string `json:"created_at"`
}

// LoginResponse membawa data akun; token sesi dikirim lewat cookie HttpOnly
// dan sengaja tidak pernah masuk badan balasan.
type LoginResponse struct {
	Token     string       `json:"token,omitempty"`
	ExpiresAt string       `json:"expires_at"`
	User      UserResponse `json:"user"`
}
