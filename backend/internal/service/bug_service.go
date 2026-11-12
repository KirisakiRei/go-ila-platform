package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"ila/backend/internal/audit"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// Batas lampiran. Nilai ini HARUS sama dengan `request_body max_size 1MB` di
// deploy/Caddyfile: berkas lebih besar sudah ditolak Caddy sebelum sampai sini,
// dan bila angka di sini lebih besar, pengguna hanya melihat galat proxy yang
// membingungkan.
const (
	BugMaxLampiranByte = 1 << 20 // 1 MiB
	BugMaxJudul        = 120
	BugMaxIsi          = 5000
)

// ErrLampiranTerlaluBesar dipakai agar handler bisa memberi pesan yang jelas,
// bukan 500 dari pembacaan badan permintaan.
var ErrLampiranTerlaluBesar = errors.Join(
	apperror.ErrBadRequest,
	fmt.Errorf("lampiran maksimal %d KB", BugMaxLampiranByte/1024),
)

// BugStore adalah kebutuhan penyimpanan laporan.
type BugStore interface {
	Create(ctx context.Context, report *model.BugReport) error
	AddAttachment(ctx context.Context, att *model.BugReportAttachment) error
	FindByID(ctx context.Context, id string) (*model.BugReport, error)
	ListByUser(ctx context.Context, userID string, limit int) ([]model.BugReport, error)
	List(ctx context.Context, status, category string, limit int) ([]model.BugReport, error)
	AttachmentsOf(ctx context.Context, reportID string) ([]model.BugReportAttachment, error)
	FindAttachment(ctx context.Context, reportID, attachmentID string) (*model.BugReportAttachment, error)
	UpdateStatus(ctx context.Context, id, status, adminNote string) error
}

// BugService adalah lapor bug dari sisi pengguna.
type BugService interface {
	Create(ctx context.Context, userID string, in BugInput) (*BugDetail, error)
	ListMine(ctx context.Context, userID string) ([]BugItem, error)
	// OpenAttachment memeriksa hak akses lalu mengembalikan berkas yang siap dibaca.
	OpenAttachment(ctx context.Context, actor BugActor, reportID, attachmentID string) (*BugFile, error)
}

// AdminBugService adalah triage laporan di panel admin.
type AdminBugService interface {
	List(ctx context.Context, status, category string) ([]BugDetail, error)
	UpdateStatus(ctx context.Context, actor AdminActor, reportID, status, note string) error
}

// BugActor adalah peminta berkas lampiran.
type BugActor struct {
	UserID string
	// Admin adalah pemilik sesi admin, bukan sesi pengguna.
	Admin bool
}

// BugInput adalah isi satu laporan yang sudah dibaca dari formulir.
type BugInput struct {
	Category  string
	Title     string
	Body      string
	PageURL   string
	UserAgent string
	// Lampiran opsional.
	Attachment *BugUpload
}

// BugUpload adalah berkas yang diunggah pengguna.
type BugUpload struct {
	FileName string
	// Isi dibaca lewat pembaca, bukan seluruhnya ke memori: batas memori
	// backend-api hanya 160 MB dan setiap unggahan tidak boleh menahannya.
	Reader io.Reader
}

// BugFile adalah lampiran yang siap disajikan handler.
type BugFile struct {
	Path     string
	FileName string
	MimeType string
}

// tipeDiizinkan memetakan awalan magic bytes ke tipe MIME yang disajikan.
//
// Tipe ditentukan dari isi berkas, BUKAN dari Content-Type kiriman browser:
// kiriman browser mudah dipalsukan, dan SVG yang lolos akan menjadi jalur XSS.
var tipeDiizinkan = []struct {
	magic []byte
	mime  string
	ext   string
	nama  string
}{
	{[]byte("\x89PNG\r\n\x1a\n"), "image/png", ".png", "png"},
	{[]byte("\xff\xd8\xff"), "image/jpeg", ".jpg", "jpeg"},
	{[]byte("GIF87a"), "image/gif", ".gif", "gif"},
	{[]byte("GIF89a"), "image/gif", ".gif", "gif"},
	// PDF diawali "%PDF-" (boleh ada byte sebelum header menurut spec, tetapi
	// praktiknya semua penulis PDF menaruhnya di awal).
	{[]byte("%PDF-"), "application/pdf", ".pdf", "pdf"},
}

// deteksiTipe mengembalikan tipe yang dikenal dari magic bytes.
// WEBP perlu pemeriksaan tambahan karena bentuknya RIFF....WEBP.
func deteksiTipe(head []byte) (mime, ext string, ok bool) {
	if len(head) >= 12 &&
		bytes.Equal(head[0:4], []byte("RIFF")) &&
		bytes.Equal(head[8:12], []byte("WEBP")) {
		return "image/webp", ".webp", true
	}
	for _, t := range tipeDiizinkan {
		if bytes.HasPrefix(head, t.magic) {
			return t.mime, t.ext, true
		}
	}
	return "", "", false
}

type bugService struct {
	store BugStore
	dir   string
	audit AdminAudit
	clock func() time.Time
}

// NewBugService merangkai layanan laporan bug.
//
// dir adalah akar penyimpanan lampiran. Nilai kosong membuat unggahan ditolak,
// supaya salah konfigurasi tidak diam-diam menyimpan berkas di folder acak.
func NewBugService(store BugStore, dir string, recorder AdminAudit) *bugService {
	return &bugService{
		store: store,
		dir:   dir,
		audit: recorder,
		clock: func() time.Time { return time.Now().UTC() },
	}
}

// Create menyimpan laporan beserta lampirannya.
func (s *bugService) Create(ctx context.Context, userID string, in BugInput) (*BugDetail, error) {
	category := strings.TrimSpace(in.Category)
	if !kategoriSah(category) {
		return nil, validation("kategori laporan tidak dikenal")
	}
	title := strings.TrimSpace(in.Title)
	if len([]rune(title)) < 3 {
		return nil, validation("judul laporan minimal 3 karakter")
	}
	if len([]rune(title)) > BugMaxJudul {
		return nil, validation("judul laporan maksimal 120 karakter")
	}
	body := strings.TrimSpace(in.Body)
	if len([]rune(body)) < 10 {
		return nil, validation("ceritakan bugnya minimal 10 karakter")
	}
	if len([]rune(body)) > BugMaxIsi {
		return nil, validation("isi laporan maksimal 5000 karakter")
	}

	report := &model.BugReport{
		UserID:    userID,
		Category:  category,
		Title:     title,
		Body:      body,
		PageURL:   potong(in.PageURL, 255),
		UserAgent: potong(in.UserAgent, 255),
		Status:    model.BugStatusBaru,
	}
	if err := s.store.Create(ctx, report); err != nil {
		return nil, err
	}

	// Lampiran disimpan setelah baris induk ada: bila berkas gagal ditulis,
	// laporan tetap tersimpan dan pengguna tidak kehilangan tulisannya.
	attachments := make([]model.BugReportAttachment, 0, 1)
	if in.Attachment != nil {
		att, err := s.simpanLampiran(ctx, report.ID, in.Attachment)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, *att)
	}

	return toBugDetail(*report, attachments), nil
}

// simpanLampiran memvalidasi isi berkas lalu menuliskannya ke disk.
//
// Urutannya penting: validasi tipe dilakukan dari magic bytes sebelum satu byte
// pun ditulis, dan nama simpan adalah UUID sehingga nama kiriman pengguna tidak
// pernah menyentuh jalur berkas (mencegah path traversal).
func (s *bugService) simpanLampiran(ctx context.Context, reportID string, up *BugUpload) (*model.BugReportAttachment, error) {
	if strings.TrimSpace(s.dir) == "" {
		return nil, errors.Join(apperror.ErrServiceUnavailable, errors.New("penyimpanan lampiran belum dikonfigurasi"))
	}

	// Baca kepala berkas untuk mengenali tipe, lalu sambungkan kembali.
	head := make([]byte, 16)
	n, err := io.ReadFull(up.Reader, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return nil, err
	}
	head = head[:n]

	mime, ext, ok := deteksiTipe(head)
	if !ok {
		return nil, validation("lampiran harus berupa gambar (png, jpg, webp, gif) atau PDF")
	}

	// Batasi total isi: kepala yang sudah dibaca ikut dihitung.
	terbatas := io.MultiReader(bytes.NewReader(head), io.LimitReader(up.Reader, BugMaxLampiranByte-int64(len(head))+1))

	folder := filepath.Join(s.dir, reportID)
	if err := os.MkdirAll(folder, 0o750); err != nil {
		return nil, fmt.Errorf("menyiapkan folder lampiran: %w", err)
	}

	storedAs := uuid.NewString() + ext
	tujuan := filepath.Join(folder, storedAs)
	berkas, err := os.OpenFile(tujuan, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return nil, fmt.Errorf("membuat berkas lampiran: %w", err)
	}

	written, copyErr := io.Copy(berkas, terbatas)
	closeErr := berkas.Close()

	if copyErr != nil || closeErr != nil {
		_ = os.Remove(tujuan)
		if copyErr != nil {
			return nil, fmt.Errorf("menulis lampiran: %w", copyErr)
		}
		return nil, fmt.Errorf("menutup lampiran: %w", closeErr)
	}

	// Melebihi batas: berkas dibuang supaya tidak ada sisa setengah jadi.
	if written > BugMaxLampiranByte {
		_ = os.Remove(tujuan)
		return nil, ErrLampiranTerlaluBesar
	}
	if written == 0 {
		_ = os.Remove(tujuan)
		return nil, validation("lampiran kosong")
	}

	att := &model.BugReportAttachment{
		ReportID:  reportID,
		FileName:  potong(sanitasiNamaBerkas(up.FileName, ext), 160),
		MimeType:  mime,
		SizeBytes: written,
		StoredAs:  storedAs,
	}
	if err := s.store.AddAttachment(ctx, att); err != nil {
		// Baris gagal disimpan: berkasnya tidak ada gunanya, hapus.
		_ = os.Remove(tujuan)
		return nil, err
	}
	return att, nil
}

// ListMine membaca laporan milik satu pengguna.
func (s *bugService) ListMine(ctx context.Context, userID string) ([]BugItem, error) {
	rows, err := s.store.ListByUser(ctx, userID, 50)
	if err != nil {
		return nil, err
	}
	out := make([]BugItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, toBugItem(row))
	}
	return out, nil
}

// OpenAttachment memeriksa hak akses lalu membuka berkas lampiran.
func (s *bugService) OpenAttachment(ctx context.Context, actor BugActor, reportID, attachmentID string) (*BugFile, error) {
	report, err := s.store.FindByID(ctx, reportID)
	if err != nil {
		return nil, err
	}

	// Pemilik laporan atau admin. Selain itu dianggap tidak ada, supaya
	// keberadaan laporan orang lain tidak bisa ditebak dari beda pesan.
	if !actor.Admin && report.UserID != actor.UserID {
		return nil, apperror.ErrNotFound
	}

	att, err := s.store.FindAttachment(ctx, reportID, attachmentID)
	if err != nil {
		return nil, err
	}

	// Path dibentuk dari nilai yang kita simpan sendiri (UUID), tetapi tetap
	// dipastikan berada di dalam folder laporan: pertahanan berlapis.
	path := filepath.Join(s.dir, reportID, att.StoredAs)
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(filepath.Join(s.dir, reportID))) {
		return nil, apperror.ErrNotFound
	}
	if _, err := os.Stat(path); err != nil {
		return nil, apperror.ErrNotFound
	}

	return &BugFile{Path: path, FileName: att.FileName, MimeType: att.MimeType}, nil
}

// ---------------------------------------------------------------------------
// Admin
// ---------------------------------------------------------------------------

// List membaca laporan untuk triage, beserta lampirannya.
func (s *bugService) List(ctx context.Context, status, category string) ([]BugDetail, error) {
	rows, err := s.store.List(ctx, status, category, 100)
	if err != nil {
		return nil, err
	}
	out := make([]BugDetail, 0, len(rows))
	for _, row := range rows {
		atts, err := s.store.AttachmentsOf(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, *toBugDetail(row, atts))
	}
	return out, nil
}

// UpdateStatus mengubah status triage dan catatan admin.
func (s *bugService) UpdateStatus(ctx context.Context, actor AdminActor, reportID, status, note string) error {
	if !statusSah(status) {
		return validation("status laporan tidak dikenal")
	}
	if err := s.store.UpdateStatus(ctx, reportID, status, strings.TrimSpace(note)); err != nil {
		return err
	}
	if s.audit != nil {
		if err := s.audit.Record(ctx, AuditEntry{
			ActorID: actor.ID, Action: audit.BugReportTriage,
			TargetType: "bug_report", TargetID: reportID,
			RequestID: actor.RequestID,
			After:     map[string]any{"status": status},
		}); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Pembantu
// ---------------------------------------------------------------------------

// sanitasiNamaBerkas membuang jalur dan karakter kendali dari nama kiriman.
//
// Nilai ini hanya untuk tampilan dan unduhan; berkas selalu disimpan dengan
// UUID. Tetap dibersihkan supaya header Content-Disposition tidak rusak.
func sanitasiNamaBerkas(nama, ext string) string {
	nama = filepath.Base(strings.ReplaceAll(nama, "\\", "/"))
	nama = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '"' || r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, nama)
	nama = strings.TrimSpace(nama)
	if nama == "" || nama == "." || nama == ".." {
		return "lampiran" + ext
	}
	return nama
}

// potong membatasi panjang teks menurut jumlah karakter.
func potong(s string, maks int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= maks {
		return string(r)
	}
	return string(r[:maks])
}

func kategoriSah(k string) bool {
	for _, item := range model.BugKategori {
		if item == k {
			return true
		}
	}
	return false
}

func statusSah(s string) bool {
	for _, item := range model.BugStatusSah {
		if item == s {
			return true
		}
	}
	return false
}

// ContentDisposition membentuk header unduhan yang aman.
//
// `attachment` wajib: tanpa itu, berkas bisa dibuka langsung di peramban.
// `nosniff` dipasang handler agar tipe tidak ditebak peramban.
func ContentDisposition(fileName string) string {
	aman := sanitasiNamaBerkas(fileName, "")
	aman = strings.ReplaceAll(aman, `\`, "_")
	return `attachment; filename="` + aman + `"`
}

// pastikan perilaku tipe sesuai yang disajikan.
var _ = http.DetectContentType
