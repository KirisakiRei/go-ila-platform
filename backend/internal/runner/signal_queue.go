package runner

import (
	"context"
	"sync"
)

// SignalPriority memisahkan sinyal yang butuh kecepatan dari yang hanya perlu
// tercatat.
type SignalPriority int

const (
	// PrioritasSiap adalah sinyal yang menawarkan entry sekarang
	// (READY_TO_ENTRY dan padanannya). Keterlambatan beberapa detik di sini
	// langsung berarti harga sudah bergerak dari level entry.
	PrioritasSiap SignalPriority = iota
	// PrioritasMenunggu adalah status antara (WAIT_*). Sinyal ini mengubah
	// tampilan radar, tetapi tidak pernah menghasilkan order.
	PrioritasMenunggu
)

// signalQueue adalah antrean berprioritas untuk pemrosesan sinyal.
//
// Latar belakang: callback stream engine dipanggil sinkron, sehingga satu
// sinyal lambat menahan seluruh sinyal di belakangnya. Data produksi
// menunjukkannya dengan jelas - sinyal rilis bersamaan diterima 3,6 detik
// sampai 20,7 detik setelah candle tutup, dan sinyal siap entry justru paling
// terakhir karena engine mengirim status WAIT_* lebih dulu.
//
// Antrean ini membuat stream tetap cepat membaca, sementara pemrosesan
// berjalan di latar dan selalu mendahulukan sinyal siap entry. Kapasitas
// dibatasi supaya lonjakan tidak menumbuhkan memori tanpa batas; bila penuh,
// sinyal dilepas (bukan diblokir) agar stream tidak pernah macet.
//
// Antrean memakai syarat dan notifikasi (sync.Cond), bukan putaran tidur,
// sehingga pekerja tidak membuang CPU saat antrean kosong.
type signalQueue struct {
	mu     sync.Mutex
	kond   *sync.Cond
	siap   []queuedSignal
	tunggu []queuedSignal
	// kapasitas membatasi jumlah sinyal menunggu per kelas.
	kapasitas int
	// dibuang mencatat berapa sinyal dilepas karena antrean penuh.
	dibuang int
	// tutup menandai antrean tidak lagi menerima pekerjaan.
	tutup bool
}

type queuedSignal struct {
	priority SignalPriority
	run      func()
}

// newSignalQueue membuat antrean dengan kapasitas per kelas.
func newSignalQueue(kapasitas int) *signalQueue {
	if kapasitas <= 0 {
		kapasitas = 256
	}
	q := &signalQueue{kapasitas: kapasitas}
	q.kond = sync.NewCond(&q.mu)
	return q
}

// push memasukkan pekerjaan ke antrean sesuai prioritasnya.
//
// Mengembalikan false bila antrean kelas itu penuh atau sudah ditutup.
// Pemanggil tidak boleh memblokir: stream engine harus tetap terbaca.
func (q *signalQueue) push(priority SignalPriority, run func()) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.tutup {
		return false
	}

	if priority == PrioritasSiap {
		if len(q.siap) >= q.kapasitas {
			q.dibuang++
			return false
		}
		q.siap = append(q.siap, queuedSignal{priority: priority, run: run})
	} else {
		if len(q.tunggu) >= q.kapasitas {
			q.dibuang++
			return false
		}
		q.tunggu = append(q.tunggu, queuedSignal{priority: priority, run: run})
	}

	q.kond.Signal()
	return true
}

// pop mengambil pekerjaan berikutnya, selalu mendahulukan sinyal siap entry.
// Mengembalikan false bila antrean kosong atau sudah ditutup.
func (q *signalQueue) pop() (queuedSignal, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for !q.tutup && len(q.siap) == 0 && len(q.tunggu) == 0 {
		q.kond.Wait()
	}
	if q.tutup {
		return queuedSignal{}, false
	}

	if len(q.siap) > 0 {
		item := q.siap[0]
		q.siap = q.siap[1:]
		return item, true
	}
	item := q.tunggu[0]
	q.tunggu = q.tunggu[1:]
	return item, true
}

// hentikan membangunkan semua pekerja dan menutup antrean.
func (q *signalQueue) hentikan() {
	q.mu.Lock()
	q.tutup = true
	q.mu.Unlock()
	q.kond.Broadcast()
}

// stats melaporkan isi antrean untuk log operasional.
func (q *signalQueue) stats() (siap, tunggu, dibuang int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.siap), len(q.tunggu), q.dibuang
}

// SignalWorkers adalah jumlah sinyal yang diproses bersamaan.
//
// Tiga pekerja dipilih agar satu akun dengan broker lambat tidak menahan
// sinyal simbol lain, tanpa membuka terlalu banyak koneksi broker sekaligus.
const SignalWorkers = 3

// RunSignalWorkers memproses antrean sinyal sampai ctx berakhir.
//
// Harus dijalankan sebelum stream engine dibuka: tanpa pekerja, sinyal yang
// masuk hanya menumpuk di antrean dan tidak pernah disimpan.
func (s *Service) RunSignalWorkers(ctx context.Context) {
	s.jalankanAntrean(ctx, s.antrean, SignalWorkers)
}

// SignalQueueStats melaporkan isi antrean untuk pemantauan.
func (s *Service) SignalQueueStats() (siap, tunggu, dibuang int) {
	return s.antrean.stats()
}

// jalankanAntrean memproses isi antrean sampai ctx berakhir.
func (s *Service) jalankanAntrean(ctx context.Context, q *signalQueue, pekerja int) {
	if pekerja <= 0 {
		pekerja = 1
	}

	var wg sync.WaitGroup
	for i := 0; i < pekerja; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				item, ok := q.pop()
				if !ok {
					return
				}
				item.run()
			}
		}()
	}

	// ctx berakhir: bangunkan pekerja yang sedang menunggu.
	<-ctx.Done()
	q.hentikan()
	wg.Wait()
}
