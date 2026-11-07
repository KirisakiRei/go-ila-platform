# AGENTS.md — Golang Project Specialist & Architecture Guide

Panduan arsitektur, konvensi, dan standar pengembangan untuk project backend berbasis **Go (Golang)** menggunakan **Gin Web Framework** dan **GORM**.

Dokumen ini berfungsi sebagai peta konteks dan acuan baku bagi AI agent serta engineer dalam membuat, mengelola, atau merefaktor project Go agar tetap konsisten, modular, type-safe, dan mudah dirawat.

---

## 1. Golden Rules & Prinsip Utama

1. **Clean Layered Architecture (Unidirectional Flow):**
   ```
   HTTP Request
        │
        ▼
   [Router / Middleware]
        │
        ▼
   [Handler]          ──► (HTTP transport, DTO binding & validation, status mapping)
        │
        ▼
   [Service]          ──► (Pure business logic, orchestration, transaction coordination)
        │
        ▼
   [Repository]       ──► (Data persistence, GORM queries, database boundary)
        │
        ▼
   [Database (GORM)]
   ```
   - Aliran data **hanya satu arah** ke bawah.
   - **Handler** DILARANG memanggil Repository atau Database secara langsung.
   - **Service** DILARANG bergantung pada Gin context (`*gin.Context`) atau detail transport HTTP.
   - **Repository** DILARANG memuat logika bisnis atau manipulasi HTTP response.

2. **Context Propagation:**
   - Semua fungsi Handler, Service, dan Repository WAJIB menerima `ctx context.Context` sebagai argumen pertama.
   - Semua query GORM WAJIB menggunakan `.WithContext(ctx)`.

3. **Separation of Concerns (Model vs DTO):**
   - **Model (`internal/model`)**: Merepresentasikan struktur tabel database dan relasi GORM.
   - **DTO (`internal/dto`)**: Merepresentasikan request payload dan response schema ke klien.
   - DILARANG mengikat (`c.ShouldBindJSON`) request langsung ke struct Model database.

4. **Mandatory Pagination for List/Get-All Endpoints:**
   - DILARANG melakukan `SELECT *` / query list data tanpa pagination dan hard limit. Ini praktik bodoh yang rawan memory leak saat data membesar di production.
   - Semua endpoint listing WAJIB mendukung pagination via query params (`page`, `per_page` / `limit`), kalkulasi `offset = (page - 1) * limit`, dan metadata (`total_items`, `total_pages`, `current_page`, `per_page`).
   - Default page: `1`, default limit: `10` atau `20`, max limit aman: `100`.

5. **Explicit & Safe Error Handling:**
   - DILARANG menelan error (`_ = fn()`) tanpa alasan eksplisit dan komentar logis.
   - DILARANG menggunakan `panic()` untuk alur kontrol aplikasi (hanya izinkan `panic` saat inisialisasi fatal di `main.go`).
   - Gunakan package error terpusat (`pkg/apperror`) untuk memetakan domain error ke HTTP status code yang sesuai.

6. **Interface Segregation:**
   - Definisikan interface pada tingkat konsumen (service mendefinisikan interface repository yang dibutuhkannya).
   - Memudahkan unit testing dengan mocking.

---

## 2. Standard Tech Stack

| Komponen | Library / Tool | Versi / Catatan |
|---|---|---|
| **Language** | Go | >= 1.22 |
| **HTTP Framework** | `github.com/gin-gonic/gin` | v1.10+ |
| **ORM** | `gorm.io/gorm` | PostgreSQL / MySQL driver |
| **Validation** | `github.com/go-playground/validator/v10` | Built-in via Gin binding |
| **Authentication** | `github.com/golang-jwt/jwt/v5` | Token-based auth |
| **Password Hash** | `golang.org/x/crypto/bcrypt` | Bcrypt hashing |
| **UUID** | `github.com/google/uuid` | UUID v4 / v7 |
| **Config Loader** | `github.com/joho/godotenv` atau `github.com/spf13/viper` | Environment-based config |
| **Logging** | `log/slog` (Standard Library) | Structured JSON logging |

---

## 3. Struktur Direktori Standar

```
.
├── cmd/
│   ├── server/
│   │   └── main.go              # Entrypoint aplikasi HTTP server
│   └── migrate/
│       └── main.go              # CLI database migration / seeder
├── internal/
│   ├── config/
│   │   ├── config.go            # Struct konfigurasi & validasi env vars
│   │   └── database.go          # Inisialisasi GORM connection & connection pool
│   ├── model/
│   │   ├── base.go              # Base struct (ID, CreatedAt, UpdatedAt, DeletedAt)
│   │   └── user.go              # Definisi entitas database + TableName()
│   ├── dto/
│   │   ├── auth_dto.go          # Request & Response structs + validation tags
│   │   └── user_dto.go
│   ├── repository/
│   │   ├── user_repository.go   # Implementasi query database (GORM)
│   │   └── interfaces.go        # Interface kontrak repository
│   ├── service/
│   │   ├── auth_service.go      # Implementasi business logic
│   │   ├── user_service.go
│   │   └── interfaces.go        # Interface kontrak service
│   ├── handler/
│   │   ├── auth_handler.go      # HTTP Controller (Gin Handlers)
│   │   └── user_handler.go
│   ├── middleware/
│   │   ├── auth.go              # JWT verification middleware
│   │   ├── recovery.go          # Panic recovery & error format
│   │   ├── logger.go            # Request logging via slog
│   │   ├── request_id.go        # X-Request-ID injector
│   │   ├── cors.go              # CORS configuration
│   │   └── rate_limiter.go      # Rate limiting middleware
│   └── router/
│       └── router.go            # Route registration & dependency wiring
├── pkg/
│   ├── apperror/
│   │   └── errors.go            # Custom domain errors & HTTP status mapper
│   ├── httpx/
│   │   └── response.go          # Standardized JSON response helper
│   └── utils/
│       ├── hash.go              # Helper hash & compare password
│       ├── jwt.go               # Helper sign & verify JWT
│       └── pagination.go        # Helper pagination math & query
├── migrations/                  # File migrasi SQL / script auto-migrate
├── .env.example
├── .gitignore
├── Makefile
├── go.mod
└── go.sum
```

---

## 4. Konvensi Kode & Implementasi Tiap Layer

### A. Model Layer (`internal/model`)
- Selalu tentukan nama tabel secara eksplisit melalui method `TableName()`.
- Definisikan GORM tags secara jelas (`gorm:"primaryKey;type:uuid"`, `gorm:"not null;uniqueIndex"`, dll).
- Gunakan field soft delete `gorm.DeletedAt` jika entitas membutuhkan riwayat.

```go
package model

import (
	"time"
	"gorm.io/gorm"
)

type User struct {
	ID        string         `gorm:"type:uuid;primaryKey" json:"id"`
	Email     string         `gorm:"type:varchar(191);uniqueIndex;not null" json:"email"`
	Password  string         `gorm:"type:varchar(255);not null" json:"-"`
	Name      string         `gorm:"type:varchar(100);not null" json:"name"`
	Role      string         `gorm:"type:varchar(50);default:'user';not null" json:"role"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}
```

---

### B. DTO, Pagination, & Validation (`internal/dto`)
- Pisahkan request DTO dari response DTO.
- Gunakan tag `binding:"..."` untuk validasi otomatis oleh Gin.
- Selalu sediakan struct standar untuk Pagination Request dan Pagination Meta Response.

```go
package dto

type PaginationQuery struct {
	Page    int    `form:"page,default=1" binding:"omitempty,min=1"`
	PerPage int    `form:"per_page,default=10" binding:"omitempty,min=1,max=100"`
	Search  string `form:"search" binding:"omitempty,max=100"`
	SortBy  string `form:"sort_by" binding:"omitempty"`
	Order   string `form:"order,default=desc" binding:"omitempty,oneof=asc desc"`
}

type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=3,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UserResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}
```

---

### C. Standard Response & Centralized Errors (`pkg/httpx` & `pkg/apperror`)

#### Standard JSON Response Format
Semua output API harus mengikuti format response yang seragam:

```json
{
  "success": true,
  "message": "User registered successfully",
  "data": { ... },
  "meta": null
}
```

Format response error:
```json
{
  "success": false,
  "message": "Invalid credentials",
  "error": "UNAUTHORIZED",
  "details": null
}
```

#### Implementasi `pkg/httpx/response.go`
```go
package httpx

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    any    `json:"meta,omitempty"`
	Error   string `json:"error,omitempty"`
	Details any    `json:"details,omitempty"`
}

func OK(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Created(c *gin.Context, message string, data any) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Error(c *gin.Context, httpStatus int, errCode string, message string, details any) {
	c.JSON(httpStatus, Response{
		Success: false,
		Message: message,
		Error:   errCode,
		Details: details,
	})
}
```

#### Implementasi `pkg/apperror/errors.go`
```go
package apperror

import (
	"errors"
	"net/http"
)

var (
	ErrNotFound       = errors.New("resource not found")
	ErrAlreadyExists  = errors.New("resource already exists")
	ErrUnauthorized   = errors.New("unauthorized access")
	ErrForbidden      = errors.New("forbidden resource")
	ErrBadRequest     = errors.New("invalid request payload")
	ErrInternalServer = errors.New("internal server error")
)

func MapHTTPStatus(err error) (int, string) {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, ErrAlreadyExists):
		return http.StatusConflict, "ALREADY_EXISTS"
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized, "UNAUTHORIZED"
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden, "FORBIDDEN"
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest, "BAD_REQUEST"
	default:
		return http.StatusInternalServerError, "INTERNAL_SERVER_ERROR"
	}
}
```

---

### D. Repository Layer (`internal/repository`)
- Menggunakan GORM dengan context `.WithContext(ctx)`.
- Hanya melakukan query database dan mapping ke model entity.

```go
package repository

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"myproject/internal/model"
	"myproject/pkg/apperror"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id string) (*model.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}
```

---

### E. Service Layer (`internal/service`)
- Menjalankan logika bisnis murni.
- Tidak boleh menerima `*gin.Context` (gunakan `context.Context`).
- Berkomunikasi antar repository via interface.

```go
package service

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"myproject/internal/dto"
	"myproject/internal/model"
	"myproject/internal/repository"
	"myproject/pkg/apperror"
	"myproject/pkg/utils"
)

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error)
}

type authService struct {
	userRepo  repository.UserRepository
	jwtSecret string
}

func NewAuthService(userRepo repository.UserRepository, jwtSecret string) AuthService {
	return &authService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, error) {
	existing, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, apperror.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, apperror.ErrAlreadyExists
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:       uuid.New().String(),
		Name:     req.Name,
		Email:    req.Email,
		Password: hashedPassword,
		Role:     "user",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return &dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, apperror.ErrUnauthorized
		}
		return nil, err
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return nil, apperror.ErrUnauthorized
	}

	token, err := utils.GenerateJWT(user.ID, user.Role, s.jwtSecret, 24)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token: token,
		User: dto.UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Role:      user.Role,
			CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}, nil
}
```

---

### F. Handler Layer (`internal/handler`)
- Menangani parsing request JSON / Query / Param.
- Mengoper `c.Request.Context()` ke service layer.
- Format response menggunakan `httpx`.

```go
package handler

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"myproject/internal/dto"
	"myproject/internal/service"
	"myproject/pkg/apperror"
	"myproject/pkg/httpx"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", err.Error())
		return
	}

	res, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		status, code := apperror.MapHTTPStatus(err)
		httpx.Error(c, status, code, err.Error(), nil)
		return
	}

	httpx.Created(c, "User registered successfully", res)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid login payload", err.Error())
		return
	}

	res, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		status, code := apperror.MapHTTPStatus(err)
		httpx.Error(c, status, code, err.Error(), nil)
		return
	}

	httpx.OK(c, "Login successful", res)
}
```

---

### G. Dependency Wiring & Router (`internal/router`)

```go
package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"myproject/internal/config"
	"myproject/internal/handler"
	"myproject/internal/middleware"
	"myproject/internal/repository"
	"myproject/internal/service"
)

func SetupRouter(cfg *config.Config, db *gorm.DB) *gin.Engine {
	r := gin.New()

	// Global Middlewares
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// Health Check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Repositories
	userRepo := repository.NewUserRepository(db)

	// Services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret)

	// Handlers
	authHandler := handler.NewAuthHandler(authService)

	// Routes
	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}
	}

	return r
}
```

---

## 5. Workflow Scaffolding Project Baru

Saat diminta membuat project Golang baru dari nol, ikuti urutan kerja ini:

1. **Inisialisasi Module & Environment:**
   ```bash
   mkdir <project-name> && cd <project-name>
   go mod init <module-path>
   ```

2. **Install Core Dependencies:**
   ```bash
   go get -u github.com/gin-gonic/gin
   go get -u gorm.io/gorm
   go get -u gorm.io/driver/postgres # atau mysql / sqlite
   go get -u github.com/golang-jwt/jwt/v5
   go get -u golang.org/x/crypto/bcrypt
   go get -u github.com/google/uuid
   go get -u github.com/joho/godotenv
   ```

3. **Buat Folder Structure:**
   Sesuai direktori di Bagian 3 (`cmd/`, `internal/`, `pkg/`, `migrations/`).

4. **Implementasikan Layer Dasar (Bawah ke Atas):**
   - Step 1: `pkg/apperror`, `pkg/httpx`, `pkg/utils`
   - Step 2: `internal/config` (load `.env` & validate)
   - Step 3: `internal/model` (entities)
   - Step 4: `internal/dto` (request/response validation)
   - Step 5: `internal/repository` (GORM query layer)
   - Step 6: `internal/service` (business rules)
   - Step 7: `internal/handler` (HTTP controller)
   - Step 8: `internal/middleware` (auth, recovery, logger)
   - Step 9: `internal/router` (wiring dependency)
   - Step 10: `cmd/server/main.go` & `cmd/migrate/main.go`

5. **Verifikasi Kompilasi & Test:**
   ```bash
   go vet ./...
   go build -o /dev/null ./...
   go test -v ./...
   ```

---

## 6. GORM Best Practices & Anti-Patterns

### ✅ DO:
1. **Always use `.WithContext(ctx)`** untuk setiap pemanggilan query database guna mendukung query cancellation & tracing.
2. **Periksa `errors.Is(err, gorm.ErrRecordNotFound)`** secara eksplisit saat query `First()` / `Take()`.
3. **Gunakan Database Connection Pool**:
   ```go
   sqlDB, _ := db.DB()
   sqlDB.SetMaxIdleConns(10)
   sqlDB.SetMaxOpenConns(100)
   sqlDB.SetConnMaxLifetime(time.Hour)
   ```
4. **Gunakan Transaction untuk Operasi Multi-Tabel**:
   ```go
   err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
       if err := tx.Create(&order).Error; err != nil {
           return err
       }
       if err := tx.Model(&inventory).Update("stock", gorm.Expr("stock - ?", qty)).Error; err != nil {
           return err
       }
       return nil
   })
   ```

### ❌ DON'T:
1. **DILARANG menggunakan GORM Hook (`BeforeCreate`, `AfterUpdate`)** untuk logika bisnis kompleks (seperti mengirim email atau hashing password). Taruh logika tersebut di Service Layer agar alur terlihat eksplisit.
2. **DILARANG mengabaikan error handling GORM**: Setiap query menghasilkan `.Error` yang wajib dicek.
3. **DILARANG melakukan query di dalam loop (N+1 Problem)**: Gunakan `Preload()` atau query batch `Where("id IN ?", ids)`.

---

## 7. Quality Checklist Sebelum Selesai (AI Agent)

Setiap kali menyelesaikan task atau fitur baru pada project Go:

- [ ] Pola arsitektur tetap bersih: `Router` → `Handler` → `Service` → `Repository` → `Model`.
- [ ] Tidak ada business logic di handler dan tidak ada framework Gin di service.
- [ ] DTO request dan response terpisah dari model GORM.
- [ ] Endpoint list / get-all data menerapkan pagination (`page`, `per_page`, `offset`, metadata pagination).
- [ ] Response API menggunakan format seragam `httpx.OK` / `httpx.Created` / `httpx.Error`.
- [ ] Error ditangani dengan jelas tanpa ada silent catch atau ignore error.
- [ ] Context `ctx` dialirkan dari handler sampai ke GORM query.
- [ ] Kode lulus kompilasi: `go vet ./...` dan `go build ./...`.
- [ ] File `.env` dan secrets tidak pernah dibocorkan atau di-commit.
