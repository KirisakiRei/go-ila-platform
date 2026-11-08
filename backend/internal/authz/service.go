package authz

import (
	"context"
	"sync"
	"time"
)

// CacheTTL adalah umur cache permission. Perubahan kewenangan berlaku paling
// lambat selama ini tanpa menuntut login ulang (plans/04 §4.3).
const CacheTTL = 30 * time.Second

// Loader membaca permission efektif milik sebuah akun.
// Implementasi aslinya membaca tabel role_permission lewat user.role_id.
type Loader interface {
	PermissionsOf(ctx context.Context, userID string) ([]string, error)
	// RoleCodeOf mengembalikan kode role RBAC akun; kosong bila tanpa role.
	RoleCodeOf(ctx context.Context, userID string) (string, error)
}

type cacheEntry struct {
	keys    map[string]struct{}
	role    string
	expires time.Time
}

// Service menyimpan hasil pembacaan permission agar tiap request tidak
// menanyakan database. Cache sengaja pendek: kewenangan harus cepat berubah.
type Service struct {
	loader Loader
	ttl    time.Duration
	now    func() time.Time

	mu      sync.RWMutex
	entries map[string]cacheEntry
}

// NewService membuat pemuat permission dengan cache.
func NewService(loader Loader) *Service {
	return &Service{
		loader:  loader,
		ttl:     CacheTTL,
		now:     func() time.Time { return time.Now().UTC() },
		entries: make(map[string]cacheEntry),
	}
}

// SetTTL mengganti umur cache; dipakai pengujian.
func (s *Service) SetTTL(ttl time.Duration) {
	s.ttl = ttl
}

// Has menandai apakah akun memiliki permission tertentu.
func (s *Service) Has(ctx context.Context, userID, permission string) (bool, error) {
	entry, err := s.load(ctx, userID)
	if err != nil {
		return false, err
	}
	// Kewenangan kritis tidak pernah lahir dari penugasan permission biasa:
	// akun wajib memegang role superadmin (plans/04 §6).
	if SuperadminOnly(permission) {
		return entry.role == RoleSuperadmin, nil
	}
	_, ok := entry.keys[permission]
	return ok, nil
}

// Permissions mengembalikan salinan daftar permission akun.
func (s *Service) Permissions(ctx context.Context, userID string) ([]string, error) {
	entry, err := s.load(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(entry.keys))
	for _, p := range Catalog {
		if _, ok := entry.keys[p.Key]; ok {
			out = append(out, p.Key)
		}
	}
	return out, nil
}

// RoleCode mengembalikan kode role RBAC akun.
func (s *Service) RoleCode(ctx context.Context, userID string) (string, error) {
	entry, err := s.load(ctx, userID)
	if err != nil {
		return "", err
	}
	return entry.role, nil
}

// Invalidate membuang cache satu akun, dipakai setelah kewenangannya berubah.
func (s *Service) Invalidate(userID string) {
	s.mu.Lock()
	delete(s.entries, userID)
	s.mu.Unlock()
}

// InvalidateAll membuang seluruh cache, dipakai setelah role diubah.
func (s *Service) InvalidateAll() {
	s.mu.Lock()
	s.entries = make(map[string]cacheEntry)
	s.mu.Unlock()
}

func (s *Service) load(ctx context.Context, userID string) (cacheEntry, error) {
	s.mu.RLock()
	entry, ok := s.entries[userID]
	s.mu.RUnlock()

	if ok && s.now().Before(entry.expires) {
		return entry, nil
	}

	keys, err := s.loader.PermissionsOf(ctx, userID)
	if err != nil {
		return cacheEntry{}, err
	}
	role, err := s.loader.RoleCodeOf(ctx, userID)
	if err != nil {
		return cacheEntry{}, err
	}

	entry = cacheEntry{
		keys:    make(map[string]struct{}, len(keys)),
		role:    role,
		expires: s.now().Add(s.ttl),
	}
	for _, key := range keys {
		entry.keys[key] = struct{}{}
	}

	s.mu.Lock()
	s.entries[userID] = entry
	s.mu.Unlock()

	return entry, nil
}
