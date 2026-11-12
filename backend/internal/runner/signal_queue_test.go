package runner

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestSignalQueueMendahulukanSiap memastikan sinyal siap entry keluar lebih
// dulu walau status WAIT_* dimasukkan lebih awal. Inilah inti perbaikan:
// sebelumnya status WAIT_* memblokir sinyal siap entry di belakangnya.
func TestSignalQueueMendahulukanSiap(t *testing.T) {
	q := newSignalQueue(8)

	var urutan []string
	var mu sync.Mutex
	catat := func(nama string) func() {
		return func() {
			mu.Lock()
			urutan = append(urutan, nama)
			mu.Unlock()
		}
	}

	// WAIT masuk lebih dulu, siap entry menyusul - persis pola engine.
	if !q.push(PrioritasMenunggu, catat("wait-1")) {
		t.Fatal("push wait-1 ditolak")
	}
	if !q.push(PrioritasMenunggu, catat("wait-2")) {
		t.Fatal("push wait-2 ditolak")
	}
	if !q.push(PrioritasSiap, catat("siap-1")) {
		t.Fatal("push siap-1 ditolak")
	}

	ctx, cancel := context.WithCancel(context.Background())
	selesai := make(chan struct{})
	svc := &Service{}
	go func() {
		defer close(selesai)
		svc.jalankanAntrean(ctx, q, 1)
	}()

	// Tunggu ketiga pekerjaan selesai.
	batas := time.After(3 * time.Second)
	for {
		mu.Lock()
		n := len(urutan)
		mu.Unlock()
		if n == 3 {
			break
		}
		select {
		case <-batas:
			t.Fatalf("hanya %d pekerjaan selesai", n)
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	<-selesai

	if urutan[0] != "siap-1" {
		t.Fatalf("urutan = %v, ingin siap-1 didahulukan", urutan)
	}
}

// TestSignalQueueMenolakSaatPenuh memastikan antrean tidak menahan pemanggil.
// Stream engine harus tetap terbaca walau pemrosesan tertinggal.
func TestSignalQueueMenolakSaatPenuh(t *testing.T) {
	q := newSignalQueue(2)

	if !q.push(PrioritasSiap, func() {}) {
		t.Fatal("push pertama harus diterima")
	}
	if !q.push(PrioritasSiap, func() {}) {
		t.Fatal("push kedua harus diterima")
	}
	if q.push(PrioritasSiap, func() {}) {
		t.Fatal("push ketiga harus ditolak saat penuh")
	}

	siap, tunggu, dibuang := q.stats()
	if siap != 2 || tunggu != 0 || dibuang != 1 {
		t.Fatalf("stats = %d/%d/%d, ingin 2/0/1", siap, tunggu, dibuang)
	}
}

// TestSignalQueueBerhentiBersih memastikan pekerja tidak tertinggal ketika
// Runner berhenti: pop harus terbangun dan mengembalikan false.
func TestSignalQueueBerhentiBersih(t *testing.T) {
	q := newSignalQueue(4)

	selesai := make(chan struct{})
	go func() {
		defer close(selesai)
		if _, ok := q.pop(); ok {
			t.Error("pop harus false setelah antrean dihentikan")
		}
	}()

	// Beri kesempatan pop menunggu, lalu hentikan.
	time.Sleep(20 * time.Millisecond)
	q.hentikan()

	select {
	case <-selesai:
	case <-time.After(2 * time.Second):
		t.Fatal("pop tidak terbangun saat antrean dihentikan")
	}
}
