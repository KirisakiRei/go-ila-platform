package logingest

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"

	logkitv1 "ila/backend/gen/logkit/v1"
)

// aliranPalsu meniru sisi server gRPC tanpa jaringan.
type aliranPalsu struct {
	grpc.ServerStream
	ctx     context.Context
	masuk   []*logkitv1.LogEntry
	posisi  int
	ringkas *logkitv1.LogSummary
}

func (a *aliranPalsu) Context() context.Context { return a.ctx }

func (a *aliranPalsu) Recv() (*logkitv1.LogEntry, error) {
	if a.posisi >= len(a.masuk) {
		return nil, io.EOF
	}
	e := a.masuk[a.posisi]
	a.posisi++
	return e, nil
}

func (a *aliranPalsu) SendAndClose(r *logkitv1.LogSummary) error {
	a.ringkas = r
	return nil
}

func baruAliran(entries ...*logkitv1.LogEntry) *aliranPalsu {
	return &aliranPalsu{ctx: context.Background(), masuk: entries}
}

// Baris sah ditulis ke berkas harian layanan yang benar.
func TestTulisKeFolderHarian(t *testing.T) {
	dir := t.TempDir()
	s := Baru(dir, nil)
	t.Cleanup(func() { _ = s.Tutup() })

	baris := `{"t":"2026-11-08T10:15:03.000Z","lvl":"INFO","msg":"halo"}`
	st := baruAliran(&logkitv1.LogEntry{
		Baris:   []byte(baris),
		Layanan: "engine",
		Waktu:   "2026-11-08T10:15:03.000Z",
	})

	if err := s.KirimLog(st); err != nil {
		t.Fatalf("KirimLog: %v", err)
	}

	jalur := filepath.Join(dir, "2026", "11", "08", "engine-2026-11-08.log")
	f, err := os.Open(jalur)
	if err != nil {
		t.Fatalf("berkas tidak ada: %v", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		t.Fatal("berkas kosong")
	}
	var m map[string]any
	if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
		t.Fatalf("baris bukan JSON: %v", err)
	}
	if m["msg"] != "halo" {
		t.Fatalf("isi salah: %v", m)
	}
	if st.ringkas.GetDiterima() != 1 {
		t.Fatalf("diterima = %d, ingin 1", st.ringkas.GetDiterima())
	}
}

// Baris yang bukan JSON harus ditolak, bukan ditulis dan merusak berkas.
func TestTolakBarisBukanJSON(t *testing.T) {
	dir := t.TempDir()
	s := Baru(dir, nil)
	t.Cleanup(func() { _ = s.Tutup() })

	st := baruAliran(&logkitv1.LogEntry{
		Baris: []byte("ini bukan json"), Layanan: "engine",
		Waktu: "2026-11-08T10:15:03.000Z",
	})
	if err := s.KirimLog(st); err != nil {
		t.Fatalf("KirimLog: %v", err)
	}

	if st.ringkas.GetDitolak() != 1 {
		t.Fatalf("ditolak = %d, ingin 1", st.ringkas.GetDitolak())
	}
	jalur := filepath.Join(dir, "2026", "11", "08", "engine-2026-11-08.log")
	if _, err := os.Stat(jalur); !os.IsNotExist(err) {
		t.Fatal("berkas tidak boleh dibuat untuk baris tidak sah")
	}
}

// Baris kebesaran ditolak agar satu klien tidak menghabiskan disk.
func TestTolakBarisKebesaran(t *testing.T) {
	s := Baru(t.TempDir(), nil)

	besar := make([]byte, BatasBaris+1)
	for i := range besar {
		besar[i] = 'x'
	}
	st := baruAliran(&logkitv1.LogEntry{
		Baris: besar, Layanan: "engine", Waktu: "2026-11-08T10:15:03.000Z",
	})
	_ = s.KirimLog(st)

	if st.ringkas.GetDitolak() != 1 {
		t.Fatalf("ditolak = %d, ingin 1", st.ringkas.GetDitolak())
	}
}

// Layanan berbeda menulis ke berkas berbeda, sehingga engine dan runner tidak
// tercampur dalam satu berkas.
func TestLayananTerpisah(t *testing.T) {
	dir := t.TempDir()
	s := Baru(dir, nil)
	t.Cleanup(func() { _ = s.Tutup() })

	jam := "2026-11-08T10:00:00.000Z"
	st := baruAliran(
		&logkitv1.LogEntry{Baris: []byte(`{"a":1}`), Layanan: "engine", Waktu: jam},
		&logkitv1.LogEntry{Baris: []byte(`{"b":2}`), Layanan: "runner", Waktu: jam},
	)
	_ = s.KirimLog(st)

	for _, layanan := range []string{"engine", "runner"} {
		jalur := filepath.Join(dir, "2026", "11", "08", layanan+"-2026-11-08.log")
		if _, err := os.Stat(jalur); err != nil {
			t.Fatalf("berkas %s tidak ada: %v", layanan, err)
		}
	}
}

// Pastikan waktu yang tidak terbaca tidak menggagalkan penulisan.
func TestWaktuRusakTetapDitulis(t *testing.T) {
	dir := t.TempDir()
	s := Baru(dir, nil)
	t.Cleanup(func() { _ = s.Tutup() })

	st := baruAliran(&logkitv1.LogEntry{
		Baris: []byte(`{"x":1}`), Layanan: "engine", Waktu: "bukan-waktu",
	})
	if err := s.KirimLog(st); err != nil {
		t.Fatalf("KirimLog: %v", err)
	}
	if st.ringkas.GetDiterima() != 1 {
		t.Fatalf("diterima = %d, ingin 1", st.ringkas.GetDiterima())
	}
	_ = time.Now
}

// Baris yang sudah berakhiran newline tidak boleh ditambah newline lagi:
// kelebihan newline menghasilkan baris kosong di antara log.
func TestTidakAdaBarisKosong(t *testing.T) {
	dir := t.TempDir()
	s := Baru(dir, nil)
	t.Cleanup(func() { _ = s.Tutup() })

	jam := "2026-11-08T10:00:00.000Z"
	st := baruAliran(
		&logkitv1.LogEntry{Baris: []byte(`{"a":1}` + "\n"), Layanan: "engine", Waktu: jam},
		&logkitv1.LogEntry{Baris: []byte(`{"a":2}`), Layanan: "engine", Waktu: jam},
	)
	if err := s.KirimLog(st); err != nil {
		t.Fatalf("KirimLog: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "2026", "11", "08", "engine-2026-11-08.log"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(data), "\n"); got != 2 {
		t.Fatalf("jumlah newline = %d, ingin 2 (tanpa baris kosong)", got)
	}
}
