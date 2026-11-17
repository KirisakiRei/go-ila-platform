// Package logingest menerima aliran log dari Engine (VPS 1) dan menuliskannya
// ke folder harian di VPS platform.
//
// Tujuannya satu: saat menelusuri insiden, semua log bisa dibaca dari satu
// tempat tanpa membuka dua SSH.
package logingest

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	logkitv1 "ila/backend/gen/logkit/v1"
)

// BatasBaris adalah ukuran maksimum satu baris log yang diterima. Baris lebih
// besar ditolak agar satu klien tidak bisa menghabiskan disk.
const BatasBaris = 256 * 1024

// Server menulis log kiriman Engine ke berkas harian.
type Server struct {
	logkitv1.UnimplementedLogServiceServer

	dir    string
	log    *slog.Logger
	lokasi *time.Location

	mu     sync.Mutex
	nama   string
	berkas *os.File
}

// Baru membuat penerima. Dir kosong berarti log hanya dihitung, tidak ditulis.
func Baru(dir string, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{dir: dir, log: log, lokasi: time.UTC}
}

// KirimLog menulis setiap baris ke berkas layanan pada tanggal kejadian.
func (s *Server) KirimLog(stream logkitv1.LogService_KirimLogServer) error {
	var diterima, ditolak int64

	for {
		entri, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			s.log.Info("aliran log engine selesai",
				"evt", "log.terima.selesai", "diterima", diterima, "ditolak", ditolak)
			// Berkas ditutup agar bisa dirotasi/kompresi tanpa menahan handle.
			_ = s.Tutup()
			return stream.SendAndClose(&logkitv1.LogSummary{
				Diterima: diterima,
				Ditolak:  ditolak,
				Pesan:    "selesai",
			})
		}
		if err != nil {
			s.log.Warn("aliran log engine terputus",
				"evt", "log.terima.terputus", "error", err.Error())
			_ = s.Tutup()
			return err
		}

		if len(entri.GetBaris()) == 0 || len(entri.GetBaris()) > BatasBaris {
			ditolak++
			continue
		}
		// Baris harus JSON utuh agar berkas tetap bisa dibaca jq.
		if !json.Valid(entri.GetBaris()) {
			ditolak++
			continue
		}

		if err := s.tulis(entri); err != nil {
			s.log.Error("gagal menulis log engine",
				"evt", "log.terima.gagal", "error", err.Error())
			ditolak++
			continue
		}
		diterima++
	}
}

// tulis menambahkan satu baris ke berkas harian layanan terkait.
func (s *Server) tulis(entri *logkitv1.LogEntry) error {
	if s.dir == "" {
		return nil
	}

	waktu, err := time.Parse("2006-01-02T15:04:05.000Z", entri.GetWaktu())
	if err != nil {
		waktu = time.Now().UTC()
	}
	waktu = waktu.In(s.lokasi)

	layanan := strings.TrimSpace(entri.GetLayanan())
	if layanan == "" {
		layanan = "engine"
	}
	if !namaLayananSah(layanan) {
		return errors.New("nama layanan log tidak sah")
	}

	hari := waktu.Format("2006/01/02")
	nama := layanan + "-" + waktu.Format("2006-01-02") + ".log"
	dasar := filepath.Clean(s.dir)
	jalur := filepath.Join(dasar, filepath.FromSlash(hari), nama)
	if rel, relErr := filepath.Rel(dasar, jalur); relErr != nil || strings.HasPrefix(rel, "..") {
		return errors.New("jalur log di luar direktori dasar")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.berkas == nil || s.nama != jalur {
		if s.berkas != nil {
			_ = s.berkas.Close()
			s.berkas = nil
		}
		if err := os.MkdirAll(filepath.Dir(jalur), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(jalur, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return err
		}
		s.berkas = f
		s.nama = jalur
	}

	// Baris dari Engine sudah diakhiri newline; jangan digandakan.
	isi := entri.GetBaris()
	if isi[len(isi)-1] != '\n' {
		isi = append(isi, '\n')
	}
	if _, err := s.berkas.Write(isi); err != nil {
		return err
	}
	return nil
}

// Tutup menutup berkas yang sedang terbuka.
func (s *Server) Tutup() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.berkas == nil {
		return nil
	}
	err := s.berkas.Close()
	s.berkas = nil
	return err
}

// namaLayananSah membatasi nama berkas log agar karakter pemisah jalur atau
// ".." tidak pernah bisa keluar dari direktori log.
func namaLayananSah(s string) bool {
	if len(s) == 0 || len(s) > 32 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		alfanumerik := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !alfanumerik && c != '-' && c != '_' && c != '.' {
			return false
		}
	}
	// Tolak ".." maupun awalan/akhiran titik agar nama berkas selalu bersih.
	if s[0] == '.' || s[len(s)-1] == '.' || strings.Contains(s, "..") {
		return false
	}
	return true
}
