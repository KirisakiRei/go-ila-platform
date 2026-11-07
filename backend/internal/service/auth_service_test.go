package service

import (
	"context"
	"testing"
	"time"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/dto"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/model"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/apperror"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/utils"
)

type fakeUserRepo struct {
	users map[string]*model.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[string]*model.User{}}
}

func (f *fakeUserRepo) Create(_ context.Context, user *model.User) error {
	if _, exists := f.users[user.Email]; exists {
		return apperror.ErrAlreadyExists
	}
	f.users[user.Email] = user
	return nil
}

func (f *fakeUserRepo) FindByID(_ context.Context, id string) (*model.User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, apperror.ErrNotFound
}

func (f *fakeUserRepo) FindByEmail(_ context.Context, email string) (*model.User, error) {
	if u, ok := f.users[email]; ok {
		return u, nil
	}
	return nil, apperror.ErrNotFound
}

func (f *fakeUserRepo) FindByTelegramChatID(_ context.Context, _ string) (*model.User, error) {
	return nil, apperror.ErrNotFound
}

func (f *fakeUserRepo) UpdateLastLogin(_ context.Context, _ string, _ time.Time) error {
	return nil
}

func (f *fakeUserRepo) Update(_ context.Context, user *model.User) error {
	f.users[user.Email] = user
	return nil
}

func (f *fakeUserRepo) CountByTier(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

type fakeAuditRepo struct {
	events []string
}

func (f *fakeAuditRepo) Record(_ context.Context, entry *model.AuditLog) error {
	f.events = append(f.events, entry.Event)
	return nil
}

type fakeTokenIssuer struct{}

func (fakeTokenIssuer) Issue(userID, role string) (string, time.Time, error) {
	return "token-" + userID, time.Now().Add(time.Hour), nil
}

func (fakeTokenIssuer) Verify(_ string) (utils.TokenClaims, error) {
	return utils.TokenClaims{}, nil
}

func newTestAuth() (AuthService, *fakeUserRepo, *fakeAuditRepo) {
	repo := newFakeUserRepo()
	audit := &fakeAuditRepo{}
	return NewAuthService(repo, audit, fakeTokenIssuer{}), repo, audit
}

func TestRegisterCreatesFreeUser(t *testing.T) {
	svc, repo, audit := newTestAuth()

	got, err := svc.Register(context.Background(), dto.RegisterRequest{
		Name:     "Budi",
		Email:    "  Budi@Example.COM ",
		Password: "rahasia-panjang-123",
	})
	if err != nil {
		t.Fatalf("register gagal: %v", err)
	}

	if got.Email != "budi@example.com" {
		t.Errorf("email harus dinormalisasi, dapat %q", got.Email)
	}
	if got.Tier != "free" {
		t.Errorf("akun baru harus tier free, dapat %q", got.Tier)
	}
	if _, ok := repo.users["budi@example.com"]; !ok {
		t.Fatal("user tidak tersimpan")
	}
	if len(audit.events) != 1 || audit.events[0] != "auth.register" {
		t.Errorf("audit register tidak tercatat: %v", audit.events)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	svc, _, _ := newTestAuth()
	req := dto.RegisterRequest{Name: "Budi", Email: "budi@example.com", Password: "rahasia-panjang-123"}

	if _, err := svc.Register(context.Background(), req); err != nil {
		t.Fatalf("register pertama gagal: %v", err)
	}

	if _, err := svc.Register(context.Background(), req); err != apperror.ErrAlreadyExists {
		t.Errorf("register duplikat harus ErrAlreadyExists, dapat %v", err)
	}
}

func TestLoginWithWrongPasswordIsUnauthorized(t *testing.T) {
	svc, _, _ := newTestAuth()
	ctx := context.Background()

	_, _ = svc.Register(ctx, dto.RegisterRequest{Name: "Budi", Email: "budi@example.com", Password: "rahasia-panjang-123"})

	_, err := svc.Login(ctx, dto.LoginRequest{Email: "budi@example.com", Password: "salah-sekali"})
	if err != apperror.ErrUnauthorized {
		t.Errorf("password salah harus ErrUnauthorized, dapat %v", err)
	}
}

func TestLoginUnknownEmailDoesNotLeakExistence(t *testing.T) {
	svc, _, _ := newTestAuth()

	_, err := svc.Login(context.Background(), dto.LoginRequest{Email: "tidak-ada@example.com", Password: "apa-saja"})
	if err != apperror.ErrUnauthorized {
		t.Errorf("email tak dikenal harus ErrUnauthorized (bukan NOT_FOUND), dapat %v", err)
	}
}

func TestLoginSucceedsAndRecordsAudit(t *testing.T) {
	svc, _, audit := newTestAuth()
	ctx := context.Background()

	_, _ = svc.Register(ctx, dto.RegisterRequest{Name: "Budi", Email: "budi@example.com", Password: "rahasia-panjang-123"})
	audit.events = nil

	got, err := svc.Login(ctx, dto.LoginRequest{Email: "budi@example.com", Password: "rahasia-panjang-123"})
	if err != nil {
		t.Fatalf("login gagal: %v", err)
	}
	if got.Token == "" {
		t.Error("token tidak boleh kosong")
	}
	if len(audit.events) != 1 || audit.events[0] != "auth.login" {
		t.Errorf("audit login tidak tercatat: %v", audit.events)
	}
}
