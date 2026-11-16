package main

import (
	"context"
	"testing"

	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

// storeKejadian menyajikan jumlah peristiwa tetap untuk menguji pembentuk kunci.
type storeKejadian struct {
	repository.RunnerRepository
	jumlah int
	err    error
}

func (s *storeKejadian) HitungKejadianSinyal(context.Context, string, string) (int, error) {
	return s.jumlah, s.err
}

// TestKejadianSinyalMembedakanTahapBerulang memastikan nomor kejadian ikut
// terbaca, karena nomor itulah yang membuat tahap yang sama bisa dikirim lagi
// setelah setup mundur ke tahap tersebut.
func TestKejadianSinyalMembedakanTahapBerulang(t *testing.T) {
	p := publisherUji(nil)
	p.store = &storeKejadian{jumlah: 3}

	if got := p.kejadianSinyal(context.Background(), "u-1", "sig-1"); got != 3 {
		t.Fatalf("nomor kejadian = %d, ingin 3", got)
	}
}

// TestKejadianSinyalSaatGagalBaca memastikan kegagalan baca tidak menghilangkan
// notifikasi: nilainya jatuh ke nol, bukan menjadi galat.
func TestKejadianSinyalSaatGagalBaca(t *testing.T) {
	p := publisherUji(nil)
	p.store = &storeKejadian{err: apperror.ErrNotFound}

	if got := p.kejadianSinyal(context.Background(), "u-1", "sig-1"); got != 0 {
		t.Fatalf("nomor kejadian = %d, ingin 0 saat baca gagal", got)
	}
}
